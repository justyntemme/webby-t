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

// BookDetailsView displays detailed book information
type BookDetailsView struct {
	client *api.Client
	config *config.Config

	// Book being displayed
	book *models.Book

	// Reading position (loaded async)
	position *models.ReadingPosition
	posErr   error

	// TOC for chapter count
	chapters []models.Chapter

	// Dimensions
	width  int
	height int
}

// NewBookDetailsView creates a new book details view
func NewBookDetailsView(client *api.Client, cfg *config.Config) *BookDetailsView {
	return &BookDetailsView{
		client: client,
		config: cfg,
		width:  80,
		height: 24,
	}
}

// SetBook sets the book to display
func (v *BookDetailsView) SetBook(book models.Book) {
	v.book = &book
	v.position = nil
	v.posErr = nil
	v.chapters = nil
}

// detailsPositionLoadedMsg is sent when reading position is loaded for book details
type detailsPositionLoadedMsg struct {
	position *models.ReadingPosition
	err      error
}

// detailsTOCLoadedMsg is sent when TOC is loaded for book details
type detailsTOCLoadedMsg struct {
	chapters []models.Chapter
	err      error
}

// Init implements View
func (v *BookDetailsView) Init() tea.Cmd {
	if v.book == nil {
		return nil
	}
	// Load reading position and TOC in parallel
	return tea.Batch(
		v.loadPosition(),
		v.loadTOC(),
	)
}

// Update implements View
func (v *BookDetailsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "i":
			// Go back to library
			return v, SwitchTo(ViewLibrary)
		case "enter":
			// Open the book for reading
			if v.book != nil {
				return v, func() tea.Msg {
					return OpenBookMsg{Book: *v.book}
				}
			}
		case "f":
			// Toggle favorite
			if v.book != nil && v.config != nil {
				_ = v.config.ToggleFavorite(v.book.ID)
			}
		case "w":
			// Toggle reading queue
			if v.book != nil && v.config != nil {
				_ = v.config.ToggleQueue(v.book.ID)
			}
		}

	case detailsPositionLoadedMsg:
		if msg.err == nil {
			v.position = msg.position
		}
		v.posErr = msg.err

	case detailsTOCLoadedMsg:
		if msg.err == nil {
			v.chapters = msg.chapters
		}
	}

	return v, nil
}

// View implements View
func (v *BookDetailsView) View() string {
	if v.book == nil {
		return "No book selected"
	}

	var b strings.Builder

	// Title section
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		MarginBottom(1)

	b.WriteString(titleStyle.Render(v.book.Title) + "\n\n")

	// Author
	if v.book.Author != "" {
		b.WriteString(v.renderField("Author", v.book.Author))
	}

	// Series
	if v.book.Series != "" {
		seriesText := v.book.Series
		if v.book.SeriesIndex > 0 {
			seriesText += fmt.Sprintf(" #%.0f", v.book.SeriesIndex)
		}
		b.WriteString(v.renderField("Series", seriesText))
	}

	// Content Type
	contentType := "Book"
	if v.book.IsComic() {
		contentType = "Comic"
	}
	b.WriteString(v.renderField("Type", contentType))

	// File Format
	if v.book.FileFormat != "" {
		b.WriteString(v.renderField("Format", strings.ToUpper(v.book.FileFormat)))
	}

	// File Size
	b.WriteString(v.renderField("Size", v.formatFileSize(v.book.FileSize)))

	// Upload Date
	uploadDate := v.book.UploadedAt.Format("January 2, 2006")
	b.WriteString(v.renderField("Uploaded", uploadDate))

	// Chapter count (if available)
	if len(v.chapters) > 0 {
		b.WriteString(v.renderField("Chapters", fmt.Sprintf("%d", len(v.chapters))))
	}

	b.WriteString("\n")

	// Reading Progress section
	b.WriteString(styles.HelpKey.Render("Reading Progress") + "\n")
	if v.position != nil {
		progressPercent := v.position.Position * 100
		b.WriteString(v.renderField("Chapter", v.position.Chapter))
		b.WriteString(v.renderField("Progress", fmt.Sprintf("%.1f%%", progressPercent)))
		b.WriteString(v.renderField("Last Read", v.position.UpdatedAt.Format("Jan 2, 2006 3:04 PM")))
	} else if v.posErr != nil {
		b.WriteString(styles.MutedText.Render("  Unable to load progress\n"))
	} else {
		b.WriteString(styles.MutedText.Render("  Not started\n"))
	}

	b.WriteString("\n")

	// Status indicators
	if v.config != nil {
		var statusItems []string
		if v.config.IsFavorite(v.book.ID) {
			favStyle := lipgloss.NewStyle().Foreground(styles.Warning)
			statusItems = append(statusItems, favStyle.Render("★ Favorited"))
		}
		if pos := v.config.GetQueuePosition(v.book.ID); pos > 0 {
			statusItems = append(statusItems, styles.SecondaryText.Render(fmt.Sprintf("Queue #%d", pos)))
		}
		if len(statusItems) > 0 {
			b.WriteString(strings.Join(statusItems, "  ") + "\n\n")
		}
	}

	// Footer
	footer := v.renderFooter()
	b.WriteString(footer)

	// Center the content
	content := lipgloss.NewStyle().
		Width(v.width - 4).
		Padding(1, 2).
		Render(b.String())

	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		styles.Dialog.Width(min(60, v.width-4)).Render(content),
	)
}

// renderField renders a label-value pair
func (v *BookDetailsView) renderField(label, value string) string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(12)
	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	return labelStyle.Render(label+":") + " " + valueStyle.Render(value) + "\n"
}

// renderFooter renders the footer help
func (v *BookDetailsView) renderFooter() string {
	help := []string{
		styles.HelpKey.Render("enter") + styles.Help.Render(" read"),
		styles.HelpKey.Render("f") + styles.Help.Render(" fav"),
		styles.HelpKey.Render("w") + styles.Help.Render(" queue"),
		styles.HelpKey.Render("esc/q") + styles.Help.Render(" back"),
	}
	return strings.Join(help, "  ")
}

// SetSize implements View
func (v *BookDetailsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// formatFileSize formats bytes to human readable size
func (v *BookDetailsView) formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// loadPosition loads the reading position for the book
func (v *BookDetailsView) loadPosition() tea.Cmd {
	return func() tea.Msg {
		if v.book == nil {
			return detailsPositionLoadedMsg{err: fmt.Errorf("no book")}
		}
		pos, err := v.client.GetPosition(v.book.ID)
		return detailsPositionLoadedMsg{position: pos, err: err}
	}
}

// loadTOC loads the table of contents for chapter count
func (v *BookDetailsView) loadTOC() tea.Cmd {
	return func() tea.Msg {
		if v.book == nil {
			return detailsTOCLoadedMsg{err: fmt.Errorf("no book")}
		}
		toc, err := v.client.GetTOC(v.book.ID)
		if err != nil {
			return detailsTOCLoadedMsg{err: err}
		}
		return detailsTOCLoadedMsg{chapters: toc.Chapters}
	}
}
