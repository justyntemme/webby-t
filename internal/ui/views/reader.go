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
	loading         bool
	err             error
	showTOC         bool
	tocCursor       int
	textScale       float64 // Current text scale (affects line width)
	pendingPosition float64 // Position to restore after chapter loads (0-1)
	hasPendingPos   bool    // Whether there's a pending position to restore

	// Bookmarks
	showBookmarks   bool
	bookmarkCursor  int
	bookmarkMsg     string // Temporary status message for bookmarks

	// Search
	searchMode    bool          // Whether we're in search input mode
	searchQuery   string        // Current search query
	searchMatches []searchMatch // All matches in current chapter
	currentMatch  int           // Index of current highlighted match (-1 if none)
	searchActive  bool          // Whether search results are being displayed

	// Dimensions
	width  int
	height int
}

// searchMatch represents a single search match location
type searchMatch struct {
	lineIndex   int // Line number in wrapped content
	startOffset int // Character offset within the line
	endOffset   int // End character offset (exclusive)
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
	v.pendingPosition = 0
	v.hasPendingPos = false
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
		// Clear bookmark message on any key
		v.bookmarkMsg = ""

		// TOC mode
		if v.showTOC {
			return v.updateTOC(msg)
		}

		// Bookmarks mode
		if v.showBookmarks {
			return v.updateBookmarks(msg)
		}

		// Search input mode
		if v.searchMode {
			return v.updateSearchInput(msg)
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
		case "n":
			// Next search match (if search active) or next chapter
			if v.searchActive && len(v.searchMatches) > 0 {
				v.nextMatch()
			} else if v.chapter < len(v.chapters)-1 {
				return v, v.goToChapter(v.chapter + 1)
			}
		case "l":
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
		case "B":
			// Add bookmark at current position
			v.addBookmark()
		case "b":
			// Show bookmarks list
			v.showBookmarks = true
			v.bookmarkCursor = 0
		case "/":
			// Enter search mode
			v.searchMode = true
			v.searchQuery = ""
		case "N":
			// Previous search match (if search is active)
			if v.searchActive && len(v.searchMatches) > 0 {
				v.prevMatch()
			}
		case "esc":
			// Clear search if active
			if v.searchActive {
				v.clearSearch()
			}
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
				// Store position to restore after chapter loads
				v.pendingPosition = msg.position.Position
				v.hasPendingPos = true
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
		// Restore saved position if available
		if v.hasPendingPos && len(v.lines) > 0 {
			v.lineOffset = int(v.pendingPosition * float64(len(v.lines)))
			// Clamp to valid range
			maxOffset := len(v.lines) - v.visibleLines()
			if maxOffset < 0 {
				maxOffset = 0
			}
			if v.lineOffset > maxOffset {
				v.lineOffset = maxOffset
			}
			if v.lineOffset < 0 {
				v.lineOffset = 0
			}
			v.hasPendingPos = false
		}
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

	if v.showBookmarks {
		return v.renderBookmarks()
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
		line := v.lines[i]
		// Apply search highlighting if search is active
		if v.searchActive && len(v.searchMatches) > 0 {
			line = v.highlightLine(i, line)
		}
		b.WriteString(styles.ReaderContent.Render(line) + "\n")
	}

	// Footer or search input
	b.WriteString("\n")
	if v.searchMode {
		b.WriteString(v.renderSearchInput())
	} else {
		b.WriteString(v.renderFooter())
	}

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
	if len(title) > v.width/3 {
		title = title[:v.width/3-3] + "..."
	}
	titlePart := styles.ReaderHeader.Render(" " + title + " ")

	// Chapter info
	chapterTitle := ""
	if len(v.chapters) > v.chapter && v.chapter >= 0 {
		chapterTitle = v.chapters[v.chapter].Title
		if len(chapterTitle) > 20 {
			chapterTitle = chapterTitle[:17] + "..."
		}
	}
	chapterPart := styles.Help.Render(fmt.Sprintf(" Ch %d/%d: %s ", v.chapter+1, len(v.chapters), chapterTitle))

	// Chapter progress (within current chapter)
	chapterProgress := v.calculateProgress()

	// Book progress (based on chapters completed + current chapter progress)
	bookProgress := v.calculateBookProgress()

	// Progress bars - use compact format
	barWidth := 10
	chapterBar := renderProgressBar(barWidth, float64(chapterProgress)/100.0)
	bookBar := renderProgressBar(barWidth, float64(bookProgress)/100.0)

	progressPart := styles.MutedText.Render("Ch:") + chapterBar +
		styles.MutedText.Render(" Book:") + bookBar +
		styles.ReaderProgress.Render(fmt.Sprintf(" %d%%", bookProgress))

	// Combine
	left := titlePart + chapterPart
	right := progressPart

	gap := v.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}

// calculateBookProgress returns overall book progress as percentage
func (v *ReaderView) calculateBookProgress() int {
	if len(v.chapters) == 0 {
		return 0
	}
	// Weight each chapter equally, add current chapter's progress
	chapterWeight := 100.0 / float64(len(v.chapters))
	completedChapters := float64(v.chapter) * chapterWeight
	currentChapterProgress := float64(v.calculateProgress()) / 100.0 * chapterWeight
	return int(completedChapters + currentChapterProgress)
}

// renderProgressBar renders a visual progress bar using Unicode block characters
// width is the total character width, progress is 0.0-1.0
func renderProgressBar(width int, progress float64) string {
	if width < 3 {
		width = 3
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}

	// Unicode block characters for smooth rendering
	const (
		empty    = "░"
		filled   = "█"
		partials = "▏▎▍▌▋▊▉" // 1/8 to 7/8 filled
	)

	// Calculate filled portion
	filledWidth := progress * float64(width)
	fullBlocks := int(filledWidth)
	remainder := filledWidth - float64(fullBlocks)

	var bar strings.Builder

	// Full blocks
	for i := 0; i < fullBlocks && i < width; i++ {
		bar.WriteString(filled)
	}

	// Partial block (if there's room and remainder)
	if fullBlocks < width && remainder > 0 {
		partialIndex := int(remainder * 8)
		if partialIndex > 7 {
			partialIndex = 7
		}
		if partialIndex > 0 {
			// Get the partial character
			runes := []rune(partials)
			bar.WriteRune(runes[partialIndex-1])
			fullBlocks++
		}
	}

	// Empty blocks
	for i := fullBlocks; i < width; i++ {
		bar.WriteString(empty)
	}

	return bar.String()
}

// renderFooter renders the reader footer
func (v *ReaderView) renderFooter() string {
	// Text scale indicator
	scaleStr := fmt.Sprintf("%.0f%%", v.textScale*100)

	// Show bookmark message if set
	if v.bookmarkMsg != "" {
		return styles.SecondaryText.Render(v.bookmarkMsg)
	}

	// Show search status if search is active
	if v.searchActive {
		searchStatus := fmt.Sprintf("/%s", v.searchQuery)
		matchInfo := ""
		if len(v.searchMatches) == 0 {
			matchInfo = styles.ErrorStyle.Render(" [No matches]")
		} else {
			matchInfo = styles.SecondaryText.Render(fmt.Sprintf(" [%d/%d]", v.currentMatch+1, len(v.searchMatches)))
		}
		help := []string{
			styles.HelpKey.Render("n/N") + styles.Help.Render(" next/prev"),
			styles.HelpKey.Render("esc") + styles.Help.Render(" clear"),
		}
		return styles.BookAuthor.Render(searchStatus) + matchInfo + "  " + strings.Join(help, "  ")
	}

	help := []string{
		styles.HelpKey.Render("j/k") + styles.Help.Render(" scroll"),
		styles.HelpKey.Render("n/p") + styles.Help.Render(" chapter"),
		styles.HelpKey.Render("t") + styles.Help.Render(" toc"),
		styles.HelpKey.Render("/") + styles.Help.Render(" search"),
		styles.HelpKey.Render("b/B") + styles.Help.Render(" marks"),
		styles.HelpKey.Render("+/-") + styles.Help.Render(" size:"+scaleStr),
		styles.HelpKey.Render("q") + styles.Help.Render(" back"),
	}
	return strings.Join(help, "  ")
}

// renderSearchInput renders the search input bar
func (v *ReaderView) renderSearchInput() string {
	cursor := "_"
	return styles.HelpKey.Render("/") + styles.BookAuthor.Render(v.searchQuery+cursor) + "  " + styles.Help.Render("enter search • esc cancel")
}

// highlightLine applies search highlighting to a line
func (v *ReaderView) highlightLine(lineIdx int, line string) string {
	// Find all matches on this line
	var lineMatches []searchMatch
	for i, m := range v.searchMatches {
		if m.lineIndex == lineIdx {
			m.lineIndex = i // Store the match index for current match detection
			lineMatches = append(lineMatches, v.searchMatches[i])
		}
	}

	if len(lineMatches) == 0 {
		return line
	}

	// Build highlighted line
	var result strings.Builder
	lastEnd := 0

	for _, m := range lineMatches {
		// Add text before match
		if m.startOffset > lastEnd {
			result.WriteString(line[lastEnd:m.startOffset])
		}

		// Determine if this is the current match
		isCurrentMatch := false
		for i, sm := range v.searchMatches {
			if sm.lineIndex == lineIdx && sm.startOffset == m.startOffset && i == v.currentMatch {
				isCurrentMatch = true
				break
			}
		}

		// Add highlighted match
		matchText := line[m.startOffset:m.endOffset]
		if isCurrentMatch {
			// Current match - more prominent highlight
			result.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("3")).Foreground(lipgloss.Color("0")).Bold(true).Render(matchText))
		} else {
			// Other matches - subtle highlight
			result.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("8")).Foreground(lipgloss.Color("15")).Render(matchText))
		}
		lastEnd = m.endOffset
	}

	// Add remaining text
	if lastEnd < len(line) {
		result.WriteString(line[lastEnd:])
	}

	return result.String()
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

