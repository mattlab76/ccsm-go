package mainmenu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// menuAction binds a key shortcut + label to a target view.
type menuAction struct {
	key   string
	label string
	view  int // -1 = quit (no view switch)
}

// menuActions defines the order of menu items shown below the sessions table.
// The cursor navigates through sessions first, then these items.
func menuActions() []menuAction {
	return []menuAction{
		{"n", i18n.T("menu_new"), viewNewSession},
		{"s", i18n.T("menu_resume"), viewBrowser},
		{"d", i18n.T("menu_delete"), viewDelete},
		{"i", i18n.T("menu_stats"), viewStats},
		{"l", i18n.T("menu_log"), viewLog},
		{"c", i18n.T("menu_settings"), viewSettings},
		{"q", i18n.T("menu_quit"), -1},
	}
}

// Model is the main menu bubbletea model.
// The cursor spans sessions (0..len(sessions)-1) and then menu items
// (len(sessions)..len(sessions)+len(menuActions)-1).
type Model struct {
	db       *db.DB
	sessions []model.Session
	rows     []components.TableRow
	actions  []menuAction
	width    int
	height   int
	cursor   int
	err      error
}

// New creates a new main menu model.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100, actions: menuActions()}
	m.loadSessions()
	m.cursor = 0 // first session, or first menu item if no sessions
	m.updateSelection()
	return m
}

func (m *Model) loadSessions() {
	sessions, err := db.ListSessions(m.db, 5)
	if err != nil {
		m.err = err
		return
	}
	m.sessions = sessions
	m.rows = make([]components.TableRow, len(sessions))
	for i, s := range sessions {
		m.rows[i] = components.TableRow{
			Num:     i + 1,
			Session: s,
			Status:  components.StatusOK,
		}
	}
}

func (m *Model) updateSelection() {
	for i := range m.rows {
		m.rows[i].Selected = (i == m.cursor)
	}
}

// SetSize updates the terminal dimensions.
func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	key := keyMsg.String()

	// Letter shortcuts: route directly regardless of cursor position.
	for _, a := range m.actions {
		if key == a.key {
			return m, m.activateAction(a)
		}
	}

	// Navigation across sessions + menu items.
	total := len(m.sessions) + len(m.actions)
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.updateSelection()
		}
	case "down", "j":
		if m.cursor < total-1 {
			m.cursor++
			m.updateSelection()
		}
	case "home", "g":
		m.cursor = 0
		m.updateSelection()
	case "end", "G":
		m.cursor = total - 1
		m.updateSelection()
	case "enter":
		if m.cursor < len(m.sessions) {
			s := m.sessions[m.cursor]
			return m, func() tea.Msg {
				return ResumeSessionMsg{SID: s.SID, CWD: s.CWD}
			}
		}
		actionIdx := m.cursor - len(m.sessions)
		if actionIdx >= 0 && actionIdx < len(m.actions) {
			return m, m.activateAction(m.actions[actionIdx])
		}
	case "1", "2", "3", "4", "5":
		idx := int(key[0] - '1')
		if idx < len(m.sessions) {
			s := m.sessions[idx]
			return m, func() tea.Msg {
				return ResumeSessionMsg{SID: s.SID, CWD: s.CWD}
			}
		}
	}
	return m, nil
}

func (m Model) activateAction(a menuAction) tea.Cmd {
	if a.view < 0 {
		return tea.Quit
	}
	return switchView(a.view)
}

func (m Model) View() string {
	var b strings.Builder

	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - 8
	if innerW < 56 {
		innerW = 56
	}

	// Count all sessions for header display.
	allSessions, _ := db.ListSessions(m.db, 0)
	totalCount := len(allSessions)

	// Logo + title
	b.WriteString("\n")
	b.WriteString("  " + styles.RenderLogo(
		fmt.Sprintf("%s v%s", i18n.T("title"), model.Version),
		totalCount,
	))
	b.WriteString("\n\n")

	// Separator
	b.WriteString(styles.DoubleLine(innerW))
	b.WriteString("\n")

	// Recent sessions
	if len(m.rows) > 0 {
		b.WriteString("\n")
		b.WriteString("  " + styles.TealBold.Render(i18n.T("menu_recent")))
		b.WriteString("\n\n")
		table := components.RenderTable(m.rows, components.TableCompact, w)
		for _, line := range strings.Split(table, "\n") {
			b.WriteString("  " + line + "\n")
		}
		legend := components.RenderLegend(m.rows)
		if legend != "" {
			b.WriteString(legend + "\n")
		}
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
		b.WriteString("  " + styles.Amber.Render(i18n.T("sessions_none")))
		b.WriteString("\n\n")
	}

	// Separator
	b.WriteString(styles.DoubleLine(innerW))
	b.WriteString("\n")

	// Menu
	b.WriteString("  " + styles.TealBold.Render(i18n.T("menu_action")))
	b.WriteString("\n\n")
	for i, a := range m.actions {
		selected := m.cursor == len(m.sessions)+i
		b.WriteString(menuItem(a.key, a.label, selected))
	}

	if m.err != nil {
		b.WriteString("\n  " + styles.Red.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	// Hints
	b.WriteString("\n")
	b.WriteString("  " + styles.Dim.Render("↑/↓ navigate  Enter activate  shortcut for direct action") + "\n")

	return b.String()
}

func menuItem(key, label string, selected bool) string {
	prefix := "  "
	if selected {
		prefix = styles.Violet.Render("▶ ")
	}
	return fmt.Sprintf("%s%s %s\n", prefix, styles.Green.Render("["+key+"]"), label)
}

// View type constants matching app.ViewType (avoid import cycle).
const (
	viewMainMenu   = 0
	viewBrowser    = 1
	viewNewSession = 2
	viewDelete     = 3
	viewStats      = 4
	viewLog        = 5
	viewSettings   = 6
)

// SwitchViewMsg is sent to the parent app to change views.
type SwitchViewMsg struct {
	View int
}

// ResumeSessionMsg is sent to the parent app to resume a session.
type ResumeSessionMsg struct {
	SID string
	CWD string
}

func switchView(v int) tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: v}
	}
}
