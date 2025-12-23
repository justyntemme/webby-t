package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary     = lipgloss.Color("#7C3AED") // Purple
	Secondary   = lipgloss.Color("#06B6D4") // Cyan
	Success     = lipgloss.Color("#10B981") // Green
	Warning     = lipgloss.Color("#F59E0B") // Amber
	Error       = lipgloss.Color("#EF4444") // Red
	Muted       = lipgloss.Color("#6B7280") // Gray
	Background  = lipgloss.Color("#1F2937") // Dark gray
	Foreground  = lipgloss.Color("#F9FAFB") // Light gray
	Border      = lipgloss.Color("#374151") // Gray border

	// Base styles
	App = lipgloss.NewStyle().
		Background(Background)

	// Title bar
	TitleBar = lipgloss.NewStyle().
		Foreground(Foreground).
		Background(Primary).
		Padding(0, 1).
		Bold(true)

	// Status bar at bottom
	StatusBar = lipgloss.NewStyle().
		Foreground(Muted).
		Background(Background).
		Padding(0, 1)

	// Help text
	Help = lipgloss.NewStyle().
		Foreground(Muted)

	HelpKey = lipgloss.NewStyle().
		Foreground(Secondary).
		Bold(true)

	// Muted text style
	MutedText = lipgloss.NewStyle().
		Foreground(Muted)

	// Secondary text style
	SecondaryText = lipgloss.NewStyle().
		Foreground(Secondary)

	// Error message
	ErrorStyle = lipgloss.NewStyle().
		Foreground(Error).
		Bold(true).
		Padding(0, 1)

	// Success message
	SuccessStyle = lipgloss.NewStyle().
		Foreground(Success).
		Bold(true).
		Padding(0, 1)

	// Input field
	InputLabel = lipgloss.NewStyle().
		Foreground(Foreground).
		Bold(true)

	InputField = lipgloss.NewStyle().
		Foreground(Foreground).
		Background(Background).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border).
		Padding(0, 1)

	InputFieldFocused = InputField.
		BorderForeground(Primary)

	// List styles
	ListItem = lipgloss.NewStyle().
		Foreground(Foreground).
		Padding(0, 2)

	ListItemSelected = lipgloss.NewStyle().
		Foreground(Foreground).
		Background(Primary).
		Padding(0, 2).
		Bold(true)

	ListItemDimmed = lipgloss.NewStyle().
		Foreground(Muted).
		Padding(0, 2)

	// Reader styles
	ReaderContent = lipgloss.NewStyle().
		Foreground(Foreground).
		Padding(1, 2)

	ReaderHeader = lipgloss.NewStyle().
		Foreground(Foreground).
		Background(Primary).
		Padding(0, 1).
		Bold(true)

	ReaderProgress = lipgloss.NewStyle().
		Foreground(Secondary).
		Align(lipgloss.Right)

	// Dialog/Modal styles
	Dialog = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(1, 2)

	DialogTitle = lipgloss.NewStyle().
		Foreground(Primary).
		Bold(true).
		MarginBottom(1)

	// Button styles
	Button = lipgloss.NewStyle().
		Foreground(Foreground).
		Background(Muted).
		Padding(0, 2).
		MarginRight(1)

	ButtonFocused = lipgloss.NewStyle().
		Foreground(Foreground).
		Background(Primary).
		Padding(0, 2).
		MarginRight(1).
		Bold(true)

	// Book info styles
	BookTitle = lipgloss.NewStyle().
		Foreground(Foreground).
		Bold(true)

	BookAuthor = lipgloss.NewStyle().
		Foreground(Secondary)

	BookSeries = lipgloss.NewStyle().
		Foreground(Muted).
		Italic(true)

	// Content type badges
	BadgeBook = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1F2937")).
		Background(lipgloss.Color("#10B981")). // Green
		Padding(0, 1).
		Bold(true)

	BadgeComic = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1F2937")).
		Background(lipgloss.Color("#F59E0B")). // Amber
		Padding(0, 1).
		Bold(true)

	// ========================================
	// Reusable Panel/Layout Styles
	// ========================================

	// HeaderBar - consistent header across all views
	HeaderBar = lipgloss.NewStyle().
		Foreground(Foreground).
		Background(Primary).
		Padding(0, 1).
		Bold(true)

	// FooterBar - consistent footer/help bar across all views
	FooterBar = lipgloss.NewStyle().
		Foreground(Muted).
		Background(lipgloss.Color("#111827")). // Darker than main background
		Padding(0, 1)

	// ContentPanel - main content area with subtle border
	ContentPanel = lipgloss.NewStyle().
		Foreground(Foreground).
		Padding(0, 1)

	// ContentPanelBordered - content area with visible border
	ContentPanelBordered = lipgloss.NewStyle().
		Foreground(Foreground).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border).
		Padding(0, 1)

	// InfoPanel - for displaying metadata/details
	InfoPanel = lipgloss.NewStyle().
		Foreground(Foreground).
		Border(lipgloss.NormalBorder()).
		BorderForeground(Border).
		Padding(1, 2)

	// StatusLine - single line status indicator
	StatusLine = lipgloss.NewStyle().
		Foreground(Secondary).
		Background(lipgloss.Color("#111827")).
		Padding(0, 1)

	// Divider line style
	Divider = lipgloss.NewStyle().
		Foreground(Border)
)

// Dimensions returns styled content with proper dimensions
func Dimensions(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Height(height)
}

// RenderHeader renders a consistent header bar with left and right content
func RenderHeader(left, right string, width int) string {
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := width - leftWidth - rightWidth
	if gap < 0 {
		gap = 0
	}
	// Use spaces to create the gap, with consistent background
	gapStr := HeaderBar.Render(repeat(" ", gap))
	return HeaderBar.Width(width).Render(left + gapStr + right)
}

// RenderFooter renders a consistent footer bar
func RenderFooter(content string, width int) string {
	return FooterBar.Width(width).Render(content)
}

// RenderDivider renders a horizontal divider line
func RenderDivider(width int) string {
	return Divider.Render(repeat("─", width))
}

// repeat returns a string of n repeated characters
func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// Layout constants
const (
	HeaderHeight = 1 // Fixed header height in lines
	FooterHeight = 1 // Fixed footer height in lines
)

// RenderLayout creates a consistent view layout with header, content, and footer
// It ensures proper spacing and alignment across all views
func RenderLayout(header, content, footer string, width, height int) string {
	// Calculate content area height
	contentHeight := height - HeaderHeight - FooterHeight - 2 // -2 for newlines

	// Ensure header spans full width
	headerLine := HeaderBar.Width(width).Render(header)

	// Content area with fixed height
	contentArea := lipgloss.NewStyle().
		Width(width).
		Height(contentHeight).
		Render(content)

	// Footer spans full width
	footerLine := FooterBar.Width(width).Render(footer)

	// Join vertically
	return lipgloss.JoinVertical(lipgloss.Left, headerLine, contentArea, footerLine)
}

// RenderCenteredContent centers content within the available space
func RenderCenteredContent(content string, width, height int) string {
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}

// TruncateText truncates a string to maxWidth visible characters with ellipsis
// Uses lipgloss.Width for accurate measurement of styled text
func TruncateText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	runes := []rune(text)
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	for i := len(runes) - 1; i >= 0; i-- {
		candidate := string(runes[:i]) + "..."
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return "..."
}