// addBookmark adds a bookmark at the current position
func (v *ReaderView) addBookmark() {
	if v.book == nil || v.config == nil {
		return
	}
	chapterTitle := ""
	if len(v.chapters) > v.chapter && v.chapter >= 0 {
		chapterTitle = v.chapters[v.chapter].Title
	}
	position := float64(v.lineOffset) / float64(max(1, len(v.lines)))
	err := v.config.AddBookmark(v.book.ID, v.book.Title, v.chapter, chapterTitle, position, "")
	if err != nil {
		v.bookmarkMsg = "Failed to add bookmark"
	} else {
		v.bookmarkMsg = "Bookmark added"
	}
}

// updateBookmarks handles bookmarks list navigation
func (v *ReaderView) updateBookmarks(msg tea.KeyMsg) (View, tea.Cmd) {
	bookmarks := v.getBookmarksForCurrentBook()

	switch msg.String() {
	case "esc", "b", "q":
		v.showBookmarks = false
	case "j", "down":
		if v.bookmarkCursor < len(bookmarks)-1 {
			v.bookmarkCursor++
		}
	case "k", "up":
		if v.bookmarkCursor > 0 {
			v.bookmarkCursor--
		}
	case "g", "home":
		v.bookmarkCursor = 0
	case "G", "end":
		if len(bookmarks) > 0 {
			v.bookmarkCursor = len(bookmarks) - 1
		}
	case "enter":
		// Navigate to selected bookmark
		if v.bookmarkCursor < len(bookmarks) {
			v.showBookmarks = false
			return v, v.goToBookmark(bookmarks[v.bookmarkCursor])
		}
	case "d", "x":
		// Delete selected bookmark
		if v.bookmarkCursor < len(bookmarks) && v.config != nil {
			_ = v.config.DeleteBookmark(bookmarks[v.bookmarkCursor].ID)
			// Adjust cursor if needed
			if v.bookmarkCursor >= len(bookmarks)-1 && v.bookmarkCursor > 0 {
				v.bookmarkCursor--
			}
		}
	}
	return v, nil
}

