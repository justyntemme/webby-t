package views

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/config"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/pkg/models"
)

var errEmptyFields = errors.New("please fill in all fields")

// loginResultMsg is the result of a login/register attempt
type loginResultMsg struct {
	user  models.User
	token string
	err   error
}

// LoginView handles login and registration
type LoginView struct {
	client *api.Client
	config *config.Config

	// Form inputs
	usernameInput textinput.Model
	emailInput    textinput.Model
	passwordInput textinput.Model

	// State
	focusIndex    int
	isRegistering bool
	loading       bool
	err           error

	// Dimensions
	width  int
	height int
}

// NewLoginView creates a new login view
func NewLoginView(client *api.Client, cfg *config.Config) *LoginView {
	// Username input
	usernameInput := textinput.New()
	usernameInput.Placeholder = "username"
	usernameInput.Focus()
	usernameInput.CharLimit = 50
	usernameInput.Width = 30

	// Email input (for registration)
	emailInput := textinput.New()
	emailInput.Placeholder = "email@example.com"
	emailInput.CharLimit = 100
	emailInput.Width = 30

	// Password input
	passwordInput := textinput.New()
	passwordInput.Placeholder = "password"
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '•'
	passwordInput.CharLimit = 100
	passwordInput.Width = 30

	return &LoginView{
		client:        client,
		config:        cfg,
		usernameInput: usernameInput,
		emailInput:    emailInput,
		passwordInput: passwordInput,
		width:         80,
		height:        24,
	}
}

// Init implements View
func (v *LoginView) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements View
func (v *LoginView) Update(msg tea.Msg) (View, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "up", "down":
			v.navigateFocus(msg.String())
			return v, nil

		case "enter":
			if v.loading {
				return v, nil
			}
			// Check if on submit button
			maxIndex := 2
			if v.isRegistering {
				maxIndex = 3
			}
			if v.focusIndex == maxIndex {
				return v, v.submit()
			}
			// Check if on toggle link
			if v.focusIndex == maxIndex+1 {
				v.toggleMode()
				return v, nil
			}
			// Move to next field
			v.navigateFocus("tab")
			return v, nil

		case "ctrl+r":
			v.toggleMode()
			return v, nil
		}

	case loginResultMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		// Save token and notify app
		v.config.Token = msg.token
		v.config.Username = msg.user.Username
		v.config.Save()
		v.client.SetToken(msg.token)
		return v, func() tea.Msg {
			return LoginSuccessMsg{User: msg.user, Token: msg.token}
		}
	}

	// Update focused input
	var cmd tea.Cmd
	switch v.focusIndex {
	case 0:
		v.usernameInput, cmd = v.usernameInput.Update(msg)
	case 1:
		if v.isRegistering {
			v.emailInput, cmd = v.emailInput.Update(msg)
		} else {
			v.passwordInput, cmd = v.passwordInput.Update(msg)
		}
	case 2:
		if v.isRegistering {
			v.passwordInput, cmd = v.passwordInput.Update(msg)
		}
	}
	cmds = append(cmds, cmd)

	return v, tea.Batch(cmds...)
}

