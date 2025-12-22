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
)

// Dimensions returns styled content with proper dimensions
func Dimensions(width, height int) lipgloss.Style {
	return lipgloss.NewStyle().
		Width(width).
		Height(height)
}
