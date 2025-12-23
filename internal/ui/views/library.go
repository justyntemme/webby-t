package views

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/config"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/internal/ui/terminal"
	"github.com/justyntemme/webby-t/pkg/models"
	"github.com/nfnt/resize"
)

// Thumbnail dimensions
const (
	thumbHeight = 5  // Lines high for thumbnail
	thumbWidth  = 10 // Characters wide for thumbnail
)

// Column layout constants for uniform text display
const (
	colIndicator = 3  // "▸ " or "  " + queue/fav indicator
	colBadge     = 4  // "[C] " or "[B] " content type badge
	colAuthor    = 20 // Fixed author column width
	colSeries    = 18 // Fixed series column width
	minTitleCol  = 20 // Minimum title column width
)

// truncateText truncates a string to maxWidth visible characters with ellipsis
// Uses lipgloss.Width for accurate measurement of styled text
func truncateText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	// Use lipgloss.Width for visible width (handles ANSI codes)
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	// For truncation, work with runes of plain text
	runes := []rune(text)
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	// Truncate and add ellipsis
	for i := len(runes) - 1; i >= 0; i-- {
		candidate := string(runes[:i]) + "..."
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return "..."
}

// padRight pads a string to exactly width visible characters
// Uses lipgloss.Width to handle ANSI styled text properly
func padRight(text string, width int) string {
	if width <= 0 {
		return ""
	}
	textWidth := lipgloss.Width(text)
	if textWidth >= width {
		return truncateText(text, width)
	}
	return text + strings.Repeat(" ", width-textWidth)
}

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
	config *config.Config

	// Books
	books       []models.Book
	cursor      int
	offset      int // For scrolling

	// State
	loading          bool
	err              error
	searchMode       bool
	searchInput      textinput.Model
	recentlyReadMode bool
	favoritesMode    bool         // Show only favorites
	queueMode        bool         // Show only reading queue
	confirmDelete    bool         // Show delete confirmation
	deleteBook       *models.Book // Book pending deletion
	filterAuthor     string       // Filter by author name
	filterSeries     string       // Filter by series name

	// Sorting
	sortBy    sortField
	sortAsc   bool

	// Content type filter ("", "book", or "comic")
	contentType string

	// Pagination
	page      int
	pageSize  int
	total     int

	// Thumbnail support
	termMode   terminal.TermImageMode
	coverCache map[string]string // Rendered image strings by book ID
	showCovers bool              // Toggle for showing covers (default true if supported)

	// Dimensions
	width  int
	height int
}