// View implements View
func (v *LoginView) View() string {
	var b strings.Builder

	// Title
	title := "Login to Webby"
	if v.isRegistering {
		title = "Create Account"
	}
	titleStyle := styles.DialogTitle.Width(40).Align(lipgloss.Center)

	// Form fields
	b.WriteString(titleStyle.Render(title) + "\n\n")

	// Username
	label := styles.InputLabel.Render("Username")
	input := v.styleInput(v.usernameInput, 0)
	b.WriteString(label + "\n" + input + "\n\n")

	// Email (registration only)
	if v.isRegistering {
		label = styles.InputLabel.Render("Email")
		input = v.styleInput(v.emailInput, 1)
		b.WriteString(label + "\n" + input + "\n\n")
	}

	// Password
	label = styles.InputLabel.Render("Password")
	passwordIndex := 1
	if v.isRegistering {
		passwordIndex = 2
	}
	input = v.styleInput(v.passwordInput, passwordIndex)
	b.WriteString(label + "\n" + input + "\n\n")

	// Submit button
	submitIndex := 2
	if v.isRegistering {
		submitIndex = 3
	}
	buttonText := "Login"
	if v.isRegistering {
		buttonText = "Register"
	}
	if v.loading {
		buttonText = "Loading..."
	}
	button := styles.Button.Render(buttonText)
	if v.focusIndex == submitIndex {
		button = styles.ButtonFocused.Render(buttonText)
	}
	b.WriteString(button + "\n\n")

	// Toggle link
	toggleText := "Don't have an account? Register"
	if v.isRegistering {
		toggleText = "Already have an account? Login"
	}
	toggleStyle := styles.Help
	if v.focusIndex == submitIndex+1 {
		toggleStyle = styles.HelpKey
	}
	b.WriteString(toggleStyle.Render(toggleText) + "\n")

	// Error message
	if v.err != nil {
		b.WriteString("\n" + styles.ErrorStyle.Render(v.err.Error()))
	}

	// Wrap in dialog
	dialog := styles.Dialog.Width(44).Render(b.String())

	// Center on screen
	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

// SetSize implements View
func (v *LoginView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// styleInput returns the styled input field
func (v *LoginView) styleInput(input textinput.Model, index int) string {
	style := styles.InputField
	if v.focusIndex == index {
		style = styles.InputFieldFocused
	}
	return style.Render(input.View())
}

// navigateFocus moves focus between form elements
func (v *LoginView) navigateFocus(key string) {
	maxIndex := 3 // username, password, submit, toggle
	if v.isRegistering {
		maxIndex = 4 // username, email, password, submit, toggle
	}

	if key == "up" || key == "shift+tab" {
		v.focusIndex--
		if v.focusIndex < 0 {
			v.focusIndex = maxIndex
		}
	} else {
		v.focusIndex++
		if v.focusIndex > maxIndex {
			v.focusIndex = 0
		}
	}

	v.updateFocus()
}

// updateFocus updates which input has focus
func (v *LoginView) updateFocus() {
	v.usernameInput.Blur()
	v.emailInput.Blur()
	v.passwordInput.Blur()

	switch v.focusIndex {
	case 0:
		v.usernameInput.Focus()
	case 1:
		if v.isRegistering {
			v.emailInput.Focus()
		} else {
			v.passwordInput.Focus()
		}
	case 2:
		if v.isRegistering {
			v.passwordInput.Focus()
		}
	}
}

// toggleMode switches between login and registration
func (v *LoginView) toggleMode() {
	v.isRegistering = !v.isRegistering
	v.err = nil
	v.focusIndex = 0
	v.updateFocus()
}

// submit handles form submission
func (v *LoginView) submit() tea.Cmd {
	v.loading = true
	v.err = nil

	username := strings.TrimSpace(v.usernameInput.Value())
	password := v.passwordInput.Value()

	if username == "" || password == "" {
		v.loading = false
		v.err = errEmptyFields
		return nil
	}

	if v.isRegistering {
		email := strings.TrimSpace(v.emailInput.Value())
		if email == "" {
			v.loading = false
			v.err = errEmptyFields
			return nil
		}
		return v.doRegister(username, email, password)
	}

	return v.doLogin(username, password)
}

// doLogin performs the login API call
func (v *LoginView) doLogin(username, password string) tea.Cmd {
	return func() tea.Msg {
		resp, err := v.client.Login(username, password)
		if err != nil {
			return loginResultMsg{err: err}
		}
		return loginResultMsg{user: resp.User, token: resp.Token}
	}
}

// doRegister performs the registration API call
func (v *LoginView) doRegister(username, email, password string) tea.Cmd {
	return func() tea.Msg {
		resp, err := v.client.Register(username, email, password)
		if err != nil {
			return loginResultMsg{err: err}
		}
		// Registration now returns token directly
		return loginResultMsg{user: resp.User, token: resp.Token}
	}
}
