package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/config"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/internal/ui/views"
	"github.com/justyntemme/webby-t/pkg/models"
)

// App is the main application model
type App struct {
	config *config.Config
	client *api.Client
	keys   KeyMap

	// Current view state
	currentView views.ViewType
	prevView    views.ViewType

	// Window dimensions
	width  int
	height int

	// User state
	user *models.User

	// View models
	loginView       views.View
	libraryView     views.View
	readerView      views.View
	collectionsView views.View
	uploadView      views.View
	comicView       views.View

	// Error/status message
	err       error
	statusMsg string
	showHelp  bool
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) *App {
	client := api.NewClient(cfg.ServerURL, cfg.Token)

	app := &App{
		config:      cfg,
		client:      client,
		keys:        DefaultKeyMap(),
		currentView: views.ViewLogin,
		width:       80,
		height:      24,
	}

	// Initialize views
	app.loginView = views.NewLoginView(client, cfg)
	app.libraryView = views.NewLibraryView(client, cfg)
	app.readerView = views.NewReaderView(client)
	app.collectionsView = views.NewCollectionsView(client)
	app.uploadView = views.NewUploadView(client)
	app.comicView = views.NewComicView(client)

	// If already authenticated, go to library
	if cfg.IsAuthenticated() {
		app.currentView = views.ViewLibrary
	}

	return app
}

// Init implements tea.Model
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.getCurrentView().Init(),
		tea.SetWindowTitle("webby-t"),
	)
}

// Update implements tea.Model
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Propagate to all views
		a.loginView.SetSize(msg.Width, msg.Height)
		a.libraryView.SetSize(msg.Width, msg.Height)
		a.readerView.SetSize(msg.Width, msg.Height)
		a.collectionsView.SetSize(msg.Width, msg.Height)
		a.uploadView.SetSize(msg.Width, msg.Height)
		a.comicView.SetSize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		// Global key handling
		switch {
		case key.Matches(msg, a.keys.Quit):
			// In reader or comic viewer, go back to library instead of quitting
			if a.currentView == views.ViewReader || a.currentView == views.ViewComic {
				return a.switchView(views.ViewLibrary)
			}
			// In library when authenticated, also allow quitting
			return a, tea.Quit

		case key.Matches(msg, a.keys.Help):
			a.showHelp = !a.showHelp
			return a, nil

		case key.Matches(msg, a.keys.Escape):
			// Handle back navigation
			if a.showHelp {
				a.showHelp = false
				return a, nil
			}
			if a.currentView == views.ViewReader {
				return a.switchView(views.ViewLibrary)
			}
			if a.currentView == views.ViewTOC {
				return a.switchView(views.ViewReader)
			}
			if a.currentView == views.ViewCollections {
				return a.switchView(views.ViewLibrary)
			}
			if a.currentView == views.ViewUpload {
				return a.switchView(views.ViewLibrary)
			}
			if a.currentView == views.ViewComic {
				return a.switchView(views.ViewLibrary)
			}
		}

	case views.LoginSuccessMsg:
		a.user = &msg.User
		a.config.Username = msg.User.Username
		return a.switchView(views.ViewLibrary)

	case views.LogoutMsg:
		a.user = nil
		a.config.ClearToken()
		return a.switchView(views.ViewLogin)

	case views.OpenBookMsg:
		// Track recently read
		_ = a.config.AddRecentlyRead(msg.Book.ID, msg.Book.Title)

		// Route CBZ files to comic viewer, everything else to text reader
		// (EPUB comics still use text reader as they have chapter-based content)
		if msg.Book.IsCBZ() {
			a.comicView.(*views.ComicView).SetBook(msg.Book)
			return a.switchView(views.ViewComic)
		}
		a.readerView.(*views.ReaderView).SetBook(msg.Book)
		return a.switchView(views.ViewReader)

	case views.ErrorMsg:
		a.err = msg.Err
		return a, nil

	case views.ClearErrorMsg:
		a.err = nil
		return a, nil

	case views.SwitchViewMsg:
		return a.switchView(msg.View)
	}

	// Delegate to current view
	var cmd tea.Cmd
	switch a.currentView {
	case views.ViewLogin, views.ViewRegister:
		a.loginView, cmd = a.loginView.Update(msg)
	case views.ViewLibrary:
		a.libraryView, cmd = a.libraryView.Update(msg)
	case views.ViewReader, views.ViewTOC:
		a.readerView, cmd = a.readerView.Update(msg)
	case views.ViewCollections:
		a.collectionsView, cmd = a.collectionsView.Update(msg)
	case views.ViewUpload:
		a.uploadView, cmd = a.uploadView.Update(msg)
	case views.ViewComic:
		a.comicView, cmd = a.comicView.Update(msg)
	}
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

