package styles

import "github.com/charmbracelet/lipgloss"

// Theme represents a color scheme for the application
type Theme struct {
	Name        string
	Description string

	// Core colors
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Background lipgloss.Color
	Foreground lipgloss.Color

	// Semantic colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Muted   lipgloss.Color

	// UI element colors
	Border          lipgloss.Color
	Selection       lipgloss.Color
	SelectionText   lipgloss.Color
	BadgeBook       lipgloss.Color
	BadgeBookText   lipgloss.Color
	BadgeComic      lipgloss.Color
	BadgeComicText  lipgloss.Color
}

// Built-in themes
var (
	// DarkTheme is the default dark theme
	DarkTheme = Theme{
		Name:            "dark",
		Description:     "Dark theme (default)",
		Primary:         lipgloss.Color("#7C3AED"),
		Secondary:       lipgloss.Color("#06B6D4"),
		Background:      lipgloss.Color("#1F2937"),
		Foreground:      lipgloss.Color("#F9FAFB"),
		Success:         lipgloss.Color("#10B981"),
		Warning:         lipgloss.Color("#F59E0B"),
		Error:           lipgloss.Color("#EF4444"),
		Muted:           lipgloss.Color("#6B7280"),
		Border:          lipgloss.Color("#374151"),
		Selection:       lipgloss.Color("#7C3AED"),
		SelectionText:   lipgloss.Color("#F9FAFB"),
		BadgeBook:       lipgloss.Color("#10B981"),
		BadgeBookText:   lipgloss.Color("#1F2937"),
		BadgeComic:      lipgloss.Color("#F59E0B"),
		BadgeComicText:  lipgloss.Color("#1F2937"),
	}

	// LightTheme is a light color scheme
	LightTheme = Theme{
		Name:            "light",
		Description:     "Light theme",
		Primary:         lipgloss.Color("#7C3AED"),
		Secondary:       lipgloss.Color("#0891B2"),
		Background:      lipgloss.Color("#FFFFFF"),
		Foreground:      lipgloss.Color("#1F2937"),
		Success:         lipgloss.Color("#059669"),
		Warning:         lipgloss.Color("#D97706"),
		Error:           lipgloss.Color("#DC2626"),
		Muted:           lipgloss.Color("#9CA3AF"),
		Border:          lipgloss.Color("#E5E7EB"),
		Selection:       lipgloss.Color("#7C3AED"),
		SelectionText:   lipgloss.Color("#FFFFFF"),
		BadgeBook:       lipgloss.Color("#059669"),
		BadgeBookText:   lipgloss.Color("#FFFFFF"),
		BadgeComic:      lipgloss.Color("#D97706"),
		BadgeComicText:  lipgloss.Color("#FFFFFF"),
	}

	// SolarizedTheme is based on the Solarized color scheme
	SolarizedTheme = Theme{
		Name:            "solarized",
		Description:     "Solarized dark theme",
		Primary:         lipgloss.Color("#268BD2"),
		Secondary:       lipgloss.Color("#2AA198"),
		Background:      lipgloss.Color("#002B36"),
		Foreground:      lipgloss.Color("#839496"),
		Success:         lipgloss.Color("#859900"),
		Warning:         lipgloss.Color("#B58900"),
		Error:           lipgloss.Color("#DC322F"),
		Muted:           lipgloss.Color("#586E75"),
		Border:          lipgloss.Color("#073642"),
		Selection:       lipgloss.Color("#268BD2"),
		SelectionText:   lipgloss.Color("#FDF6E3"),
		BadgeBook:       lipgloss.Color("#859900"),
		BadgeBookText:   lipgloss.Color("#002B36"),
		BadgeComic:      lipgloss.Color("#B58900"),
		BadgeComicText:  lipgloss.Color("#002B36"),
	}

	// NordTheme is based on the Nord color palette
	NordTheme = Theme{
		Name:            "nord",
		Description:     "Nord theme",
		Primary:         lipgloss.Color("#88C0D0"),
		Secondary:       lipgloss.Color("#81A1C1"),
		Background:      lipgloss.Color("#2E3440"),
		Foreground:      lipgloss.Color("#ECEFF4"),
		Success:         lipgloss.Color("#A3BE8C"),
		Warning:         lipgloss.Color("#EBCB8B"),
		Error:           lipgloss.Color("#BF616A"),
		Muted:           lipgloss.Color("#4C566A"),
		Border:          lipgloss.Color("#3B4252"),
		Selection:       lipgloss.Color("#88C0D0"),
		SelectionText:   lipgloss.Color("#2E3440"),
		BadgeBook:       lipgloss.Color("#A3BE8C"),
		BadgeBookText:   lipgloss.Color("#2E3440"),
		BadgeComic:      lipgloss.Color("#EBCB8B"),
		BadgeComicText:  lipgloss.Color("#2E3440"),
	}

	// GruvboxTheme is based on the Gruvbox color scheme
	GruvboxTheme = Theme{
		Name:            "gruvbox",
		Description:     "Gruvbox dark theme",
		Primary:         lipgloss.Color("#D79921"),
		Secondary:       lipgloss.Color("#458588"),
		Background:      lipgloss.Color("#282828"),
		Foreground:      lipgloss.Color("#EBDBB2"),
		Success:         lipgloss.Color("#98971A"),
		Warning:         lipgloss.Color("#D79921"),
		Error:           lipgloss.Color("#CC241D"),
		Muted:           lipgloss.Color("#928374"),
		Border:          lipgloss.Color("#3C3836"),
		Selection:       lipgloss.Color("#D79921"),
		SelectionText:   lipgloss.Color("#282828"),
		BadgeBook:       lipgloss.Color("#98971A"),
		BadgeBookText:   lipgloss.Color("#282828"),
		BadgeComic:      lipgloss.Color("#D79921"),
		BadgeComicText:  lipgloss.Color("#282828"),
	}

	// BuiltinThemes is a list of all available built-in themes
	BuiltinThemes = []Theme{
		DarkTheme,
		LightTheme,
		SolarizedTheme,
		NordTheme,
		GruvboxTheme,
	}

	// currentTheme holds the active theme
	currentTheme = DarkTheme
)