// NewLibraryView creates a new library view
func NewLibraryView(client *api.Client, cfg *config.Config) *LibraryView {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search books..."
	searchInput.CharLimit = 100
	searchInput.Width = 40

	termMode := terminal.DetectTerminalMode()
	return &LibraryView{
		client:      client,
		config:      cfg,
		pageSize:    50,
		page:        1,
		sortBy:      sortTitle,
		sortAsc:     true,
		searchInput: searchInput,
		termMode:    termMode,
		coverCache:  make(map[string]string),
		showCovers:  termMode != terminal.TermModeNone, // Enable by default if supported
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

// bookDeletedMsg is sent when a book is deleted
type bookDeletedMsg struct {
	bookID string
	err    error
}

// coverLoadedMsg is sent when a book cover is fetched and rendered
type coverLoadedMsg struct {
	bookID        string
	renderedImage string
	err           error
}

// loadCoverCmd creates a command to fetch, render, and cache a book cover
func (v *LibraryView) loadCoverCmd(bookID string) tea.Cmd {
	if v.termMode == terminal.TermModeNone {
		return nil // No image support
	}
	if _, exists := v.coverCache[bookID]; exists {
		return nil // Already cached
	}

	return func() tea.Msg {
		imgData, _, err := v.client.GetBookCover(bookID)
		if err != nil || len(imgData) == 0 {
			return coverLoadedMsg{bookID: bookID, err: err}
		}

		img, _, err := image.Decode(bytes.NewReader(imgData))
		if err != nil {
			return coverLoadedMsg{bookID: bookID, err: err}
		}

		// Resize to thumbnail size (height in pixels, roughly 8 pixels per line)
		resizedImg := resize.Resize(0, uint(thumbHeight*8), img, resize.Lanczos3)

		renderedImage, err := terminal.RenderImageToString(resizedImg, v.termMode)
		if err != nil {
			return coverLoadedMsg{bookID: bookID, err: err}
		}

		return coverLoadedMsg{bookID: bookID, renderedImage: renderedImage}
	}
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
		// Handle delete confirmation mode
		if v.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				// Confirm delete
				v.confirmDelete = false
				if v.deleteBook != nil {
					return v, v.deleteBookCmd(v.deleteBook.ID)
				}
			case "n", "N", "esc":
				// Cancel delete
				v.confirmDelete = false
				v.deleteBook = nil
			}
			return v, nil
		}

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
		case "b":
			// Filter books only
			if v.contentType == models.ContentTypeBook {
				v.contentType = "" // Toggle off
			} else {
				v.contentType = models.ContentTypeBook
			}
			v.page = 1
			return v, v.loadBooks()
		case "m":
			// Filter comics only
			if v.contentType == models.ContentTypeComic {
				v.contentType = "" // Toggle off
			} else {
				v.contentType = models.ContentTypeComic
			}
			v.page = 1
			return v, v.loadBooks()
		case "v":
			// Cycle through content types: all -> books -> comics -> all
			switch v.contentType {
			case "":
				v.contentType = models.ContentTypeBook
			case models.ContentTypeBook:
				v.contentType = models.ContentTypeComic
			case models.ContentTypeComic:
				v.contentType = ""
			}
			v.page = 1
			return v, v.loadBooks()
		case "R":
			// Toggle recently read filter
			v.recentlyReadMode = !v.recentlyReadMode
			v.page = 1
			v.cursor = 0
			v.offset = 0
			return v, v.loadBooks()
		case "d":
			// Delete book (with confirmation)
			if len(v.books) > 0 && v.cursor < len(v.books) {
				book := v.books[v.cursor]
				v.deleteBook = &book
				v.confirmDelete = true
			}
		case "f":
			// Toggle favorite on selected book
			if len(v.books) > 0 && v.cursor < len(v.books) && v.config != nil {
				book := v.books[v.cursor]
				_ = v.config.ToggleFavorite(book.ID)
			}
		case "F":
			// Toggle favorites filter
			v.favoritesMode = !v.favoritesMode
			v.queueMode = false
			v.page = 1
			v.cursor = 0
			v.offset = 0
			return v, v.loadBooks()
		case "w":
			// Toggle reading queue on selected book
			if len(v.books) > 0 && v.cursor < len(v.books) && v.config != nil {
				book := v.books[v.cursor]
				_ = v.config.ToggleQueue(book.ID)
			}
		case "W":
			// Toggle reading queue filter
			v.queueMode = !v.queueMode
			v.favoritesMode = false
			v.page = 1
			v.cursor = 0
			v.offset = 0
			return v, v.loadBooks()
		case "J":
			// Move book down in queue (when in queue mode)
			if v.queueMode && len(v.books) > 0 && v.cursor < len(v.books) && v.config != nil {
				book := v.books[v.cursor]
				_ = v.config.MoveInQueue(book.ID, 1)
				if v.cursor < len(v.books)-1 {
					v.cursor++
				}
				return v, v.loadBooks()
			}
		case "K":
			// Move book up in queue (when in queue mode)
			if v.queueMode && len(v.books) > 0 && v.cursor < len(v.books) && v.config != nil {
				book := v.books[v.cursor]
				_ = v.config.MoveInQueue(book.ID, -1)
				if v.cursor > 0 {
					v.cursor--
				}
				return v, v.loadBooks()
			}
		case "A":
			// Filter by selected book's author
			if len(v.books) > 0 && v.cursor < len(v.books) {
				book := v.books[v.cursor]
				if book.Author != "" {
					v.filterAuthor = book.Author
					v.filterSeries = ""
					v.page = 1
					v.cursor = 0
					v.offset = 0
					return v, v.loadBooks()
				}
			}
		case "E":
			// Filter by selected book's series (E for sEries, since S is sort)
			if len(v.books) > 0 && v.cursor < len(v.books) {
				book := v.books[v.cursor]
				if book.Series != "" {
					v.filterSeries = book.Series
					v.filterAuthor = ""
					v.page = 1
					v.cursor = 0
					v.offset = 0
					return v, v.loadBooks()
				}
			}
		case "x":
			// Clear author/series filter
			if v.filterAuthor != "" || v.filterSeries != "" {
				v.filterAuthor = ""
				v.filterSeries = ""
				v.page = 1
				v.cursor = 0
				v.offset = 0
				return v, v.loadBooks()
			}
		case "i":
			// Show book details
			if len(v.books) > 0 && v.cursor < len(v.books) {
				book := v.books[v.cursor]
				return v, func() tea.Msg {
					return ShowBookDetailsMsg{Book: book}
				}
			}
		case "T":
			// Cycle through themes
			newTheme := styles.NextTheme()
			if v.config != nil {
				_ = v.config.SetTheme(newTheme)
			}
			return v, NotifyThemeChanged(newTheme)
		case "C":
			// Toggle cover thumbnails (only if terminal supports images)
			if v.termMode != terminal.TermModeNone {
				v.showCovers = !v.showCovers
				// Load covers if enabling and we have books
				if v.showCovers && len(v.books) > 0 {
					var cmds []tea.Cmd
					visibleCount := v.visibleLines()
					for i := 0; i < min(visibleCount, len(v.books)); i++ {
						if cmd := v.loadCoverCmd(v.books[i].ID); cmd != nil {
							cmds = append(cmds, cmd)
						}
					}
					return v, tea.Batch(cmds...)
				}
			}
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

		// Load covers for visible books if image support available
		var cmds []tea.Cmd
		if v.termMode != terminal.TermModeNone {
			visibleCount := v.visibleLines()
			for i := 0; i < min(visibleCount, len(v.books)); i++ {
				if cmd := v.loadCoverCmd(v.books[i].ID); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}
		return v, tea.Batch(cmds...)

	case coverLoadedMsg:
		if msg.err == nil && msg.renderedImage != "" {
			v.coverCache[msg.bookID] = msg.renderedImage
		}
		return v, nil

	case bookDeletedMsg:
		v.deleteBook = nil
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		// Refresh the book list
		return v, v.loadBooks()
	}

	return v, nil
}

