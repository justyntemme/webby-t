package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/config"
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

	return &LibraryView{
		client:      client,
		config:      cfg,
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

// bookDeletedMsg is sent when a book is deleted
type bookDeletedMsg struct {
	bookID string
	err    error
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

// renderHeader renders the header bar
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
		titleText = " Author: " + v.filterAuthor + " "
	} else if v.filterSeries != "" {
		titleText = " Series: " + v.filterSeries + " "
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
	// Queue position or favorite star indicator
	indicator := "   "
	if v.config != nil {
		if queuePos := v.config.GetQueuePosition(book.ID); queuePos > 0 {
			indicator = fmt.Sprintf("%2d ", queuePos)
		} else if v.config.IsFavorite(book.ID) {
			indicator = " ★ "
		}
	}

	// Content type badge (only show when viewing "all")
	badge := ""
	if v.contentType == "" && book.ContentType != "" {
		if book.IsComic() {
			badge = styles.BadgeComic.Render("C") + " "
		} else {
			badge = styles.BadgeBook.Render("B") + " "
		}
	}

	// Format: [badge] Title - Author (Series #N)
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

	// Truncate if needed (account for badge and indicator width)
	badgeWidth := 0
	if badge != "" {
		badgeWidth = 4 // "[X] " width
	}
	indicatorWidth := 3 // "★ " or "  " or "99 "
	maxWidth := v.width - 4 - badgeWidth - indicatorWidth
	line := fmt.Sprintf("%s - %s%s", title, author, series)
	if len(line) > maxWidth {
		line = line[:maxWidth-3] + "..."
	}

	if selected {
		return styles.ListItemSelected.Width(v.width).Render("▸ " + indicator + badge + line)
	}
	return styles.ListItem.Render("  " + indicator + badge + line)
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