// GetTheme returns a theme by name, or the default theme if not found
func GetTheme(name string) Theme {
	for _, t := range BuiltinThemes {
		if t.Name == name {
			return t
		}
	}
	return DarkTheme
}

// GetThemeNames returns a list of all available theme names
func GetThemeNames() []string {
	names := make([]string, len(BuiltinThemes))
	for i, t := range BuiltinThemes {
		names[i] = t.Name
	}
	return names
}

// CurrentTheme returns the currently active theme
func CurrentTheme() Theme {
	return currentTheme
}

// SetCurrentTheme sets the active theme by name
func SetCurrentTheme(name string) {
	currentTheme = GetTheme(name)
	ApplyTheme(currentTheme)
}

// NextTheme cycles to the next theme and returns its name
func NextTheme() string {
	for i, t := range BuiltinThemes {
		if t.Name == currentTheme.Name {
			nextIdx := (i + 1) % len(BuiltinThemes)
			SetCurrentTheme(BuiltinThemes[nextIdx].Name)
			return BuiltinThemes[nextIdx].Name
		}
	}
	return currentTheme.Name
}

// ApplyTheme updates all global styles to use the given theme's colors
func ApplyTheme(theme Theme) {
	// Update color variables
	Primary = theme.Primary
	Secondary = theme.Secondary
	Success = theme.Success
	Warning = theme.Warning
	Error = theme.Error
	Muted = theme.Muted
	Background = theme.Background
	Foreground = theme.Foreground
	Border = theme.Border

	// Update styles
	App = lipgloss.NewStyle().
		Background(theme.Background)

	TitleBar = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(theme.Primary).
		Padding(0, 1).
		Bold(true)

	StatusBar = lipgloss.NewStyle().
		Foreground(theme.Muted).
		Background(theme.Background).
		Padding(0, 1)

	Help = lipgloss.NewStyle().
		Foreground(theme.Muted)

	HelpKey = lipgloss.NewStyle().
		Foreground(theme.Secondary).
		Bold(true)

	MutedText = lipgloss.NewStyle().
		Foreground(theme.Muted)

	SecondaryText = lipgloss.NewStyle().
		Foreground(theme.Secondary)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(theme.Error).
		Bold(true).
		Padding(0, 1)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(theme.Success).
		Bold(true).
		Padding(0, 1)

	InputLabel = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Bold(true)

	InputField = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(theme.Background).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Padding(0, 1)

	InputFieldFocused = InputField.
		BorderForeground(theme.Primary)

	ListItem = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Padding(0, 2)

	ListItemSelected = lipgloss.NewStyle().
		Foreground(theme.SelectionText).
		Background(theme.Selection).
		Padding(0, 2).
		Bold(true)

	ListItemDimmed = lipgloss.NewStyle().
		Foreground(theme.Muted).
		Padding(0, 2)

	ReaderContent = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Padding(1, 2)

	ReaderHeader = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(theme.Primary).
		Padding(0, 1).
		Bold(true)

	ReaderProgress = lipgloss.NewStyle().
		Foreground(theme.Secondary).
		Align(lipgloss.Right)

	Dialog = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2)

	DialogTitle = lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		MarginBottom(1)

	Button = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(theme.Muted).
		Padding(0, 2).
		MarginRight(1)

	ButtonFocused = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Background(theme.Primary).
		Padding(0, 2).
		MarginRight(1).
		Bold(true)

	BookTitle = lipgloss.NewStyle().
		Foreground(theme.Foreground).
		Bold(true)

	BookAuthor = lipgloss.NewStyle().
		Foreground(theme.Secondary)

	BookSeries = lipgloss.NewStyle().
		Foreground(theme.Muted).
		Italic(true)

	BadgeBook = lipgloss.NewStyle().
		Foreground(theme.BadgeBookText).
		Background(theme.BadgeBook).
		Padding(0, 1).
		Bold(true)

	BadgeComic = lipgloss.NewStyle().
		Foreground(theme.BadgeComicText).
		Background(theme.BadgeComic).
		Padding(0, 1).
		Bold(true)
}

// init applies the default theme on package load
func init() {
	ApplyTheme(DarkTheme)
}
