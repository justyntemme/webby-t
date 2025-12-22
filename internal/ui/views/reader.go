package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/config"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/pkg/models"
)

// ReaderView displays book content
type ReaderView struct {
	client *api.Client
	config *config.Config

	// Current book
	book     *models.Book
	chapters []models.Chapter
	chapter  int

	// Content
	content    string
	lines      []string
	lineOffset int

	// State
	loading   bool
	err       error
	showTOC   bool
	tocCursor int
	textScale float64 // Current text scale (affects line width)

	// Dimensions
	width  int
	height int
}

// NewReaderView creates a new reader view
func NewReaderView(client *api.Client, cfg *config.Config) *ReaderView {
	return &ReaderView{
		client:    client,
		config:    cfg,
		textScale: cfg.GetTextScale(),
		width:     80,
		height:    24,
	}
}

// SetBook sets the current book to read
func (v *ReaderView) SetBook(book models.Book) {
	v.book = &book
	v.chapter = 0
	v.lineOffset = 0
	v.chapters = nil
	v.content = ""
	v.lines = nil
	v.showTOC = false
}

// SavePositionOnExit saves the current position (called when leaving reader)
func (v *ReaderView) SavePositionOnExit() {
	v.savePosition()
}

// Message types
type tocLoadedMsg struct {
	chapters []models.Chapter
	err      error
}

type chapterLoadedMsg struct {
	content string
	chapter int
	err     error
}

type positionLoadedMsg struct {
	position *models.ReadingPosition
	err      error
}

// Init implements View
func (v *ReaderView) Init() tea.Cmd {
	if v.book == nil {
		return nil
	}
	v.loading = true
	// Load TOC, position, and first chapter
	return tea.Batch(
		v.loadTOC(),
		v.loadPosition(),
	)
}

// Update implements View
func (v *ReaderView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// TOC mode
		if v.showTOC {
			return v.updateTOC(msg)
		}

		// Reader mode
		switch msg.String() {
		case "j", "down":
			v.scroll(1)
		case "k", "up":
			v.scroll(-1)
		case "ctrl+d", "pgdown":
			v.scroll(v.visibleLines() / 2)
		case "ctrl+u", "pgup":
			v.scroll(-v.visibleLines() / 2)
		case "g", "home":
			v.lineOffset = 0
		case "G", "end":
			v.lineOffset = max(0, len(v.lines)-v.visibleLines())
		case "n", "l":
			// Next chapter
			if v.chapter < len(v.chapters)-1 {
				return v, v.goToChapter(v.chapter + 1)
			}
		case "p", "h":
			// Previous chapter
			if v.chapter > 0 {
				return v, v.goToChapter(v.chapter - 1)
			}
		case "t":
			// Toggle TOC
			v.showTOC = true
			v.tocCursor = v.chapter
		case " ":
			// Space for page down
			v.scroll(v.visibleLines() - 2)
		case "+", "=":
			// Increase text size (= is unshifted + on most keyboards)
			v.adjustTextScale(config.TextScaleStep)
		case "-", "_":
			// Decrease text size
			v.adjustTextScale(-config.TextScaleStep)
		case "0":
			// Reset text size to default
			v.setTextScale(config.DefaultTextScale)
		}

	case tocLoadedMsg:
		if msg.err != nil {
			v.err = msg.err
			v.loading = false
			return v, nil
		}
		v.chapters = msg.chapters
		// Load first chapter if we haven't loaded position yet
		if v.content == "" && len(v.chapters) > 0 {
			return v, v.loadChapter(v.chapter)
		}
		return v, nil

	case positionLoadedMsg:
		if msg.err == nil && msg.position != nil {
			// Parse chapter from position
			var chapterNum int
			fmt.Sscanf(msg.position.Chapter, "%d", &chapterNum)
			if chapterNum >= 0 && (len(v.chapters) == 0 || chapterNum < len(v.chapters)) {
				v.chapter = chapterNum
				// Calculate line offset from position percentage
				// Will be applied after content loads
			}
		}
		// Load the chapter
		return v, v.loadChapter(v.chapter)

	case chapterLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.content = msg.content
		v.chapter = msg.chapter
		v.wrapContent()
		v.err = nil
		return v, nil
	}

	return v, nil
}

