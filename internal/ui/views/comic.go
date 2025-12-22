package views

import (
	"bytes"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/BourgeoisBear/rasterm"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/pkg/models"
)

// termImageMode represents the terminal's image display capability
type termImageMode int

const (
	termModeNone termImageMode = iota
	termModeKitty
	termModeIterm
	termModeSixel
)

// ComicView displays comic pages with image rendering
type ComicView struct {
	client *api.Client

	// Book info
	book      models.Book
	pageCount int

	// Current state
	currentPage int
	loading     bool
	err         error

	// Image data
	imageData   []byte
	imageType   string
	imageLoaded bool

	// Terminal capabilities
	termMode termImageMode

	// Dimensions
	width  int
	height int
}

// NewComicView creates a new comic viewer
func NewComicView(client *api.Client) *ComicView {
	// Detect terminal image mode
	mode := detectTerminalMode()

	return &ComicView{
		client:      client,
		currentPage: 1,
		width:       80,
		height:      24,
		termMode:    mode,
	}
}

// detectTerminalMode checks which image protocol the terminal supports
func detectTerminalMode() termImageMode {
	// Check for Kitty protocol support
	if rasterm.IsKittyCapable() {
		return termModeKitty
	}

	// Check for iTerm2 protocol support
	if rasterm.IsItermCapable() {
		return termModeIterm
	}

	// Check for Sixel support
	if capable, _ := rasterm.IsSixelCapable(); capable {
		return termModeSixel
	}

	// No image support
	return termModeNone
}

// SetBook sets the comic to display
func (v *ComicView) SetBook(book models.Book) {
	v.book = book
	v.currentPage = 1
	v.imageData = nil
	v.imageLoaded = false
	v.err = nil
}

// comicPagesLoadedMsg is sent when page count is retrieved
type comicPagesLoadedMsg struct {
	pageCount int
	err       error
}

// comicPageLoadedMsg is sent when a page image is loaded
type comicPageLoadedMsg struct {
	data      []byte
	imageType string
	page      int
	err       error
}

// Init implements View
func (v *ComicView) Init() tea.Cmd {
	v.loading = true
	return v.loadPageCount()
}

// Update implements View
func (v *ComicView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			// Clear screen to remove any lingering images before switching views
			return v, tea.Sequence(tea.ClearScreen, SwitchTo(ViewLibrary))
		case "j", "down", "l", "right", "n", " ", "pgdown":
			// Next page
			if v.currentPage < v.pageCount {
				v.currentPage++
				v.imageLoaded = false
				return v, v.loadPage(v.currentPage)
			}
		case "k", "up", "h", "left", "p", "pgup":
			// Previous page
			if v.currentPage > 1 {
				v.currentPage--
				v.imageLoaded = false
				return v, v.loadPage(v.currentPage)
			}
		case "g", "home":
			// First page
			if v.currentPage != 1 {
				v.currentPage = 1
				v.imageLoaded = false
				return v, v.loadPage(v.currentPage)
			}
		case "G", "end":
			// Last page
			if v.currentPage != v.pageCount && v.pageCount > 0 {
				v.currentPage = v.pageCount
				v.imageLoaded = false
				return v, v.loadPage(v.currentPage)
			}
		}

	case comicPagesLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.pageCount = msg.pageCount
		// Load first page
		return v, v.loadPage(1)

	case comicPageLoadedMsg:
		if msg.page == v.currentPage {
			if msg.err != nil {
				v.err = msg.err
				return v, nil
			}
			v.imageData = msg.data
			v.imageType = msg.imageType
			v.imageLoaded = true
			v.err = nil
		}
	}

	return v, nil
}