// getBookmarksForCurrentBook returns bookmarks for the current book
func (v *ReaderView) getBookmarksForCurrentBook() []config.Bookmark {
	if v.book == nil || v.config == nil {
		return nil
	}
	return v.config.GetBookmarksForBook(v.book.ID)
}

// goToBookmark navigates to a bookmark
func (v *ReaderView) goToBookmark(bookmark config.Bookmark) tea.Cmd {
	// Store position to restore after chapter loads
	v.pendingPosition = bookmark.Position
	v.hasPendingPos = true
	return v.loadChapter(bookmark.Chapter)
}

// renderBookmarks renders the bookmarks overlay
func (v *ReaderView) renderBookmarks() string {
	var b strings.Builder

	b.WriteString(styles.DialogTitle.Render("Bookmarks") + "\n\n")

	bookmarks := v.getBookmarksForCurrentBook()

	if len(bookmarks) == 0 {
		b.WriteString(styles.MutedText.Render("No bookmarks for this book.\n\nPress B to add a bookmark."))
	} else {
		// Calculate visible range
		maxVisible := v.height - 10
		offset := 0
		if v.bookmarkCursor >= maxVisible {
			offset = v.bookmarkCursor - maxVisible + 1
		}

		for i := offset; i < min(offset+maxVisible, len(bookmarks)); i++ {
			bm := bookmarks[i]
			chapterLabel := fmt.Sprintf("Ch %d", bm.Chapter+1)
			if bm.ChapterTitle != "" {
				title := bm.ChapterTitle
				if len(title) > 20 {
					title = title[:17] + "..."
				}
				chapterLabel = fmt.Sprintf("Ch %d: %s", bm.Chapter+1, title)
			}
			progress := fmt.Sprintf("%.0f%%", bm.Position*100)
			line := fmt.Sprintf("%s [%s]", chapterLabel, progress)

			if i == v.bookmarkCursor {
				b.WriteString(styles.ListItemSelected.Render("▸ "+line) + "\n")
			} else {
				b.WriteString(styles.ListItem.Render("  "+line) + "\n")
			}
		}
	}

	b.WriteString("\n" + styles.Help.Render("j/k navigate • enter go • d delete • esc close"))

	dialog := styles.Dialog.Width(min(50, v.width-4)).Render(b.String())

	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// updateSearchInput handles keyboard input during search mode
func (v *ReaderView) updateSearchInput(msg tea.KeyMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel search input
		v.searchMode = false
		v.searchQuery = ""
	case "enter":
		// Execute search
		v.searchMode = false
		if v.searchQuery != "" {
			v.executeSearch()
		}
	case "backspace":
		// Delete last character
		if len(v.searchQuery) > 0 {
			v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
		}
	case "ctrl+u":
		// Clear search query
		v.searchQuery = ""
	default:
		// Add character to search query (filter control characters)
		if len(msg.String()) == 1 && msg.String()[0] >= 32 {
			v.searchQuery += msg.String()
		} else if msg.Type == tea.KeyRunes {
			v.searchQuery += string(msg.Runes)
		}
	}
	return v, nil
}

