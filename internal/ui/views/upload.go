package views

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/pkg/models"
)

// UploadView displays a file picker for uploading epubs
type UploadView struct {
	client     *api.Client
	filepicker filepicker.Model
	selected   string
	uploading  bool
	result     *uploadResult
	err        error

	width  int
	height int
}

type uploadResult struct {
	book    *models.Book
	success bool
	err     error
}

// Message types
type fileSelectedMsg struct {
	path string
}

type uploadCompleteMsg struct {
	book *models.Book
	err  error
}

type clearResultMsg struct{}

// NewUploadView creates a new upload view
func NewUploadView(client *api.Client) *UploadView {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	fp := filepicker.New()
	fp.AllowedTypes = []string{".epub"}
	fp.CurrentDirectory = cwd
	fp.ShowHidden = false
	fp.ShowPermissions = false
	fp.ShowSize = true
	fp.Height = 15

	return &UploadView{
		client:     client,
		filepicker: fp,
		width:      80,
		height:     24,
	}
}

// Init implements View
func (v *UploadView) Init() tea.Cmd {
	return v.filepicker.Init()
}

// Update implements View
func (v *UploadView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if v.uploading {
				return v, nil // Can't cancel during upload
			}
			// Return to library
			return v, SwitchTo(ViewLibrary)
		case "q":
			if !v.uploading {
				return v, SwitchTo(ViewLibrary)
			}
		}

	case uploadCompleteMsg:
		v.uploading = false
		if msg.err != nil {
			v.result = &uploadResult{success: false, err: msg.err}
		} else {
			v.result = &uploadResult{book: msg.book, success: true}
		}
		// Clear result after 3 seconds
		return v, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return clearResultMsg{}
		})

	case clearResultMsg:
		v.result = nil
		v.selected = ""
		return v, nil
	}

	// Update file picker
	var cmd tea.Cmd
	v.filepicker, cmd = v.filepicker.Update(msg)

	// Check if a file was selected
	if didSelect, path := v.filepicker.DidSelectFile(msg); didSelect {
		v.selected = path
		v.uploading = true
		v.result = nil
		return v, v.uploadFile(path)
	}

	// Check if user tried to select a disabled file
	if didSelect, path := v.filepicker.DidSelectDisabledFile(msg); didSelect {
		v.err = fmt.Errorf("cannot select %s (not an epub file)", path)
		return v, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearErrorMsg{}
		})
	}

	return v, cmd
}

type clearErrorMsg struct{}

// View implements View
func (v *UploadView) View() string {
	var b strings.Builder

	// Header
	b.WriteString(styles.TitleBar.Render(" Add Book ") + "\n\n")

	// Instructions
	b.WriteString(styles.Help.Render("Navigate to an .epub file and press Enter to upload") + "\n")
	b.WriteString(styles.Help.Render("Press Esc to go back") + "\n\n")

	// Show uploading state
	if v.uploading {
		b.WriteString(styles.SecondaryText.Render(fmt.Sprintf("Uploading %s...", v.selected)) + "\n\n")
	}

	// Show result
	if v.result != nil {
		if v.result.success {
			successMsg := fmt.Sprintf("Uploaded: %s by %s", v.result.book.Title, v.result.book.Author)
			b.WriteString(styles.SuccessStyle.Render(successMsg) + "\n\n")
		} else {
			b.WriteString(styles.ErrorStyle.Render("Upload failed: "+v.result.err.Error()) + "\n\n")
		}
	}

	// Show error
	if v.err != nil {
		b.WriteString(styles.ErrorStyle.Render(v.err.Error()) + "\n\n")
	}

	// File picker
	b.WriteString(v.filepicker.View())

	// Footer
	b.WriteString("\n\n")
	help := []string{
		styles.HelpKey.Render("↑/↓") + styles.Help.Render(" navigate"),
		styles.HelpKey.Render("enter") + styles.Help.Render(" select"),
		styles.HelpKey.Render("esc") + styles.Help.Render(" back"),
	}
	b.WriteString(strings.Join(help, "  "))

	// Center the content
	content := styles.Dialog.Width(v.width - 4).Render(b.String())

	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// SetSize implements View
func (v *UploadView) SetSize(width, height int) {
	v.width = width
	v.height = height
	v.filepicker.Height = height - 15 // Leave room for header/footer
	if v.filepicker.Height < 5 {
		v.filepicker.Height = 5
	}
}

// uploadFile uploads the selected file
func (v *UploadView) uploadFile(path string) tea.Cmd {
	return func() tea.Msg {
		book, err := v.client.UploadBook(path)
		return uploadCompleteMsg{book: book, err: err}
	}
}
