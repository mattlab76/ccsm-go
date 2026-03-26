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

// Model is the main menu bubbletea model.
type Model struct {
	db       *db.DB
	sessions []model.Session
	rows     []components.TableRow
	width    int
	height   int
	cursor   int // -1 = no session selected, 0-4 = session row
	err      error
}

// New creates a new main menu model.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100, cursor: -1}
	m.loadSessions()
	if len(m.sessions) > 0 {
		m.cursor = 0
		m.updateSelection()
	}
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "n":
			return m, switchView(viewNewSession)
		case "s":
			return m, switchView(viewBrowser)
		case "d":
			return m, switchView(viewDelete)
		case "i":
			return m, switchView(viewStats)
		case "l":
			return m, switchView(viewLog)
		case "c":
			return m, switchView(viewSettings)
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateSelection()
			}
		case "down", "j":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
				m.updateSelection()
			}
		case "enter":
			if m.cursor >= 0 && m.cursor < len(m.sessions) {
				s := m.sessions[m.cursor]
				return m, func() tea.Msg {
					return ResumeSessionMsg{SID: s.SID, CWD: s.CWD}
				}
			}
		case "1", "2", "3", "4", "5":
			idx := int(msg.String()[0] - '1')
			if idx < len(m.sessions) {
				s := m.sessions[idx]
				return m, func() tea.Msg {
					return ResumeSessionMsg{SID: s.SID, CWD: s.CWD}
				}
			}
			return m, nil
		}
	}
	return m, nil
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
	b.WriteString(menuItem("n", i18n.T("menu_new")))
	b.WriteString(menuItem("s", i18n.T("menu_resume")))
	b.WriteString(menuItem("d", i18n.T("menu_delete")))
	b.WriteString(menuItem("i", i18n.T("menu_stats")))
	b.WriteString(menuItem("l", i18n.T("menu_log")))
	b.WriteString(menuItem("c", i18n.T("menu_settings")))
	b.WriteString(menuItem("q", i18n.T("menu_quit")))

	if m.err != nil {
		b.WriteString("\n  " + styles.Red.Render(fmt.Sprintf("Error: %v", m.err)))
	}

	// Hints
	b.WriteString("\n")
	if len(m.sessions) > 0 {
		b.WriteString("  " + styles.Dim.Render("↑/↓ select session  Enter resume  or press key for action") + "\n")
	}

	return b.String()
}

func menuItem(key, label string) string {
	return fmt.Sprintf("  %s %s\n", styles.Green.Render("["+key+"]"), label)
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
