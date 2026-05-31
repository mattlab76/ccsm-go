package app

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/claude"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/browser"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
	"github.com/mattlab76/ccsm-go/internal/ui/delete"
	uilog "github.com/mattlab76/ccsm-go/internal/ui/log"
	"github.com/mattlab76/ccsm-go/internal/ui/mainmenu"
	"github.com/mattlab76/ccsm-go/internal/ui/newsession"
	"github.com/mattlab76/ccsm-go/internal/ui/settings"
	"github.com/mattlab76/ccsm-go/internal/ui/startup"
	"github.com/mattlab76/ccsm-go/internal/ui/stats"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
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
	ViewStartupCheck
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
	mainMenu     mainmenu.Model
	newSession   newsession.Model
	browser      browser.Model
	deleteView   delete.Model
	statsView    stats.Model
	logView      uilog.Model
	settingsView settings.Model
	startupView  startup.Model

	// quitConfirm gates Ctrl+C so a stray press during paste
	// (Ctrl+Shift+V) doesn't tear down ccsm without warning.
	quitConfirm bool
}

// New creates a new root model. If expired, non-dismissed sessions exist,
// it opens with the startup check; otherwise it goes straight to the
// main menu.
func New(database *db.DB) Model {
	m := Model{db: database}
	startupModel := startup.New(database)
	if startupModel.HasWork() {
		m.currentView = ViewStartupCheck
		m.startupView = startupModel
	} else {
		m.currentView = ViewMainMenu
		m.mainMenu = mainmenu.New(database)
	}
	return m
}

