package app

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/claude"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/ui/browser"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
	"github.com/mattlab76/ccsm-go/internal/ui/delete"
	uilog "github.com/mattlab76/ccsm-go/internal/ui/log"
	"github.com/mattlab76/ccsm-go/internal/ui/mainmenu"
	"github.com/mattlab76/ccsm-go/internal/ui/newsession"
	"github.com/mattlab76/ccsm-go/internal/ui/settings"
	"github.com/mattlab76/ccsm-go/internal/ui/stats"
)

// ViewType identifies the active view.
type ViewType int

const (
	ViewMainMenu ViewType = iota
	ViewBrowser
	ViewNewSession
	ViewDelete
	ViewStats
	ViewLog
	ViewSettings
)

// SwitchViewMsg tells the app to switch to a different view.
type SwitchViewMsg struct {
	View ViewType
}

// Model is the root bubbletea model that routes to child views.
type Model struct {
	db          *db.DB
	currentView ViewType
	width       int
	height      int

	// Child models
	mainMenu   mainmenu.Model
	newSession newsession.Model
	browser    browser.Model
	deleteView delete.Model
	statsView    stats.Model
	logView      uilog.Model
	settingsView settings.Model
}

// New creates a new root model.
func New(database *db.DB) Model {
	return Model{
		db:          database,
		currentView: ViewMainMenu,
		mainMenu:    mainmenu.New(database),
	}
}

func (m Model) Init() tea.Cmd {
	return m.mainMenu.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mainMenu = m.mainMenu.SetSize(msg.Width, msg.Height)
		m.newSession = m.newSession.SetSize(msg.Width, msg.Height)
		m.browser = m.browser.SetSize(msg.Width, msg.Height)
		m.deleteView = m.deleteView.SetSize(msg.Width, msg.Height)
		m.statsView = m.statsView.SetSize(msg.Width, msg.Height)
		m.logView = m.logView.SetSize(msg.Width, msg.Height)
		m.settingsView = m.settingsView.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	// View switch messages from children.
	case mainmenu.SwitchViewMsg:
		return m.handleChildSwitch(ViewType(msg.View))
	case newsession.SwitchViewMsg:
		return m.handleChildSwitch(ViewType(msg.View))
	case browser.SwitchViewMsg:
		return m.handleChildSwitch(ViewType(msg.View))
	case delete.SwitchViewMsg:
		return m.handleChildSwitch(ViewType(msg.View))
	case stats.SwitchViewMsg:
		return m.handleChildSwitch(ViewType(msg.View))
	case uilog.SwitchViewMsg:
		return m.handleChildSwitch(ViewType(msg.View))
	case settings.SwitchViewMsg:
		return m.handleChildSwitch(ViewType(msg.View))

	// Resume messages from children.
	case mainmenu.ResumeSessionMsg:
		return m.startResume(msg.SID, msg.CWD)
	case browser.ResumeSessionMsg:
		return m.startResume(msg.SID, msg.CWD)
	}

	// Route to active view.
	switch m.currentView {
	case ViewMainMenu:
		var cmd tea.Cmd
		m.mainMenu, cmd = m.mainMenu.Update(msg)
		return m, cmd
	case ViewNewSession:
		var cmd tea.Cmd
		m.newSession, cmd = m.newSession.Update(msg)
		return m, cmd
	case ViewBrowser:
		var cmd tea.Cmd
		m.browser, cmd = m.browser.Update(msg)
		return m, cmd
	case ViewDelete:
		var cmd tea.Cmd
		m.deleteView, cmd = m.deleteView.Update(msg)
		return m, cmd
	case ViewStats:
		var cmd tea.Cmd
		m.statsView, cmd = m.statsView.Update(msg)
		return m, cmd
	case ViewLog:
		var cmd tea.Cmd
		m.logView, cmd = m.logView.Update(msg)
		return m, cmd
	case ViewSettings:
		var cmd tea.Cmd
		m.settingsView, cmd = m.settingsView.Update(msg)
		return m, cmd
	default:
		// Stub views: q returns to main menu.
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "q" {
			return m.handleChildSwitch(ViewMainMenu)
		}
	}

	return m, nil
}

func (m Model) handleChildSwitch(view ViewType) (tea.Model, tea.Cmd) {
	m.currentView = view
	switch view {
	case ViewMainMenu:
		m.mainMenu = mainmenu.New(m.db)
		m.mainMenu = m.mainMenu.SetSize(m.width, m.height)
	case ViewNewSession:
		m.newSession = newsession.New(m.db)
		m.newSession = m.newSession.SetSize(m.width, m.height)
	case ViewBrowser:
		m.browser = browser.New(m.db)
		m.browser = m.browser.SetSize(m.width, m.height)
	case ViewDelete:
		m.deleteView = delete.New(m.db)
		m.deleteView = m.deleteView.SetSize(m.width, m.height)
	case ViewStats:
		m.statsView = stats.New(m.db)
		m.statsView = m.statsView.SetSize(m.width, m.height)
	case ViewLog:
		m.logView = uilog.New(m.db)
		m.logView = m.logView.SetSize(m.width, m.height)
	case ViewSettings:
		m.settingsView = settings.New(m.db)
		m.settingsView = m.settingsView.SetSize(m.width, m.height)
	}
	return m, nil
}

func (m Model) startResume(sid, cwd string) (tea.Model, tea.Cmd) {
	// Check if claude is already running in this directory.
	if claude.IsClaudeRunningInDir(cwd) {
		m.currentView = ViewNewSession
		m.newSession = newsession.NewAlreadyRunning(m.db, cwd)
		m.newSession = m.newSession.SetSize(m.width, m.height)
		return m, nil
	}

	m.currentView = ViewNewSession
	m.newSession = newsession.NewForResume(m.db, sid)
	m.newSession = m.newSession.SetSize(m.width, m.height)

	// Create lock file before resuming.
	claude.CreateSessionLock(cwd, os.Getpid())

	if s, err := db.GetSession(m.db, sid); err == nil {
		db.LogAction(m.db, "RESUME", s.Subject+" ("+cwd+")")
		db.MoveToEnd(m.db, sid)
	}

	return m, func() tea.Msg {
		return newsession.StartResumeMsg{SID: sid, CWD: cwd}
	}
}

func (m Model) View() string {
	var content string
	var hints string

	switch m.currentView {
	case ViewMainMenu:
		content = m.mainMenu.View()
		hints = components.StatusBarItems("↑↓", "select", "Enter", "resume", "n", "new", "s", "browse", "q", "quit")
	case ViewNewSession:
		content = m.newSession.View()
		hints = components.StatusBarItems("Enter", "confirm", "Esc", "cancel")
	case ViewBrowser:
		content = m.browser.View()
		hints = components.StatusBarItems("↑↓", "navigate", "Enter", "resume", "/", "search", "q", "back")
	case ViewDelete:
		content = m.deleteView.View()
		hints = components.StatusBarItems("↑↓", "navigate", "Space", "mark", "Enter", "delete", "q", "back")
	case ViewStats:
		content = m.statsView.View()
		hints = components.StatusBarItems("↑↓", "scroll", "q", "back")
	case ViewLog:
		content = m.logView.View()
		hints = components.StatusBarItems("↑↓", "scroll", "q", "back")
	case ViewSettings:
		content = m.settingsView.View()
		hints = components.StatusBarItems("1-3", "edit", "q", "back")
	}

	return content + "\n" + components.StatusBar(hints, m.width)
}
