package delete

import (
	"fmt"
	"os"
	"strconv"
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
	stateSelect state = iota
	stateConfirm
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
	input    string
	selected []int // 0-based indices
	message  string
}

// New creates a new delete model.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100, state: stateSelect}
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
		case "q", "esc":
			return m, switchView(0)
		case "enter":
			indices := parseIndices(m.input, len(m.sessions))
			m.input = ""
			if len(indices) == 0 {
				m.message = i18n.T("settings_invalid")
				return m, nil
			}
			m.selected = indices
			m.state = stateConfirm
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(key) == 1 {
				m.input += key
			} else if key == "space" {
				m.input += " "
			}
		}

	case stateConfirm:
		switch strings.ToLower(key) {
		case "y", "j":
			return m.doDelete()
		case "n", "q", "esc":
			m.state = stateSelect
			m.selected = nil
		}

	case stateDone:
		return m, switchView(0)
	}

	return m, nil
}

func (m Model) doDelete() (Model, tea.Cmd) {
	var sids []string
	var names []string
	for _, idx := range m.selected {
		s := m.sessions[idx]
		sids = append(sids, s.SID)
		names = append(names, s.Subject)
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

	table := components.RenderTable(m.rows, components.TableFull, w)
	for _, line := range strings.Split(table, "\n") {
		b.WriteString("  " + line + "\n")
	}
	legend := components.RenderLegend(m.rows)
	if legend != "" {
		b.WriteString(legend + "\n")
	}
	b.WriteString("\n")

	switch m.state {
	case stateSelect:
		if m.message != "" {
			b.WriteString("  " + styles.Amber.Render(m.message) + "\n")
		}
		b.WriteString("  " + i18n.T("cleanup_select_prompt") + "\n")
		b.WriteString("  > " + m.input + "█\n")

	case stateConfirm:
		// Show what will be deleted.
		for _, idx := range m.selected {
			s := m.sessions[idx]
			b.WriteString("  " + styles.Red.Render(fmt.Sprintf("  %d. %s", idx+1, s.Subject)) + "\n")
		}
		b.WriteString("\n  " + i18n.T("delete_confirm") + " [" + i18n.ConfirmPrompt() + "] ")

	case stateDone:
		if m.message != "" {
			b.WriteString("  " + styles.Green.Render("✓ "+m.message) + "\n")
		}
		b.WriteString("\n  " + styles.Dim.Render(i18n.T("press_q")) + "\n")
	}

	return b.String()
}

// parseIndices parses "1,3,5" into 0-based indices, validating against max.
func parseIndices(input string, max int) []int {
	var result []int
	seen := make(map[int]bool)
	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > max {
			continue
		}
		idx := n - 1
		if !seen[idx] {
			seen[idx] = true
			result = append(result, idx)
		}
	}
	return result
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
