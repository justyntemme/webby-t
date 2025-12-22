package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/pkg/models"
)

// Sort options
type sortField int

const (
	sortTitle sortField = iota
	sortAuthor
	sortSeries
	sortDate
)

func (s sortField) String() string {
	switch s {
	case sortTitle:
		return "title"
	case sortAuthor:
		return "author"
	case sortSeries:
		return "series"
	case sortDate:
		return "uploaded_at"
	default:
		return "title"
	}
}

func (s sortField) Label() string {
	switch s {
	case sortTitle:
		return "Title"
	case sortAuthor:
		return "Author"
	case sortSeries:
		return "Series"
	case sortDate:
		return "Date"
	default:
		return "Title"
	}
}

// LibraryView displays the book library
type LibraryView struct {
	client *api.Client

	// Books
	books       []models.Book
	cursor      int
	offset      int // For scrolling

	// State
	loading     bool
	err         error
	searchMode  bool
	searchInput textinput.Model

	// Sorting
	sortBy    sortField
	sortAsc   bool

	// Pagination
	page      int
	pageSize  int
	total     int

	// Dimensions
	width  int
	height int
}

// NewLibraryView creates a new library view
func NewLibraryView(client *api.Client) *LibraryView {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search books..."
	searchInput.CharLimit = 100
	searchInput.Width = 40

	return &LibraryView{
		client:      client,
		pageSize:    50,
		page:        1,
		sortBy:      sortTitle,
		sortAsc:     true,
		searchInput: searchInput,
		width:       80,
		height:      24,
	}
}

// booksLoadedMsg is sent when books are loaded
type booksLoadedMsg struct {
	books []models.Book
	total int
	err   error
}

// Init implements View
func (v *LibraryView) Init() tea.Cmd {
	v.loading = true
	return v.loadBooks()
}

// Update implements View
func (v *LibraryView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode
		if v.searchMode {
			switch msg.String() {
			case "esc":
				v.searchMode = false
				v.searchInput.Blur()
				return v, nil
			case "enter":
				v.searchMode = false
				v.searchInput.Blur()
				v.page = 1
				return v, v.loadBooks()
			default:
				var cmd tea.Cmd
				v.searchInput, cmd = v.searchInput.Update(msg)
				return v, cmd
			}
		}

		// Normal mode key handling
		switch msg.String() {
		case "j", "down":
			v.moveCursor(1)
		case "k", "up":
			v.moveCursor(-1)
		case "g", "home":
			v.cursor = 0
			v.offset = 0
		case "G", "end":
			v.cursor = len(v.books) - 1
			v.updateOffset()
		case "ctrl+d", "pgdown":
			v.moveCursor(v.visibleLines() / 2)
		case "ctrl+u", "pgup":
			v.moveCursor(-v.visibleLines() / 2)
		case "/":
			v.searchMode = true
			v.searchInput.Focus()
			return v, textinput.Blink
		case "s":
			// Cycle sort field
			v.sortBy = (v.sortBy + 1) % 4
			v.page = 1
			return v, v.loadBooks()
		case "S":
			// Toggle sort order
			v.sortAsc = !v.sortAsc
			v.page = 1
			return v, v.loadBooks()
		case "enter":
			if len(v.books) > 0 && v.cursor < len(v.books) {
				book := v.books[v.cursor]
				return v, func() tea.Msg {
					return OpenBookMsg{Book: book}
				}
			}
		case "n":
			// Next page
			if v.hasNextPage() {
				v.page++
				return v, v.loadBooks()
			}
		case "p":
			// Previous page
			if v.page > 1 {
				v.page--
				return v, v.loadBooks()
			}
		case "r":
			// Refresh
			return v, v.loadBooks()
		case "c":
			// Collections
			return v, SwitchTo(ViewCollections)
		case "a":
			// Add/upload book
			return v, SwitchTo(ViewUpload)
		}

	case booksLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.books = msg.books
		v.total = msg.total
		v.err = nil
		if v.cursor >= len(v.books) {
			v.cursor = max(0, len(v.books)-1)
		}
		return v, nil
	}

	return v, nil
}