// View implements View
func (v *LibraryView) View() string {
	var b strings.Builder

	// Delete confirmation dialog
	if v.confirmDelete && v.deleteBook != nil {
		return v.renderDeleteConfirmation()
	}

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

// GetTermMode returns the terminal image mode for cleanup purposes
func (v *LibraryView) GetTermMode() terminal.TermImageMode {
	return v.termMode
}

// renderHeader renders the header bar with proper truncation
func (v *LibraryView) renderHeader() string {
	// Title based on mode and content type filter
	titleText := " Library "
	if v.queueMode {
		titleText = " Reading Queue "
	} else if v.favoritesMode {
		titleText = " ★ Favorites "
	} else if v.recentlyReadMode {
		titleText = " Recently Read "
	} else if v.filterAuthor != "" {
		// Truncate long author names in header
		authorName := truncateText(v.filterAuthor, 20)
		titleText = " Author: " + authorName + " "
	} else if v.filterSeries != "" {
		// Truncate long series names in header
		seriesName := truncateText(v.filterSeries, 20)
		titleText = " Series: " + seriesName + " "
	} else {
		switch v.contentType {
		case models.ContentTypeBook:
			titleText = " Books "
		case models.ContentTypeComic:
			titleText = " Comics "
		}
	}
	title := styles.TitleBar.Render(titleText)

	// Sort indicator
	sortDir := "↑"
	if !v.sortAsc {
		sortDir = "↓"
	}
	sortInfo := styles.Help.Render(fmt.Sprintf(" Sort: %s %s ", v.sortBy.Label(), sortDir))

	// Page info (fixed width, always show)
	totalPages := (v.total + v.pageSize - 1) / v.pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	pageInfo := styles.Help.Render(fmt.Sprintf(" Page %d/%d ", v.page, totalPages))

	// Calculate available space for search info
	fixedWidth := lipgloss.Width(title) + lipgloss.Width(sortInfo) + lipgloss.Width(pageInfo) + 2
	availableForSearch := v.width - fixedWidth

	// Search indicator (truncated to fit)
	searchInfo := ""
	if v.searchInput.Value() != "" {
		searchQuery := v.searchInput.Value()
		if availableForSearch > 12 {
			maxQueryLen := availableForSearch - 12 // Account for " [Search: ]"
			if maxQueryLen > 0 {
				searchQuery = truncateText(searchQuery, maxQueryLen)
			}
			searchInfo = styles.SecondaryText.Render(fmt.Sprintf(" [Search: %s]", searchQuery))
		}
	}

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
	// Check if we have image support and covers are enabled
	if v.showCovers && v.termMode != terminal.TermModeNone {
		return v.renderBookLineWithThumbnail(book, selected)
	}
	return v.renderBookLineTextOnly(book, selected)
}

// renderBookLineTextOnly renders a compact text-only book line with fixed-width columns
func (v *LibraryView) renderBookLineTextOnly(book models.Book, selected bool) string {
	// --- Build fixed-width columns ---

	// Indicator column (3 chars): queue position or favorite star
	indicator := "   "
	if v.config != nil {
		if queuePos := v.config.GetQueuePosition(book.ID); queuePos > 0 {
			indicator = fmt.Sprintf("%2d ", queuePos)
		} else if v.config.IsFavorite(book.ID) {
			indicator = " ★ "
		}
	}

	// Badge column (4 chars): content type badge
	badgeText := "    "
	if v.contentType == "" && book.ContentType != "" {
		if book.IsComic() {
			badgeText = styles.BadgeComic.Render("[C]") + " "
		} else {
			badgeText = styles.BadgeBook.Render("[B]") + " "
		}
	}
	badge := padRight(badgeText, colBadge)

	// Author column (fixed width)
	author := padRight(book.Author, colAuthor)

	// Series column (fixed width)
	seriesText := ""
	if book.Series != "" {
		seriesText = book.Series
		if book.SeriesIndex > 0 {
			seriesText += fmt.Sprintf(" #%.0f", book.SeriesIndex)
		}
	}
	series := padRight(seriesText, colSeries)

	// Title column (dynamic - fills remaining space)
	const selectorWidth = 2
	titleWidth := v.width - selectorWidth - colIndicator - colBadge - colAuthor - colSeries - 1
	if titleWidth < minTitleCol {
		titleWidth = minTitleCol
	}
	title := padRight(book.Title, titleWidth)

	// --- Assemble line with fixed columns ---
	line := indicator + badge + title + " " + author + " " + series

	// --- Apply selection styling ---
	if selected {
		return styles.ListItemSelected.Width(v.width).Render("▸ " + line)
	}
	return styles.ListItem.Width(v.width).Render("  " + line)
}

// renderBookLineWithThumbnail renders a book line with cover thumbnail and aligned details
func (v *LibraryView) renderBookLineWithThumbnail(book models.Book, selected bool) string {
	// Left column: Thumbnail or placeholder
	var leftCol string
	if renderedImg, ok := v.coverCache[book.ID]; ok && renderedImg != "" {
		leftCol = lipgloss.NewStyle().
			Width(thumbWidth).
			Height(thumbHeight).
			Render(renderedImg)
	} else {
		// Placeholder while loading
		placeholder := styles.MutedText.Render("[...]")
		leftCol = lipgloss.NewStyle().
			Width(thumbWidth).
			Height(thumbHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render(placeholder)
	}

	// Right column: Book details with proper truncation
	const selectorWidth = 2
	rightColWidth := v.width - thumbWidth - selectorWidth - 2

	// Build book info with truncation to prevent overflow
	titleStyle := styles.BookTitle
	if selected {
		titleStyle = titleStyle.Bold(true)
	}
	title := titleStyle.Render(truncateText(book.Title, rightColWidth-2))

	authorText := ""
	if book.Author != "" {
		authorText = "by " + book.Author
	}
	author := styles.BookAuthor.Render(truncateText(authorText, rightColWidth-2))

	// Series info (truncated)
	series := ""
	if book.Series != "" {
		seriesText := book.Series
		if book.SeriesIndex > 0 {
			seriesText += fmt.Sprintf(" #%.0f", book.SeriesIndex)
		}
		series = styles.MutedText.Render(truncateText(seriesText, rightColWidth-2))
	}

	// Build indicators line (queue pos, favorite, badge on same line)
	var indicators []string
	if v.config != nil {
		if queuePos := v.config.GetQueuePosition(book.ID); queuePos > 0 {
			indicators = append(indicators, styles.SecondaryText.Render(fmt.Sprintf("#%d", queuePos)))
		} else if v.config.IsFavorite(book.ID) {
			indicators = append(indicators, styles.SecondaryText.Render("★"))
		}
	}
	if v.contentType == "" && book.ContentType != "" {
		if book.IsComic() {
			indicators = append(indicators, styles.BadgeComic.Render("[C]"))
		} else {
			indicators = append(indicators, styles.BadgeBook.Render("[B]"))
		}
	}

	// Combine details vertically (max 4 lines to fit thumbHeight=5)
	var lines []string
	lines = append(lines, title)
	lines = append(lines, author)
	if series != "" {
		lines = append(lines, series)
	}
	if len(indicators) > 0 {
		lines = append(lines, strings.Join(indicators, " "))
	}

	details := lipgloss.JoinVertical(lipgloss.Left, lines...)

	rightCol := lipgloss.NewStyle().
		Width(rightColWidth).
		Height(thumbHeight).
		Padding(0, 1).
		Render(details)

	// Join columns
	fullLine := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)

	// Selection styling
	selector := "  "
	if selected {
		selector = "▸ "
		return styles.ListItemSelected.Width(v.width).Render(selector + fullLine)
	}
	return styles.ListItem.Width(v.width).Render(selector + fullLine)
}

// renderFooter renders the footer help
func (v *LibraryView) renderFooter() string {
	var help []string
	if v.queueMode {
		help = []string{
			styles.HelpKey.Render("j/k") + styles.Help.Render(" nav"),
			styles.HelpKey.Render("J/K") + styles.Help.Render(" reorder"),
			styles.HelpKey.Render("enter") + styles.Help.Render(" open"),
			styles.HelpKey.Render("w") + styles.Help.Render(" remove"),
			styles.HelpKey.Render("W") + styles.Help.Render(" exit"),
			styles.HelpKey.Render("q") + styles.Help.Render(" quit"),
		}
	} else if v.filterAuthor != "" || v.filterSeries != "" {
		// Show filter-specific help when a filter is active
		help = []string{
			styles.HelpKey.Render("j/k") + styles.Help.Render(" nav"),
			styles.HelpKey.Render("enter") + styles.Help.Render(" open"),
			styles.HelpKey.Render("x") + styles.Help.Render(" clear filter"),
			styles.HelpKey.Render("f") + styles.Help.Render(" fav"),
			styles.HelpKey.Render("w") + styles.Help.Render(" queue"),
			styles.HelpKey.Render("q") + styles.Help.Render(" quit"),
		}
	} else {
		help = []string{
			styles.HelpKey.Render("j/k") + styles.Help.Render(" nav"),
			styles.HelpKey.Render("enter") + styles.Help.Render(" open"),
			styles.HelpKey.Render("i") + styles.Help.Render(" info"),
			styles.HelpKey.Render("/") + styles.Help.Render(" search"),
			styles.HelpKey.Render("f") + styles.Help.Render(" fav"),
			styles.HelpKey.Render("w") + styles.Help.Render(" queue"),
			styles.HelpKey.Render("d") + styles.Help.Render(" del"),
			styles.HelpKey.Render("q") + styles.Help.Render(" quit"),
		}
	}

	// Add theme indicator
	themeName := styles.CurrentTheme().Name
	themeIndicator := styles.MutedText.Render(" [Theme: " + themeName + "] ") + styles.HelpKey.Render("T") + styles.Help.Render(" change")

	helpText := strings.Join(help, "  ")
	gap := v.width - lipgloss.Width(helpText) - lipgloss.Width(themeIndicator)
	if gap < 0 {
		gap = 0
	}

	return helpText + strings.Repeat(" ", gap) + themeIndicator
}

// renderDeleteConfirmation renders the delete confirmation dialog
func (v *LibraryView) renderDeleteConfirmation() string {
	title := v.deleteBook.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	dialog := styles.Dialog.Width(50).Render(
		styles.DialogTitle.Render("Delete Book?") + "\n\n" +
			styles.BookTitle.Render(title) + "\n" +
			styles.BookAuthor.Render("by "+v.deleteBook.Author) + "\n\n" +
			styles.ErrorStyle.Render("This action cannot be undone.") + "\n\n" +
			styles.Help.Render("Press ") +
			styles.HelpKey.Render("y") +
			styles.Help.Render(" to confirm, ") +
			styles.HelpKey.Render("n") +
			styles.Help.Render(" to cancel"),
	)

	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// deleteBookCmd creates a command to delete a book
func (v *LibraryView) deleteBookCmd(bookID string) tea.Cmd {
	return func() tea.Msg {
		err := v.client.DeleteBook(bookID)
		return bookDeletedMsg{bookID: bookID, err: err}
	}
}

// loadBooks fetches books from the API
func (v *LibraryView) loadBooks() tea.Cmd {
	return func() tea.Msg {
		order := "asc"
		if !v.sortAsc {
			order = "desc"
		}
		resp, err := v.client.ListBooks(v.page, v.pageSize, v.sortBy.String(), order, v.searchInput.Value(), v.contentType)
		if err != nil {
			return booksLoadedMsg{err: err}
		}

		// Filter by recently read if in that mode
		if v.recentlyReadMode && v.config != nil {
			recentIDs := v.config.GetRecentlyReadIDs()
			recentIDSet := make(map[string]bool)
			for _, id := range recentIDs {
				recentIDSet[id] = true
			}

			// Filter books and maintain recently read order
			filteredBooks := make([]models.Book, 0)
			bookByID := make(map[string]models.Book)
			for _, book := range resp.Books {
				if recentIDSet[book.ID] {
					bookByID[book.ID] = book
				}
			}
			// Order by recently read order
			for _, id := range recentIDs {
				if book, exists := bookByID[id]; exists {
					filteredBooks = append(filteredBooks, book)
				}
			}

			return booksLoadedMsg{books: filteredBooks, total: len(filteredBooks)}
		}

		// Filter by favorites if in that mode
		if v.favoritesMode && v.config != nil {
			favoriteIDs := v.config.GetFavoriteIDs()
			favoriteIDSet := make(map[string]bool)
			for _, id := range favoriteIDs {
				favoriteIDSet[id] = true
			}

			filteredBooks := make([]models.Book, 0)
			for _, book := range resp.Books {
				if favoriteIDSet[book.ID] {
					filteredBooks = append(filteredBooks, book)
				}
			}

			return booksLoadedMsg{books: filteredBooks, total: len(filteredBooks)}
		}

		// Filter by reading queue if in that mode (maintain queue order)
		if v.queueMode && v.config != nil {
			queueIDs := v.config.GetQueueIDs()
			bookByID := make(map[string]models.Book)
			for _, book := range resp.Books {
				bookByID[book.ID] = book
			}

			filteredBooks := make([]models.Book, 0)
			for _, id := range queueIDs {
				if book, exists := bookByID[id]; exists {
					filteredBooks = append(filteredBooks, book)
				}
			}

			return booksLoadedMsg{books: filteredBooks, total: len(filteredBooks)}
		}

		// Filter by author if filter is active
		if v.filterAuthor != "" {
			filteredBooks := make([]models.Book, 0)
			for _, book := range resp.Books {
				if book.Author == v.filterAuthor {
					filteredBooks = append(filteredBooks, book)
				}
			}
			return booksLoadedMsg{books: filteredBooks, total: len(filteredBooks)}
		}

		// Filter by series if filter is active
		if v.filterSeries != "" {
			filteredBooks := make([]models.Book, 0)
			for _, book := range resp.Books {
				if book.Series == v.filterSeries {
					filteredBooks = append(filteredBooks, book)
				}
			}
			return booksLoadedMsg{books: filteredBooks, total: len(filteredBooks)}
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
	availableHeight := v.height - 5
	if v.searchMode {
		availableHeight--
	}

	// If covers are shown, each item takes multiple lines
	if v.showCovers && v.termMode != terminal.TermModeNone {
		// Add 1 for spacing between items
		lines := availableHeight / (thumbHeight + 1)
		if lines < 1 {
			return 1
		}
		return lines
	}

	// Text-only mode: one line per book
	if availableHeight < 1 {
		return 1
	}
	return availableHeight
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
