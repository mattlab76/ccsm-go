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
	cursor   int          // 0-based cursor position
	offset   int          // index of the first row currently visible
	marked   map[int]bool // marked for deletion (0-based indices)
	message  string
}

// visibleRows returns how many table rows fit on screen.
// Mirrors browser.visibleRows so both lists behave consistently.
func (m Model) visibleRows() int {
	// Reserve lines for: logo(3) + separator(1) + controls(4) + table header(3) +
	// bottom border(1) + legend(1) + scroll indicator(1) + status bar(2) + padding(2)
	available := m.height - 18
	if available < 3 {
		available = 3
	}
	return available
}

// scrollIntoView adjusts m.offset so that m.cursor is visible.
func (m *Model) scrollIntoView() {
	vis := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
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
	// Ctrl+C is intercepted globally at app level (quit-confirm modal).

	switch m.state {
	case stateSelect:
		switch key {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateSelection()
				m.scrollIntoView()
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
				m.updateSelection()
				m.scrollIntoView()
			}
		case "home", "g":
			m.cursor = 0
			m.offset = 0
			m.updateSelection()
		case "end", "G":
			m.cursor = len(m.rows) - 1
			m.updateSelection()
			m.scrollIntoView()
		case "pgup":
			vis := m.visibleRows()
			m.cursor -= vis
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.updateSelection()
			m.scrollIntoView()
		case "pgdown":
			vis := m.visibleRows()
			m.cursor += vis
			if m.cursor >= len(m.rows) {
				m.cursor = len(m.rows) - 1
			}
			m.updateSelection()
			m.scrollIntoView()
		case " ": // Space toggles mark
			if m.cursor >= 0 && m.cursor < len(m.sessions) {
				if m.marked[m.cursor] {
					delete(m.marked, m.cursor)
				} else {
					m.marked[m.cursor] = true
				}
			}
		case "a":
			// Toggle-mark all expired sessions ([!]).
			m.toggleMarkBy(func(r components.TableRow) bool {
				return r.Status == components.StatusExpired
			})
		case "A":
			// Toggle-mark every visible session.
			m.toggleMarkBy(func(r components.TableRow) bool { return true })
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

// toggleMarkBy marks every row matching pred. If every matching row is
// already marked, it unmarks them all instead — so pressing the hotkey
// twice acts like an undo.
func (m *Model) toggleMarkBy(pred func(components.TableRow) bool) {
	var matches []int
	for i, r := range m.rows {
		if pred(r) {
			matches = append(matches, i)
		}
	}
	if len(matches) == 0 {
		return
	}
	allMarked := true
	for _, i := range matches {
		if !m.marked[i] {
			allMarked = false
			break
		}
	}
	for _, i := range matches {
		if allMarked {
			delete(m.marked, i)
		} else {
			m.marked[i] = true
		}
	}
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
		b.WriteString("  " + styles.Dim.Render("↑/↓ navigate  Space mark  a all expired  A all  Enter delete  q back") + "\n\n")

	case stateConfirm:
		b.WriteString("  " + styles.Red.Render(fmt.Sprintf("Delete %d session(s)?", len(m.marked))) + "\n")
		shown := 0
		for idx := range m.marked {
			if idx < len(m.sessions) {
				if shown >= 10 {
					b.WriteString(fmt.Sprintf("    %s\n",
						styles.Dim.Render(fmt.Sprintf("... and %d more", len(m.marked)-shown))))
					break
				}
				b.WriteString("    " + styles.Red.Render("✗ "+m.sessions[idx].Subject) + "\n")
				shown++
			}
		}
		b.WriteString("\n  " + i18n.T("delete_confirm") + " [" + i18n.ConfirmPrompt() + "] \n\n")

	case stateDone:
		if m.message != "" {
			b.WriteString("  " + styles.Green.Render("✓ "+m.message) + "\n")
		}
		b.WriteString("  " + styles.Dim.Render(i18n.T("press_q")) + "\n\n")
	}

	// Slice rows to the visible window. Apply the ✗ prefix only to the
	// rows we'll actually render.
	vis := m.visibleRows()
	end := m.offset + vis
	if end > len(m.rows) {
		end = len(m.rows)
	}
	displayRows := make([]components.TableRow, end-m.offset)
	for i := m.offset; i < end; i++ {
		displayRows[i-m.offset] = m.rows[i]
		if m.marked[i] {
			displayRows[i-m.offset].Session.Subject = "✗ " + m.rows[i].Session.Subject
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
	if len(m.rows) > vis {
		b.WriteString(fmt.Sprintf("  "+styles.Dim.Render("(%d-%d of %d)")+"\n",
			m.offset+1, end, len(m.rows)))
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
