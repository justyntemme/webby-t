package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/justyntemme/webby-t/pkg/models"
)

// ViewType represents different screens in the application
type ViewType int

const (
	ViewLogin ViewType = iota
	ViewRegister
	ViewLibrary
	ViewReader
	ViewTOC
	ViewCollections
	ViewUpload
	ViewSettings
	ViewComic
)

// String returns the name of the view
func (v ViewType) String() string {
	switch v {
	case ViewLogin:
		return "Login"
	case ViewRegister:
		return "Register"
	case ViewLibrary:
		return "Library"
	case ViewReader:
		return "Reader"
	case ViewTOC:
		return "Table of Contents"
	case ViewCollections:
		return "Collections"
	case ViewUpload:
		return "Upload"
	case ViewSettings:
		return "Settings"
	case ViewComic:
		return "Comic Viewer"
	default:
		return "Unknown"
	}
}

// View is the interface that all views must implement
type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	View() string
	SetSize(width, height int)
}

// Message types for inter-view communication

// LoginSuccessMsg is sent when login succeeds
type LoginSuccessMsg struct {
	User  models.User
	Token string
}

// LogoutMsg is sent when user logs out
type LogoutMsg struct{}

// OpenBookMsg is sent when a book is selected to read
type OpenBookMsg struct {
	Book models.Book
}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}

// ClearErrorMsg clears the current error
type ClearErrorMsg struct{}

// SwitchViewMsg requests a view switch
type SwitchViewMsg struct {
	View ViewType
}

// Helper functions to create messages

// SendError creates an error message command
func SendError(err error) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{Err: err}
	}
}

// ClearError creates a command to clear errors
func ClearError() tea.Cmd {
	return func() tea.Msg {
		return ClearErrorMsg{}
	}
}

// SwitchTo creates a command to switch views
func SwitchTo(view ViewType) tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: view}
	}
}
