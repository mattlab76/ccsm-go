package delete

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/claude"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// SwitchViewMsg is sent to the parent app to change views.
type SwitchViewMsg struct {
	View int
}

type state int

const (
	stateSelect  state = iota // Navigate + space to mark
	stateConfirm              // Confirm deletion
	stateDone
)

// Model handles session deletion.
type Model struct {
	db       *db.DB
	sessions []model.Session
	rows     []components.TableRow
	width    int
	height   int
	state    state
	cursor   int            // 0-based cursor position
	marked   map[int]bool   // marked for deletion (0-based indices)
	message  string
}

// New creates a new delete model.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100, state: stateSelect, marked: make(map[int]bool)}
	m.loadSessions()
	return m
}

func (m *Model) loadSessions() {
	sessions, err := db.ListSessions(m.db, 0)
	if err != nil {
		m.message = err.Error()
		return
	}
	m.sessions = sessions
	m.rows = make([]components.TableRow, len(sessions))
	for i, s := range sessions {
		status := components.StatusOK
		if !claude.IsSessionValid(s.SID) {
			status = components.StatusExpired
		} else if !dirExists(s.CWD) {
			status = components.StatusMissingDir
		}
		m.rows[i] = components.TableRow{
			Num:     i + 1,
			Session: s,
			Status:  status,
		}
	}
	m.cursor = 0
	m.updateSelection()
}

func (m *Model) updateSelection() {
	for i := range m.rows {
		m.rows[i].Selected = (i == m.cursor)
	}
}

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
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "ctrl+c":
		return m, tea.Quit
	}

	switch m.state {
	case stateSelect:
		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateSelection()
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
				m.updateSelection()
			}
		case " ": // Space toggles mark
			if m.cursor >= 0 && m.cursor < len(m.sessions) {
				if m.marked[m.cursor] {
					delete(m.marked, m.cursor)
				} else {
					m.marked[m.cursor] = true
				}
			}
		case "enter":
			if len(m.marked) == 0 {
				// If nothing marked, mark current cursor position.
				if m.cursor >= 0 && m.cursor < len(m.sessions) {
					m.marked[m.cursor] = true
				}
			}
			if len(m.marked) > 0 {
				m.state = stateConfirm
			}
		case "q", "esc":
			return m, switchView(0)
		}

	case stateConfirm:
		switch strings.ToLower(key) {
		case "y", "j", "enter":
			return m.doDelete()
		case "n", "q", "esc":
			m.state = stateSelect
			m.marked = make(map[int]bool)
		}

	case stateDone:
		return m, switchView(0)
	}

	return m, nil
}

func (m Model) doDelete() (Model, tea.Cmd) {
	var sids []string
	var names []string
	for idx := range m.marked {
		if idx < len(m.sessions) {
			s := m.sessions[idx]
			sids = append(sids, s.SID)
			names = append(names, s.Subject)
		}
	}

	if err := db.DeleteSessions(m.db, sids); err != nil {
		m.message = err.Error()
		m.state = stateDone
		return m, nil
	}

	for _, name := range names {
		db.LogAction(m.db, model.ActionDelete, name)
	}

	m.message = fmt.Sprintf("%d %s", len(sids), i18n.T("delete_done"))
	m.state = stateDone
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	count := len(m.sessions)
	b.WriteString("\n")
	b.WriteString("  " + styles.RenderLogo(i18n.T("delete_title"), count))
	b.WriteString("\n\n")

	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - 8
	if innerW < 56 {
		innerW = 56
	}
	b.WriteString(styles.DoubleLine(innerW))
	b.WriteString("\n\n")

	if len(m.rows) == 0 {
		b.WriteString("  " + styles.Amber.Render(i18n.T("sessions_none")) + "\n")
		b.WriteString("\n  " + styles.Dim.Render(i18n.T("press_q")) + "\n")
		return b.String()
	}

	// Render interactive controls ABOVE the table so they stay visible
	// even when the table is taller than the terminal.
	switch m.state {
	case stateSelect:
		if m.message != "" {
			b.WriteString("  " + styles.Amber.Render(m.message) + "\n")
		}
		markedCount := len(m.marked)
		if markedCount > 0 {
			b.WriteString(fmt.Sprintf("  "+styles.Red.Render("%d marked for deletion")+"\n", markedCount))
		}
		b.WriteString("  " + styles.Dim.Render("↑/↓ navigate  Space mark  Enter delete marked  q back") + "\n\n")

	case stateConfirm:
		b.WriteString("  " + styles.Red.Render(fmt.Sprintf("Delete %d session(s)?", len(m.marked))) + "\n")
		for idx := range m.marked {
			if idx < len(m.sessions) {
				b.WriteString("    " + styles.Red.Render("✗ "+m.sessions[idx].Subject) + "\n")
			}
		}
		b.WriteString("\n  " + i18n.T("delete_confirm") + " [" + i18n.ConfirmPrompt() + "] \n\n")

	case stateDone:
		if m.message != "" {
			b.WriteString("  " + styles.Green.Render("✓ "+m.message) + "\n")
		}
		b.WriteString("  " + styles.Dim.Render(i18n.T("press_q")) + "\n\n")
	}

	// Render table with marked rows indicated.
	displayRows := make([]components.TableRow, len(m.rows))
	copy(displayRows, m.rows)
	for i := range displayRows {
		if m.marked[i] {
			// Prefix subject with ✗ to show marked for deletion.
			displayRows[i].Session.Subject = "✗ " + displayRows[i].Session.Subject
		}
	}

	table := components.RenderTable(displayRows, components.TableFull, w)
	for _, line := range strings.Split(table, "\n") {
		b.WriteString("  " + line + "\n")
	}
	legend := components.RenderLegend(m.rows)
	if legend != "" {
		b.WriteString(legend + "\n")
	}

	return b.String()
}

func switchView(v int) tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: v}
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