// updateTOC handles TOC navigation
func (v *ReaderView) updateTOC(msg tea.KeyMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc", "t", "q":
		v.showTOC = false
	case "j", "down":
		if v.tocCursor < len(v.chapters)-1 {
			v.tocCursor++
		}
	case "k", "up":
		if v.tocCursor > 0 {
			v.tocCursor--
		}
	case "g", "home":
		v.tocCursor = 0
	case "G", "end":
		v.tocCursor = len(v.chapters) - 1
	case "enter":
		v.showTOC = false
		return v, v.goToChapter(v.tocCursor)
	}
	return v, nil
}

// View implements View
func (v *ReaderView) View() string {
	if v.book == nil {
		return styles.ErrorStyle.Render("No book selected")
	}

	if v.showTOC {
		return v.renderTOC()
	}

	var b strings.Builder

	// Header
	b.WriteString(v.renderHeader() + "\n")

	// Loading state
	if v.loading {
		content := lipgloss.Place(
			v.width,
			v.height-4,
			lipgloss.Center,
			lipgloss.Center,
			styles.MutedText.Render("Loading..."),
		)
		b.WriteString(content)
		return b.String()
	}

	// Error state
	if v.err != nil {
		content := lipgloss.Place(
			v.width,
			v.height-4,
			lipgloss.Center,
			lipgloss.Center,
			styles.ErrorStyle.Render("Error: "+v.err.Error()),
		)
		b.WriteString(content)
		return b.String()
	}

	// Content
	visibleLines := v.visibleLines()
	for i := v.lineOffset; i < min(v.lineOffset+visibleLines, len(v.lines)); i++ {
		b.WriteString(styles.ReaderContent.Render(v.lines[i]) + "\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(v.renderFooter())

	return b.String()
}

// SetSize implements View
func (v *ReaderView) SetSize(width, height int) {
	v.width = width
	v.height = height
	if v.content != "" {
		v.wrapContent()
	}
}

// renderHeader renders the reader header
func (v *ReaderView) renderHeader() string {
	// Book title
	title := v.book.Title
	if len(title) > v.width/2 {
		title = title[:v.width/2-3] + "..."
	}
	titlePart := styles.ReaderHeader.Render(" " + title + " ")

	// Chapter info
	chapterTitle := ""
	if len(v.chapters) > v.chapter && v.chapter >= 0 {
		chapterTitle = v.chapters[v.chapter].Title
		if len(chapterTitle) > 30 {
			chapterTitle = chapterTitle[:27] + "..."
		}
	}
	chapterPart := styles.Help.Render(fmt.Sprintf(" Ch %d/%d: %s ", v.chapter+1, len(v.chapters), chapterTitle))

	// Progress
	progress := v.calculateProgress()
	progressPart := styles.ReaderProgress.Render(fmt.Sprintf(" %d%% ", progress))

	// Combine
	left := titlePart + chapterPart
	right := progressPart

	gap := v.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}

// renderFooter renders the reader footer
func (v *ReaderView) renderFooter() string {
	// Text scale indicator
	scaleStr := fmt.Sprintf("%.0f%%", v.textScale*100)

	help := []string{
		styles.HelpKey.Render("j/k") + styles.Help.Render(" scroll"),
		styles.HelpKey.Render("n/p") + styles.Help.Render(" chapter"),
		styles.HelpKey.Render("t") + styles.Help.Render(" toc"),
		styles.HelpKey.Render("+/-") + styles.Help.Render(" size:"+scaleStr),
		styles.HelpKey.Render("q") + styles.Help.Render(" back"),
	}
	return strings.Join(help, "  ")
}

// renderTOC renders the table of contents overlay
func (v *ReaderView) renderTOC() string {
	var b strings.Builder

	b.WriteString(styles.DialogTitle.Render("Table of Contents") + "\n\n")

	// Calculate visible range
	maxVisible := v.height - 8
	offset := 0
	if v.tocCursor >= maxVisible {
		offset = v.tocCursor - maxVisible + 1
	}

	for i := offset; i < min(offset+maxVisible, len(v.chapters)); i++ {
		ch := v.chapters[i]
		line := fmt.Sprintf("%d. %s", i+1, ch.Title)
		if len(line) > v.width-10 {
			line = line[:v.width-13] + "..."
		}

		if i == v.tocCursor {
			b.WriteString(styles.ListItemSelected.Render("▸ "+line) + "\n")
		} else if i == v.chapter {
			b.WriteString(styles.BookAuthor.Render("  "+line+" (current)") + "\n")
		} else {
			b.WriteString(styles.ListItem.Render("  "+line) + "\n")
		}
	}

	b.WriteString("\n" + styles.Help.Render("j/k navigate • enter select • esc close"))

	dialog := styles.Dialog.Width(min(60, v.width-4)).Render(b.String())

	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// wrapContent wraps content to fit the terminal width
func (v *ReaderView) wrapContent() {
	v.lines = nil
	// Apply text scale to width: larger scale = narrower lines (simulates bigger text)
	// Scale of 1.0 = full width, 2.0 = half width, 0.5 = full width (capped)
	baseWidth := v.width - 4 // Account for padding
	scaledWidth := int(float64(baseWidth) / v.textScale)
	if scaledWidth < 20 {
		scaledWidth = 20 // Minimum readable width
	}
	if scaledWidth > baseWidth {
		scaledWidth = baseWidth
	}
	maxWidth := scaledWidth

	for _, paragraph := range strings.Split(v.content, "\n") {
		if paragraph == "" {
			v.lines = append(v.lines, "")
			continue
		}

		// Wrap long lines
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			v.lines = append(v.lines, "")
			continue
		}

		var currentLine strings.Builder
		for _, word := range words {
			if currentLine.Len() == 0 {
				currentLine.WriteString(word)
			} else if currentLine.Len()+1+len(word) <= maxWidth {
				currentLine.WriteString(" ")
				currentLine.WriteString(word)
			} else {
				v.lines = append(v.lines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			}
		}
		if currentLine.Len() > 0 {
			v.lines = append(v.lines, currentLine.String())
		}
	}
}

// scroll scrolls the content by delta lines
func (v *ReaderView) scroll(delta int) {
	v.lineOffset += delta
	if v.lineOffset < 0 {
		v.lineOffset = 0
	}
	maxOffset := len(v.lines) - v.visibleLines()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if v.lineOffset > maxOffset {
		v.lineOffset = maxOffset
	}
}

// visibleLines returns the number of visible content lines
func (v *ReaderView) visibleLines() int {
	lines := v.height - 5 // Header, footer, margins
	if lines < 1 {
		lines = 1
	}
	return lines
}

// calculateProgress returns reading progress as percentage
func (v *ReaderView) calculateProgress() int {
	if len(v.lines) == 0 {
		return 0
	}
	visible := v.visibleLines()
	if v.lineOffset+visible >= len(v.lines) {
		return 100
	}
	return (v.lineOffset * 100) / len(v.lines)
}

// loadTOC loads the table of contents
func (v *ReaderView) loadTOC() tea.Cmd {
	return func() tea.Msg {
		resp, err := v.client.GetTOC(v.book.ID)
		if err != nil {
			return tocLoadedMsg{err: err}
		}
		return tocLoadedMsg{chapters: resp.Chapters}
	}
}

// loadChapter loads a chapter's content
func (v *ReaderView) loadChapter(chapter int) tea.Cmd {
	v.loading = true
	return func() tea.Msg {
		content, err := v.client.GetChapterText(v.book.ID, chapter)
		if err != nil {
			return chapterLoadedMsg{err: err, chapter: chapter}
		}
		return chapterLoadedMsg{content: content.Content, chapter: chapter}
	}
}

// loadPosition loads saved reading position
func (v *ReaderView) loadPosition() tea.Cmd {
	return func() tea.Msg {
		pos, err := v.client.GetPosition(v.book.ID)
		return positionLoadedMsg{position: pos, err: err}
	}
}

// goToChapter navigates to a specific chapter
func (v *ReaderView) goToChapter(chapter int) tea.Cmd {
	v.lineOffset = 0
	// Save current position before leaving
	go v.savePosition()
	return v.loadChapter(chapter)
}

// savePosition saves the current reading position
func (v *ReaderView) savePosition() {
	if v.book == nil {
		return
	}
	position := float64(v.lineOffset) / float64(max(1, len(v.lines)))
	v.client.SavePosition(v.book.ID, fmt.Sprintf("%d", v.chapter), position)
}

// adjustTextScale changes text scale by delta
func (v *ReaderView) adjustTextScale(delta float64) {
	v.setTextScale(v.textScale + delta)
}

// setTextScale sets the text scale and rewraps content
func (v *ReaderView) setTextScale(scale float64) {
	if scale < config.MinTextScale {
		scale = config.MinTextScale
	}
	if scale > config.MaxTextScale {
		scale = config.MaxTextScale
	}
	if scale == v.textScale {
		return
	}
	v.textScale = scale
	// Save to config
	if v.config != nil {
		_ = v.config.SetTextScale(scale)
	}
	// Rewrap content with new scale
	if v.content != "" {
		v.wrapContent()
	}
}
