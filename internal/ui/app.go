package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/justyntemme/webby-t/internal/api"
	"github.com/justyntemme/webby-t/internal/config"
	"github.com/justyntemme/webby-t/internal/ui/styles"
	"github.com/justyntemme/webby-t/internal/ui/terminal"
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
	bookDetailsView views.View

	// Error/status message
	err       error
	statusMsg string
	showHelp  bool
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config) *App {
	client := api.NewClient(cfg.ServerURL, cfg.Token)

	// Apply saved theme from config
	styles.SetCurrentTheme(cfg.GetThemeName())

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
	app.readerView = views.NewReaderView(client, cfg)
	app.collectionsView = views.NewCollectionsView(client)
	app.uploadView = views.NewUploadView(client)
	app.comicView = views.NewComicView(client)
	app.bookDetailsView = views.NewBookDetailsView(client, cfg)

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

// Update implements tea.Model - dispatches to focused handlers
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.handleWindowSize(msg)
		return a, nil
	case tea.KeyMsg:
		if model, cmd := a.handleKeyMsg(msg); cmd != nil || model != a {
			return model, cmd
		}
	case views.LoginSuccessMsg, views.LogoutMsg, views.OpenBookMsg,
		views.ShowBookDetailsMsg, views.SwitchViewMsg, views.ErrorMsg, views.ClearErrorMsg:
		return a.handleAppMsg(msg)
	}
	return a.delegateToView(msg)
}

// handleWindowSize propagates size changes to all views
func (a *App) handleWindowSize(msg tea.WindowSizeMsg) {
	a.width = msg.Width
	a.height = msg.Height
	a.loginView.SetSize(msg.Width, msg.Height)
	a.libraryView.SetSize(msg.Width, msg.Height)
	a.readerView.SetSize(msg.Width, msg.Height)
	a.collectionsView.SetSize(msg.Width, msg.Height)
	a.uploadView.SetSize(msg.Width, msg.Height)
	a.comicView.SetSize(msg.Width, msg.Height)
	a.bookDetailsView.SetSize(msg.Width, msg.Height)
}

// handleKeyMsg processes global keybindings
func (a *App) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Quit):
		if a.currentView == views.ViewReader || a.currentView == views.ViewComic {
			return a.switchView(views.ViewLibrary)
		}
		return a, tea.Quit
	case key.Matches(msg, a.keys.Help):
		a.showHelp = !a.showHelp
		return a, nil
	case key.Matches(msg, a.keys.Escape):
		return a.handleEscapeKey()
	}
	return a, nil
}

// handleEscapeKey centralizes back-navigation logic
func (a *App) handleEscapeKey() (tea.Model, tea.Cmd) {
	if a.showHelp {
		a.showHelp = false
		return a, nil
	}
	backMap := map[views.ViewType]views.ViewType{
		views.ViewReader:      views.ViewLibrary,
		views.ViewTOC:         views.ViewReader,
		views.ViewCollections: views.ViewLibrary,
		views.ViewUpload:      views.ViewLibrary,
		views.ViewComic:       views.ViewLibrary,
		views.ViewBookDetails: views.ViewLibrary,
	}
	if dest, ok := backMap[a.currentView]; ok {
		return a.switchView(dest)
	}
	return a, nil
}

// handleAppMsg processes application-level events
func (a *App) handleAppMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case views.LoginSuccessMsg:
		a.user = &msg.User
		a.config.Username = msg.User.Username
		return a.switchView(views.ViewLibrary)
	case views.LogoutMsg:
		a.user = nil
		a.config.ClearToken()
		return a.switchView(views.ViewLogin)
	case views.OpenBookMsg:
		_ = a.config.AddRecentlyRead(msg.Book.ID, msg.Book.Title)
		if msg.Book.IsCBZ() {
			a.comicView.(*views.ComicView).SetBook(msg.Book)
			return a.switchView(views.ViewComic)
		}
		a.readerView.(*views.ReaderView).SetBook(msg.Book)
		return a.switchView(views.ViewReader)
	case views.ShowBookDetailsMsg:
		a.bookDetailsView.(*views.BookDetailsView).SetBook(msg.Book)
		return a.switchView(views.ViewBookDetails)
	case views.ErrorMsg:
		a.err = msg.Err
		return a, nil
	case views.ClearErrorMsg:
		a.err = nil
		return a, nil
	case views.SwitchViewMsg:
		return a.switchView(msg.View)
	}
	return a, nil
}

// delegateToView passes messages to the current view
func (a *App) delegateToView(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case views.ViewBookDetails:
		a.bookDetailsView, cmd = a.bookDetailsView.Update(msg)
	}
	return a, cmd
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
	case views.ViewBookDetails:
		content = a.bookDetailsView.View()
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

	// Clear terminal images when leaving views that display them
	// This prevents image artifacts from persisting across view transitions
	if a.currentView == views.ViewComic {
		termMode := a.comicView.(*views.ComicView).GetTermMode()
		terminal.ClearImagesCmd(termMode)()
	} else if a.currentView == views.ViewLibrary {
		termMode := a.libraryView.(*views.LibraryView).GetTermMode()
		if termMode != terminal.TermModeNone {
			terminal.ClearImagesCmd(termMode)()
		}
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
	case views.ViewBookDetails:
		return a.bookDetailsView
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
			"  t       Table of contents\n" +
			"  B       Add bookmark\n" +
			"  b       View bookmarks\n\n" +
			styles.HelpKey.Render("Comic Viewer") + "\n" +
			"  hjkl    Navigate pages\n" +
			"  [/]     First/Last page\n" +
			"  ←→↑↓    Pan/scroll image\n" +
			"  +/-     Zoom in/out\n" +
			"  0       Reset zoom\n\n" +
			styles.HelpKey.Render("Library") + "\n" +
			"  /       Search\n" +
			"  s       Sort\n" +
			"  v       Filter (All/Books/Comics)\n" +
			"  b/m     Books only / Comics only\n" +
			"  A       Filter by author\n" +
			"  E       Filter by series\n" +
			"  x       Clear filter\n" +
			"  i       Book details\n" +
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
