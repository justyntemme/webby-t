package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/pkg/models"
)

// CollectionsView displays and manages collections
type CollectionsView struct {
	client *api.Client

	// Collections
	collections []models.Collection
	cursor      int

	// State
	loading      bool
	err          error
	createMode   bool
	createInput  textinput.Model

	// Dimensions
	width  int
	height int
}

// NewCollectionsView creates a new collections view
func NewCollectionsView(client *api.Client) *CollectionsView {
	createInput := textinput.New()
	createInput.Placeholder = "Collection name..."
	createInput.CharLimit = 100
	createInput.Width = 40

	return &CollectionsView{
		client:      client,
		createInput: createInput,
		width:       80,
		height:      24,
	}
}

// Message types
type collectionsLoadedMsg struct {
	collections []models.Collection
	err         error
}

type collectionCreatedMsg struct {
	collection *models.Collection
	err        error
}

// Init implements View
func (v *CollectionsView) Init() tea.Cmd {
	v.loading = true
	return v.loadCollections()
}

// Update implements View
func (v *CollectionsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Create mode
		if v.createMode {
			switch msg.String() {
			case "esc":
				v.createMode = false
				v.createInput.Blur()
				v.createInput.SetValue("")
				return v, nil
			case "enter":
				name := strings.TrimSpace(v.createInput.Value())
				if name != "" {
					v.createMode = false
					v.createInput.Blur()
					return v, v.createCollection(name)
				}
				return v, nil
			default:
				var cmd tea.Cmd
				v.createInput, cmd = v.createInput.Update(msg)
				return v, cmd
			}
		}

		// Normal mode
		switch msg.String() {
		case "j", "down":
			if v.cursor < len(v.collections)-1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "c":
			// Create new collection
			v.createMode = true
			v.createInput.Focus()
			v.createInput.SetValue("")
			return v, textinput.Blink
		case "d":
			// Delete collection
			if len(v.collections) > 0 {
				return v, v.deleteCollection(v.collections[v.cursor].ID)
			}
		case "enter":
			// Select collection (could filter library by this collection)
			if len(v.collections) > 0 {
				// Return to library with filter
				return v, SwitchTo(ViewLibrary)
			}
		case "r":
			// Refresh
			return v, v.loadCollections()
		}

	case collectionsLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.collections = msg.collections
		v.err = nil
		if v.cursor >= len(v.collections) {
			v.cursor = max(0, len(v.collections)-1)
		}
		return v, nil

	case collectionCreatedMsg:
		if msg.err != nil {
			v.err = msg.err
			return v, nil
		}
		v.createInput.SetValue("")
		return v, v.loadCollections()
	}

	return v, nil
}

// View implements View
func (v *CollectionsView) View() string {
	var b strings.Builder

	// Header
	b.WriteString(styles.TitleBar.Render(" Collections ") + "\n\n")

	// Create mode input
	if v.createMode {
		b.WriteString(styles.InputLabel.Render("New Collection:") + "\n")
		b.WriteString(styles.InputFieldFocused.Render(v.createInput.View()) + "\n\n")
	}

	// Loading state
	if v.loading {
		content := lipgloss.Place(
			v.width,
			v.height-4,
			lipgloss.Center,
			lipgloss.Center,
			styles.MutedText.Render("Loading collections..."),
		)
		b.WriteString(content)
		return b.String()
	}

	// Error state
	if v.err != nil {
		b.WriteString(styles.ErrorStyle.Render("Error: "+v.err.Error()) + "\n\n")
	}

	// Empty state
	if len(v.collections) == 0 {
		b.WriteString(styles.MutedText.Render("No collections yet. Press 'c' to create one.") + "\n")
	} else {
		// Collection list
		for i, col := range v.collections {
			line := col.Name
			if i == v.cursor {
				b.WriteString(styles.ListItemSelected.Render("▸ "+line) + "\n")
			} else {
				b.WriteString(styles.ListItem.Render("  "+line) + "\n")
			}
		}
	}

	// Footer
	b.WriteString("\n")
	help := []string{
		styles.HelpKey.Render("c") + styles.Help.Render(" create"),
		styles.HelpKey.Render("d") + styles.Help.Render(" delete"),
		styles.HelpKey.Render("esc") + styles.Help.Render(" back"),
	}
	b.WriteString(strings.Join(help, "  "))

	return b.String()
}

// SetSize implements View
func (v *CollectionsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// loadCollections fetches collections from the API
func (v *CollectionsView) loadCollections() tea.Cmd {
	return func() tea.Msg {
		resp, err := v.client.ListCollections()
		if err != nil {
			return collectionsLoadedMsg{err: err}
		}
		return collectionsLoadedMsg{collections: resp.Collections}
	}
}

// createCollection creates a new collection
func (v *CollectionsView) createCollection(name string) tea.Cmd {
	return func() tea.Msg {
		col, err := v.client.CreateCollection(name)
		if err != nil {
			return collectionCreatedMsg{err: err}
		}
		return collectionCreatedMsg{collection: col}
	}
}

// deleteCollection deletes a collection
func (v *CollectionsView) deleteCollection(id string) tea.Cmd {
	return func() tea.Msg {
		err := v.client.DeleteCollection(id)
		if err != nil {
			return collectionsLoadedMsg{err: err}
		}
		// Reload collections
		resp, err := v.client.ListCollections()
		if err != nil {
			return collectionsLoadedMsg{err: err}
		}
		return collectionsLoadedMsg{collections: resp.Collections}
	}
}