// View implements tea.Model
func (a *App) View() string {
	// Main content
	var content string
	switch a.currentView {
	case views.ViewLogin, views.ViewRegister:
		content = a.loginView.View()
	case views.ViewLibrary:
		content = a.libraryView.View()
	case views.ViewReader, views.ViewTOC:
		content = a.readerView.View()
	case views.ViewCollections:
		content = a.collectionsView.View()
	case views.ViewUpload:
		content = a.uploadView.View()
	case views.ViewComic:
		content = a.comicView.View()
	default:
		content = "Unknown view"
	}

	// Add error bar if there's an error
	if a.err != nil {
		errorBar := styles.ErrorStyle.Render("Error: " + a.err.Error())
		content = lipgloss.JoinVertical(lipgloss.Left, content, errorBar)
	}

	// Add help overlay if shown
	if a.showHelp {
		content = a.renderHelp()
	}

	return content
}

// switchView changes the current view and initializes it
func (a *App) switchView(view views.ViewType) (*App, tea.Cmd) {
	// Save position when leaving the reader
	if a.currentView == views.ViewReader || a.currentView == views.ViewTOC {
		a.readerView.(*views.ReaderView).SavePositionOnExit()
	}

	a.prevView = a.currentView
	a.currentView = view
	a.err = nil

	return a, a.getCurrentView().Init()
}

// getCurrentView returns the current view model
func (a *App) getCurrentView() views.View {
	switch a.currentView {
	case views.ViewLogin, views.ViewRegister:
		return a.loginView
	case views.ViewLibrary:
		return a.libraryView
	case views.ViewReader, views.ViewTOC:
		return a.readerView
	case views.ViewCollections:
		return a.collectionsView
	case views.ViewUpload:
		return a.uploadView
	case views.ViewComic:
		return a.comicView
	default:
		return a.loginView
	}
}

// renderHelp renders the help overlay
func (a *App) renderHelp() string {
	help := styles.Dialog.Width(60).Render(
		styles.DialogTitle.Render("Keyboard Shortcuts") + "\n\n" +
			styles.HelpKey.Render("Navigation") + "\n" +
			"  j/↓     Move down\n" +
			"  k/↑     Move up\n" +
			"  g       Go to top\n" +
			"  G       Go to bottom\n" +
			"  Ctrl+d  Page down\n" +
			"  Ctrl+u  Page up\n\n" +
			styles.HelpKey.Render("Reader") + "\n" +
			"  n/l     Next chapter\n" +
			"  p/h     Previous chapter\n" +
			"  t       Table of contents\n\n" +
			styles.HelpKey.Render("Comic Viewer") + "\n" +
			"  l/n     Next page\n" +
			"  h/p     Previous page\n" +
			"  g/G     First/Last page\n\n" +
			styles.HelpKey.Render("Library") + "\n" +
			"  /       Search\n" +
			"  s       Sort\n" +
			"  v       Filter (All/Books/Comics)\n" +
			"  b/m     Books only / Comics only\n" +
			"  Enter   Open book\n\n" +
			styles.HelpKey.Render("General") + "\n" +
			"  q       Quit/Back\n" +
			"  Esc     Back\n" +
			"  ?       Toggle help\n",
	)

	// Center the help dialog
	return lipgloss.Place(
		a.width,
		a.height,
		lipgloss.Center,
		lipgloss.Center,
		help,
	)
}
