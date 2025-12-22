package ui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all application key bindings
type KeyMap struct {
	// Navigation
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding

	// Actions
	Enter  key.Binding
	Escape key.Binding
	Quit   key.Binding
	Help   key.Binding
	Search key.Binding
	Tab    key.Binding

	// Reader specific
	NextChapter key.Binding
	PrevChapter key.Binding
	TOC         key.Binding

	// Library specific
	SortToggle key.Binding
	ViewToggle key.Binding
}

// DefaultKeyMap returns the default vim-like key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("PgUp/^u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("PgDn/^d", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("Home/g", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("End/G", "bottom"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "select"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("Esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "next field"),
		),
		NextChapter: key.NewBinding(
			key.WithKeys("n", "l"),
			key.WithHelp("n/l", "next chapter"),
		),
		PrevChapter: key.NewBinding(
			key.WithKeys("p", "h"),
			key.WithHelp("p/h", "prev chapter"),
		),
		TOC: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "table of contents"),
		),
		SortToggle: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
		ViewToggle: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "view mode"),
		),
	}
}