func (m Model) Init() tea.Cmd {
	if m.currentView == ViewStartupCheck {
		return m.startupView.Init()
	}
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
		m.startupView = m.startupView.SetSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		key := msg.String()
		// While the quit-confirm overlay is up, swallow keys so the
		// underlying view doesn't react to them.
		if m.quitConfirm {
			switch strings.ToLower(key) {
			case "y", "j", "enter", "ctrl+c":
				return m, tea.Quit
			case "n", "esc", "q":
				m.quitConfirm = false
			}
			return m, nil
		}
		// First Ctrl+C: show the confirm modal instead of quitting.
		if key == "ctrl+c" {
			m.quitConfirm = true
			return m, nil
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
	case startup.SwitchViewMsg:
		return m.handleChildSwitch(ViewType(msg.View))

	// Resume messages from children.
	case mainmenu.ResumeSessionMsg:
		return m.startResume(msg.SID, msg.CWD)
	case browser.ResumeSessionMsg:
		return m.startResume(msg.SID, msg.CWD)

	// Fork message: spin up a fresh Claude session in an existing
	// session's working directory with subject + tags pre-filled.
	case mainmenu.ForkSessionMsg:
		return m.startFork(msg.CWD, msg.Subject, msg.Tags)
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
	case ViewStartupCheck:
		var cmd tea.Cmd
		m.startupView, cmd = m.startupView.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleChildSwitch swaps the active view in and runs the new child's
// Init so any startup commands (e.g. async data loads in mainmenu)
// actually fire. Without forwarding the Init cmd, async work registered
// in a child's Init never executes when transitioning between views.
func (m Model) handleChildSwitch(view ViewType) (tea.Model, tea.Cmd) {
	m.currentView = view
	var cmd tea.Cmd
	switch view {
	case ViewMainMenu:
		m.mainMenu = mainmenu.New(m.db)
		m.mainMenu = m.mainMenu.SetSize(m.width, m.height)
		cmd = m.mainMenu.Init()
	case ViewNewSession:
		m.newSession = newsession.New(m.db)
		m.newSession = m.newSession.SetSize(m.width, m.height)
		cmd = m.newSession.Init()
	case ViewBrowser:
		m.browser = browser.New(m.db)
		m.browser = m.browser.SetSize(m.width, m.height)
		cmd = m.browser.Init()
	case ViewDelete:
		m.deleteView = delete.New(m.db)
		m.deleteView = m.deleteView.SetSize(m.width, m.height)
		cmd = m.deleteView.Init()
	case ViewStats:
		m.statsView = stats.New(m.db)
		m.statsView = m.statsView.SetSize(m.width, m.height)
		cmd = m.statsView.Init()
	case ViewLog:
		m.logView = uilog.New(m.db)
		m.logView = m.logView.SetSize(m.width, m.height)
		cmd = m.logView.Init()
	case ViewSettings:
		m.settingsView = settings.New(m.db)
		m.settingsView = m.settingsView.SetSize(m.width, m.height)
		cmd = m.settingsView.Init()
	case ViewStartupCheck:
		m.startupView = startup.New(m.db)
		m.startupView = m.startupView.SetSize(m.width, m.height)
		cmd = m.startupView.Init()
	}
	return m, cmd
}

func (m Model) startResume(sid, cwd string) (tea.Model, tea.Cmd) {
	// Self-heal: if the stored cwd doesn't match where Claude actually
	// kept the transcript, rewrite the row before launching. Otherwise
	// `claude --resume` would print "No conversation found" because it
	// looks inside the stored cwd's project directory.
	if resolved := claude.ResolveSessionCWD(sid, cwd); resolved != cwd {
		if s, err := db.GetSession(m.db, sid); err == nil {
			s.CWD = resolved
			_ = db.SaveSession(m.db, s)
			db.LogAction(m.db, "RESUME", "auto-fixed cwd: "+cwd+" → "+resolved)
		}
		cwd = resolved
	}

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

// startFork launches a brand-new Claude session in cwd with subject +
// tags pre-filled from the source session. Distinct from startResume:
// new SID is assigned by Claude, no `--resume` flag is passed.
func (m Model) startFork(cwd, subject, tags string) (tea.Model, tea.Cmd) {
	if claude.IsClaudeRunningInDir(cwd) {
		m.currentView = ViewNewSession
		m.newSession = newsession.NewAlreadyRunning(m.db, cwd)
		m.newSession = m.newSession.SetSize(m.width, m.height)
		return m, nil
	}

	m.currentView = ViewNewSession
	m.newSession = newsession.NewForFork(m.db, cwd, subject, tags)
	m.newSession = m.newSession.SetSize(m.width, m.height)

	claude.CreateSessionLock(cwd, os.Getpid())
	db.LogAction(m.db, model.ActionNew, "fork: "+subject+" ("+cwd+")")

	return m, claude.StartSession(cwd)
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
		hints = components.StatusBarItems("↑↓", "nav", "Space", "mark", "a", "all expired", "A", "all", "Enter", "delete", "q", "back")
	case ViewStats:
		content = m.statsView.View()
		hints = components.StatusBarItems("↑↓", "scroll", "q", "back")
	case ViewLog:
		content = m.logView.View()
		hints = components.StatusBarItems("↑↓", "scroll", "q", "back")
	case ViewSettings:
		content = m.settingsView.View()
		hints = components.StatusBarItems("1-7", "edit", "q", "back")
	case ViewStartupCheck:
		content = m.startupView.View()
		hints = components.StatusBarItems("↑↓", "nav", "Enter", "activate", "p", "purge", "d", "dismiss", "s/Esc", "skip")
	}

	full := content + "\n" + components.StatusBar(hints, m.width)
	if m.quitConfirm {
		// Overlay the modal beneath the current content. Replacing the
		// whole screen would lose context; keeping it visible reminds
		// the user where they were about to lose their work.
		return full + "\n\n" + renderQuitConfirm(m.width)
	}
	return full
}

// renderQuitConfirm builds a centered red-bordered modal asking whether
// to really exit ccsm. Three keys answer it: y/Enter/Ctrl+C confirm,
// n/Esc/q dismiss.
func renderQuitConfirm(width int) string {
	title := i18n.T("quit_title")
	warning := i18n.T("quit_warning")
	hint := i18n.T("quit_hint")

	innerW := 58
	pad := func(s string, w int) string {
		runes := []rune(s)
		if len(runes) >= w {
			return string(runes[:w])
		}
		return s + strings.Repeat(" ", w-len(runes))
	}
	bar := strings.Repeat("═", innerW+2)

	lines := []string{
		styles.Red.Render("╔" + bar + "╗"),
		styles.Red.Render("║ " + pad("", innerW) + " ║"),
		styles.Red.Render("║ ") + styles.Bold.Render(pad(title, innerW)) + styles.Red.Render(" ║"),
		styles.Red.Render("║ " + pad("", innerW) + " ║"),
		styles.Red.Render("║ ") + styles.Dim.Render(pad(warning, innerW)) + styles.Red.Render(" ║"),
		styles.Red.Render("║ " + pad("", innerW) + " ║"),
		styles.Red.Render("║ ") + pad(hint, innerW) + styles.Red.Render(" ║"),
		styles.Red.Render("║ " + pad("", innerW) + " ║"),
		styles.Red.Render("╚" + bar + "╝"),
	}

	leftPad := 4
	if width > innerW+6 {
		leftPad = (width - innerW - 4) / 2
	}
	indent := strings.Repeat(" ", leftPad)
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(indent + l + "\n")
	}
	return b.String()
}
