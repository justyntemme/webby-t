package views

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/internal/ui/terminal"
	"github.com/justyntemme/webby-t/pkg/models"
)

// Zoom levels available
var zoomLevels = []float64{1.0, 1.5, 2.0, 3.0, 4.0}

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
	decodedImg  image.Image // Cached decoded image for zoom/pan

	// Zoom and pan state
	zoomIndex int     // Index into zoomLevels
	panX      float64 // Pan position as fraction (0.0 = left, 1.0 = right)
	panY      float64 // Pan position as fraction (0.0 = top, 1.0 = bottom)

	// Terminal capabilities
	termMode terminal.TermImageMode

	// Dimensions
	width  int
	height int
}

// NewComicView creates a new comic viewer
func NewComicView(client *api.Client) *ComicView {
	return &ComicView{
		client:      client,
		currentPage: 1,
		width:       80,
		height:      24,
		termMode:    terminal.DetectTerminalMode(),
	}
}

// SetBook sets the comic to display
func (v *ComicView) SetBook(book models.Book) {
	v.book = book
	v.currentPage = 1
	v.imageData = nil
	v.imageLoaded = false
	v.decodedImg = nil
	v.err = nil
	v.resetZoomPan()
}

// resetZoomPan resets zoom and pan to default
func (v *ComicView) resetZoomPan() {
	v.zoomIndex = 0
	v.panX = 0.5 // Center
	v.panY = 0.5 // Center
}

// currentZoom returns the current zoom level
func (v *ComicView) currentZoom() float64 {
	if v.zoomIndex >= 0 && v.zoomIndex < len(zoomLevels) {
		return zoomLevels[v.zoomIndex]
	}
	return 1.0
}

// isZoomed returns true if currently zoomed in
func (v *ComicView) isZoomed() bool {
	return v.zoomIndex > 0
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
		return v.handleKeyMsg(msg)
	case comicPagesLoadedMsg:
		return v.handlePagesLoaded(msg)
	case comicPageLoadedMsg:
		return v.handlePageLoaded(msg)
	}
	return v, nil
}

// handleKeyMsg processes key presses
func (v *ComicView) handleKeyMsg(msg tea.KeyMsg) (View, tea.Cmd) {
	key := msg.String()

	// Exit
	if key == "q" || key == "esc" {
		terminal.ClearImagesCmd(v.termMode)()
		return v, SwitchTo(ViewLibrary)
	}

	// Zoom controls
	switch key {
	case "+", "=":
		v.zoomIn()
		return v, nil
	case "-", "_":
		v.zoomOut()
		return v, nil
	case "0":
		v.resetZoomPan()
		return v, nil
	}

	// When zoomed, arrow keys pan; when not zoomed, they navigate pages
	if v.isZoomed() {
		switch key {
		case "h", "left":
			v.panLeft()
			return v, nil
		case "l", "right":
			v.panRight()
			return v, nil
		case "k", "up":
			v.panUp()
			return v, nil
		case "j", "down":
			v.panDown()
			return v, nil
		}
	}

	// Page navigation (when not zoomed, or using specific keys)
	switch key {
	case "n", " ", "pgdown":
		return v, v.nextPage()
	case "p", "pgup":
		return v, v.prevPage()
	case "g", "home":
		return v, v.firstPage()
	case "G", "end":
		return v, v.lastPage()
	}

	// When not zoomed, h/l also navigate pages
	if !v.isZoomed() {
		switch key {
		case "l", "right", "j", "down":
			return v, v.nextPage()
		case "h", "left", "k", "up":
			return v, v.prevPage()
		}
	}

	return v, nil
}

// Zoom methods
func (v *ComicView) zoomIn() {
	if v.zoomIndex < len(zoomLevels)-1 {
		v.zoomIndex++
	}
}

func (v *ComicView) zoomOut() {
	if v.zoomIndex > 0 {
		v.zoomIndex--
		// Reset pan to center when zooming out to 1x
		if v.zoomIndex == 0 {
			v.panX = 0.5
			v.panY = 0.5
		}
	}
}

// Pan methods (move in 10% increments)
const panStep = 0.1

func (v *ComicView) panLeft() {
	v.panX -= panStep
	if v.panX < 0 {
		v.panX = 0
	}
}

func (v *ComicView) panRight() {
	v.panX += panStep
	if v.panX > 1 {
		v.panX = 1
	}
}

func (v *ComicView) panUp() {
	v.panY -= panStep
	if v.panY < 0 {
		v.panY = 0
	}
}

func (v *ComicView) panDown() {
	v.panY += panStep
	if v.panY > 1 {
		v.panY = 1
	}
}

// Page navigation methods
func (v *ComicView) nextPage() tea.Cmd {
	if v.currentPage < v.pageCount {
		v.currentPage++
		v.imageLoaded = false
		v.decodedImg = nil
		v.resetZoomPan()
		return v.loadPage(v.currentPage)
	}
	return nil
}

func (v *ComicView) prevPage() tea.Cmd {
	if v.currentPage > 1 {
		v.currentPage--
		v.imageLoaded = false
		v.decodedImg = nil
		v.resetZoomPan()
		return v.loadPage(v.currentPage)
	}
	return nil
}

func (v *ComicView) firstPage() tea.Cmd {
	if v.currentPage != 1 {
		v.currentPage = 1
		v.imageLoaded = false
		v.decodedImg = nil
		v.resetZoomPan()
		return v.loadPage(v.currentPage)
	}
	return nil
}

func (v *ComicView) lastPage() tea.Cmd {
	if v.currentPage != v.pageCount && v.pageCount > 0 {
		v.currentPage = v.pageCount
		v.imageLoaded = false
		v.decodedImg = nil
		v.resetZoomPan()
		return v.loadPage(v.currentPage)
	}
	return nil
}