// View implements View
func (v *ComicView) View() string {
	var b strings.Builder

	// Header
	b.WriteString(v.renderHeader() + "\n")

	// Content area
	contentHeight := v.height - 4 // Header + footer + margins

	if v.loading {
		content := lipgloss.Place(
			v.width,
			contentHeight,
			lipgloss.Center,
			lipgloss.Center,
			styles.MutedText.Render("Loading comic..."),
		)
		b.WriteString(content)
	} else if v.err != nil {
		content := lipgloss.Place(
			v.width,
			contentHeight,
			lipgloss.Center,
			lipgloss.Center,
			styles.ErrorStyle.Render("Error: "+v.err.Error()),
		)
		b.WriteString(content)
	} else if v.termMode == termModeNone {
		// No image protocol support
		content := lipgloss.Place(
			v.width,
			contentHeight,
			lipgloss.Center,
			lipgloss.Center,
			styles.MutedText.Render("Terminal does not support images.\n\nSupported terminals: Kitty, iTerm2, or Sixel-capable terminals."),
		)
		b.WriteString(content)
	} else if !v.imageLoaded {
		content := lipgloss.Place(
			v.width,
			contentHeight,
			lipgloss.Center,
			lipgloss.Center,
			styles.MutedText.Render(fmt.Sprintf("Loading page %d...", v.currentPage)),
		)
		b.WriteString(content)
	} else {
		// Render the image
		imageStr := v.renderImage()
		b.WriteString(imageStr)
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(v.renderFooter())

	return b.String()
}

// renderHeader renders the header bar
func (v *ComicView) renderHeader() string {
	// Title
	title := v.book.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}
	titleBar := styles.TitleBar.Render(" " + title + " ")

	// Page indicator
	pageInfo := ""
	if v.pageCount > 0 {
		pageInfo = styles.Help.Render(fmt.Sprintf(" Page %d/%d ", v.currentPage, v.pageCount))
	}

	// Combine
	gap := v.width - lipgloss.Width(titleBar) - lipgloss.Width(pageInfo)
	if gap < 0 {
		gap = 0
	}

	return titleBar + strings.Repeat(" ", gap) + pageInfo
}

// renderImage renders the current page image to the terminal
func (v *ComicView) renderImage() string {
	if len(v.imageData) == 0 {
		return styles.MutedText.Render("No image data")
	}

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(v.imageData))
	if err != nil {
		return styles.ErrorStyle.Render("Failed to decode image: " + err.Error())
	}

	// Render the image to terminal
	var buf bytes.Buffer
	var renderErr error

	switch v.termMode {
	case termModeKitty:
		renderErr = rasterm.KittyWriteImage(&buf, img, rasterm.KittyImgOpts{})
	case termModeIterm:
		renderErr = rasterm.ItermWriteImage(&buf, img)
	case termModeSixel:
		// Sixel requires paletted image
		paletted := imageToPaletted(img)
		renderErr = rasterm.SixelWriteImage(os.Stdout, paletted)
		// Sixel writes directly to stdout, return empty string
		if renderErr != nil {
			return styles.ErrorStyle.Render("Sixel error: " + renderErr.Error())
		}
		return ""
	default:
		return styles.MutedText.Render("No supported image protocol")
	}

	if renderErr != nil {
		return styles.ErrorStyle.Render("Render error: " + renderErr.Error())
	}

	return buf.String()
}

// imageToPaletted converts an image to a paletted image for Sixel
func imageToPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	paletted := image.NewPaletted(bounds, palette.Plan9)
	draw.Draw(paletted, bounds, img, bounds.Min, draw.Src)
	return paletted
}

// renderFooter renders the footer help
func (v *ComicView) renderFooter() string {
	help := []string{
		styles.HelpKey.Render("h/l") + styles.Help.Render(" prev/next"),
		styles.HelpKey.Render("g/G") + styles.Help.Render(" first/last"),
		styles.HelpKey.Render("q") + styles.Help.Render(" back"),
	}
	return strings.Join(help, "  ")
}

// SetSize implements View
func (v *ComicView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// loadPageCount fetches the comic page count
func (v *ComicView) loadPageCount() tea.Cmd {
	return func() tea.Msg {
		resp, err := v.client.GetComicPages(v.book.ID)
		if err != nil {
			return comicPagesLoadedMsg{err: err}
		}
		return comicPagesLoadedMsg{pageCount: resp.PageCount}
	}
}

// loadPage fetches a specific page image (converts 1-indexed to 0-indexed for API)
func (v *ComicView) loadPage(page int) tea.Cmd {
	return func() tea.Msg {
		// API uses 0-indexed pages, UI uses 1-indexed
		data, imageType, err := v.client.GetComicPage(v.book.ID, page-1)
		if err != nil {
			return comicPageLoadedMsg{page: page, err: err}
		}
		return comicPageLoadedMsg{page: page, data: data, imageType: imageType}
	}
}