// View implements View
func (v *LibraryView) View() string {
	var b strings.Builder

	// Header
	header := v.renderHeader()
	b.WriteString(header + "\n")

	// Search bar (if active)
	if v.searchMode {
		searchBar := styles.InputFieldFocused.Render(v.searchInput.View())
		b.WriteString(searchBar + "\n")
	}

	// Loading state
	if v.loading {
		content := lipgloss.Place(
			v.width,
			v.height-4,
			lipgloss.Center,
			lipgloss.Center,
			styles.MutedText.Render("Loading books..."),
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

	// Empty state
	if len(v.books) == 0 {
		content := lipgloss.Place(
			v.width,
			v.height-4,
			lipgloss.Center,
			lipgloss.Center,
			styles.MutedText.Render("No books found"),
		)
		b.WriteString(content)
		return b.String()
	}

	// Book list
	visibleLines := v.visibleLines()
	for i := v.offset; i < min(v.offset+visibleLines, len(v.books)); i++ {
		book := v.books[i]
		line := v.renderBookLine(book, i == v.cursor)
		b.WriteString(line + "\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(v.renderFooter())

	return b.String()
}

// SetSize implements View
func (v *LibraryView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.searchInput.Width = min(40, width-10)
}

// renderHeader renders the header bar
func (v *LibraryView) renderHeader() string {
	title := styles.TitleBar.Render(" Library ")

	// Sort indicator
	sortDir := "↑"
	if !v.sortAsc {
		sortDir = "↓"
	}
	sortInfo := styles.Help.Render(fmt.Sprintf(" Sort: %s %s ", v.sortBy.Label(), sortDir))

	// Search indicator
	searchInfo := ""
	if v.searchInput.Value() != "" {
		searchInfo = styles.SecondaryText.Render(fmt.Sprintf(" [Search: %s]", v.searchInput.Value()))
	}

	// Page info
	totalPages := (v.total + v.pageSize - 1) / v.pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	pageInfo := styles.Help.Render(fmt.Sprintf(" Page %d/%d ", v.page, totalPages))

	// Combine
	left := title + sortInfo + searchInfo
	right := pageInfo

	gap := v.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}

// renderBookLine renders a single book line
func (v *LibraryView) renderBookLine(book models.Book, selected bool) string {
	// Format: [cursor] Title - Author (Series #N)
	title := book.Title
	author := book.Author
	series := ""
	if book.Series != "" {
		series = fmt.Sprintf(" (%s", book.Series)
		if book.SeriesIndex > 0 {
			series += fmt.Sprintf(" #%.0f", book.SeriesIndex)
		}
		series += ")"
	}

	// Truncate if needed
	maxWidth := v.width - 4
	line := fmt.Sprintf("%s - %s%s", title, author, series)
	if len(line) > maxWidth {
		line = line[:maxWidth-3] + "..."
	}

	if selected {
		return styles.ListItemSelected.Width(v.width).Render("▸ " + line)
	}
	return styles.ListItem.Render("  " + line)
}

// renderFooter renders the footer help
func (v *LibraryView) renderFooter() string {
	help := []string{
		styles.HelpKey.Render("j/k") + styles.Help.Render(" nav"),
		styles.HelpKey.Render("enter") + styles.Help.Render(" open"),
		styles.HelpKey.Render("/") + styles.Help.Render(" search"),
		styles.HelpKey.Render("s") + styles.Help.Render(" sort"),
		styles.HelpKey.Render("a") + styles.Help.Render(" add"),
		styles.HelpKey.Render("c") + styles.Help.Render(" collections"),
		styles.HelpKey.Render("q") + styles.Help.Render(" quit"),
	}
	return strings.Join(help, "  ")
}

// loadBooks fetches books from the API
func (v *LibraryView) loadBooks() tea.Cmd {
	return func() tea.Msg {
		order := "asc"
		if !v.sortAsc {
			order = "desc"
		}
		resp, err := v.client.ListBooks(v.page, v.pageSize, v.sortBy.String(), order, v.searchInput.Value())
		if err != nil {
			return booksLoadedMsg{err: err}
		}
		return booksLoadedMsg{books: resp.Books, total: resp.Total}
	}
}

// moveCursor moves the cursor by delta
func (v *LibraryView) moveCursor(delta int) {
	v.cursor += delta
	if v.cursor < 0 {
		v.cursor = 0
	}
	if v.cursor >= len(v.books) {
		v.cursor = len(v.books) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}
	v.updateOffset()
}

// updateOffset ensures the cursor is visible
func (v *LibraryView) updateOffset() {
	visibleLines := v.visibleLines()
	if v.cursor < v.offset {
		v.offset = v.cursor
	}
	if v.cursor >= v.offset+visibleLines {
		v.offset = v.cursor - visibleLines + 1
	}
}

// visibleLines returns the number of visible book lines
func (v *LibraryView) visibleLines() int {
	// Account for header, footer, and margins
	lines := v.height - 5
	if v.searchMode {
		lines--
	}
	if lines < 1 {
		lines = 1
	}
	return lines
}

// hasNextPage returns true if there are more pages
func (v *LibraryView) hasNextPage() bool {
	return v.page*v.pageSize < v.total
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