// Message handlers
func (v *ComicView) handlePagesLoaded(msg comicPagesLoadedMsg) (View, tea.Cmd) {
	v.loading = false
	if msg.err != nil {
		v.err = msg.err
		return v, nil
	}
	v.pageCount = msg.pageCount
	return v, v.loadPage(1)
}

func (v *ComicView) handlePageLoaded(msg comicPageLoadedMsg) (View, tea.Cmd) {
	if msg.page == v.currentPage {
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.imageData = msg.data
		v.imageType = msg.imageType
		v.imageLoaded = true
		v.decodedImg = nil // Will be decoded on render
		v.err = nil
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
	} else if v.termMode == terminal.TermModeNone {
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

// renderHeader renders the header bar with proper truncation
func (v *ComicView) renderHeader() string {
	// Title (unicode-safe truncation)
	maxTitleWidth := 40
	if v.width > 0 && v.width/2 < maxTitleWidth {
		maxTitleWidth = v.width / 2
	}
	title := styles.TruncateText(v.book.Title, maxTitleWidth)
	titlePart := styles.BookTitle.Render(title)

	// Page and zoom indicator
	rightPart := ""
	if v.pageCount > 0 {
		pageStr := fmt.Sprintf("%d/%d", v.currentPage, v.pageCount)
		if v.isZoomed() {
			zoomPct := int(v.currentZoom() * 100)
			pageStr += fmt.Sprintf(" [%d%%]", zoomPct)
		}
		rightPart = styles.MutedText.Render(pageStr)
	}

	// Combine
	gap := v.width - lipgloss.Width(titlePart) - lipgloss.Width(rightPart)
	if gap < 0 {
		gap = 0
	}

	return titlePart + strings.Repeat(" ", gap) + rightPart
}

// renderImage renders the current page image to the terminal
func (v *ComicView) renderImage() string {
	if len(v.imageData) == 0 {
		return styles.MutedText.Render("No image data")
	}

	// Decode and cache the image if not already done
	if v.decodedImg == nil {
		img, _, err := image.Decode(bytes.NewReader(v.imageData))
		if err != nil {
			return styles.ErrorStyle.Render("Failed to decode image: " + err.Error())
		}
		v.decodedImg = img
	}

	// Get the image to render (possibly cropped for zoom)
	imgToRender := v.getViewportImage()

	// Use shared utility to render the image
	imgStr, renderErr := terminal.RenderImageToString(imgToRender, v.termMode)
	if renderErr != nil {
		return styles.ErrorStyle.Render("Render error: " + renderErr.Error())
	}

	return imgStr
}

// getViewportImage returns the portion of the image visible at current zoom/pan
func (v *ComicView) getViewportImage() image.Image {
	if v.decodedImg == nil {
		return nil
	}

	zoom := v.currentZoom()
	if zoom <= 1.0 {
		// No zoom, return full image
		return v.decodedImg
	}

	bounds := v.decodedImg.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Calculate viewport size (1/zoom of the full image)
	viewWidth := int(float64(imgWidth) / zoom)
	viewHeight := int(float64(imgHeight) / zoom)

	// Calculate viewport position based on pan (0.0-1.0)
	// Pan represents the center of the viewport
	maxOffsetX := imgWidth - viewWidth
	maxOffsetY := imgHeight - viewHeight

	offsetX := int(v.panX * float64(maxOffsetX))
	offsetY := int(v.panY * float64(maxOffsetY))

	// Clamp offsets
	if offsetX < 0 {
		offsetX = 0
	}
	if offsetY < 0 {
		offsetY = 0
	}
	if offsetX > maxOffsetX {
		offsetX = maxOffsetX
	}
	if offsetY > maxOffsetY {
		offsetY = maxOffsetY
	}

	// Create cropped image using SubImage if available
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	if si, ok := v.decodedImg.(subImager); ok {
		cropRect := image.Rect(
			bounds.Min.X+offsetX,
			bounds.Min.Y+offsetY,
			bounds.Min.X+offsetX+viewWidth,
			bounds.Min.Y+offsetY+viewHeight,
		)
		return si.SubImage(cropRect)
	}

	// Fallback: return full image if SubImage not supported
	return v.decodedImg
}

// renderFooter renders the footer help with consistent styling
func (v *ComicView) renderFooter() string {
	var help []string

	if v.isZoomed() {
		// Zoomed mode: show pan controls
		zoomPct := int(v.currentZoom() * 100)
		help = []string{
			styles.HelpKey.Render("hjkl") + styles.Help.Render(" pan"),
			styles.HelpKey.Render("+/-") + styles.Help.Render(fmt.Sprintf(" zoom (%d%%)", zoomPct)),
			styles.HelpKey.Render("0") + styles.Help.Render(" reset"),
			styles.HelpKey.Render("n/p") + styles.Help.Render(" page"),
			styles.HelpKey.Render("q") + styles.Help.Render(" back"),
		}
	} else {
		// Normal mode: show page navigation
		help = []string{
			styles.HelpKey.Render("h/l") + styles.Help.Render(" prev/next"),
			styles.HelpKey.Render("g/G") + styles.Help.Render(" first/last"),
			styles.HelpKey.Render("+/-") + styles.Help.Render(" zoom"),
			styles.HelpKey.Render("q") + styles.Help.Render(" back"),
		}
	}

	return styles.FooterBar.Width(v.width).Render(strings.Join(help, "  "))
}

// SetSize implements View
func (v *ComicView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// GetTermMode returns the terminal image mode for cleanup purposes
func (v *ComicView) GetTermMode() terminal.TermImageMode {
	return v.termMode
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