// executeSearch finds all matches in current chapter content
func (v *ReaderView) executeSearch() {
	v.searchMatches = nil
	v.currentMatch = -1
	v.searchActive = false

	if v.searchQuery == "" || len(v.lines) == 0 {
		return
	}

	query := strings.ToLower(v.searchQuery)

	// Search through all wrapped lines
	for lineIdx, line := range v.lines {
		lineLower := strings.ToLower(line)
		offset := 0
		for {
			idx := strings.Index(lineLower[offset:], query)
			if idx == -1 {
				break
			}
			match := searchMatch{
				lineIndex:   lineIdx,
				startOffset: offset + idx,
				endOffset:   offset + idx + len(v.searchQuery),
			}
			v.searchMatches = append(v.searchMatches, match)
			offset += idx + 1
		}
	}

	if len(v.searchMatches) > 0 {
		v.searchActive = true
		v.currentMatch = 0
		v.scrollToMatch(0)
	}
}

// nextMatch moves to the next search match
func (v *ReaderView) nextMatch() {
	if len(v.searchMatches) == 0 {
		return
	}
	v.currentMatch = (v.currentMatch + 1) % len(v.searchMatches)
	v.scrollToMatch(v.currentMatch)
}

// prevMatch moves to the previous search match
func (v *ReaderView) prevMatch() {
	if len(v.searchMatches) == 0 {
		return
	}
	v.currentMatch--
	if v.currentMatch < 0 {
		v.currentMatch = len(v.searchMatches) - 1
	}
	v.scrollToMatch(v.currentMatch)
}

// scrollToMatch scrolls to make the given match visible
func (v *ReaderView) scrollToMatch(matchIdx int) {
	if matchIdx < 0 || matchIdx >= len(v.searchMatches) {
		return
	}
	match := v.searchMatches[matchIdx]
	visibleLines := v.visibleLines()

	// If match is above visible area, scroll up
	if match.lineIndex < v.lineOffset {
		v.lineOffset = match.lineIndex
	}
	// If match is below visible area, scroll down
	if match.lineIndex >= v.lineOffset+visibleLines {
		v.lineOffset = match.lineIndex - visibleLines + 1
	}
}

// clearSearch clears search state
func (v *ReaderView) clearSearch() {
	v.searchActive = false
	v.searchQuery = ""
	v.searchMatches = nil
	v.currentMatch = -1
}
