package browser

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

// ResumeSessionMsg is sent to the parent app to resume a session.
type ResumeSessionMsg struct {
	SID string
	CWD string
}

type state int

const (
	stateBrowse state = iota
	stateSearch
)

// Model is the session browser view.
type Model struct {
	db       *db.DB
	sessions []model.Session
	rows     []components.TableRow
	width    int
	height   int
	state    state
	input    string
	search   string
	message  string
}

// New creates a new browser model.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100, state: stateBrowse}
	m.loadSessions("")
	return m
}

func (m *Model) loadSessions(query string) {
	var sessions []model.Session
	var err error
	if query != "" {
		sessions, err = db.SearchSessions(m.db, query)
	} else {
		sessions, err = db.ListSessions(m.db, 0)
	}
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
	case "esc", "q":
		if m.search != "" {
			// Clear search, show all.
			m.search = ""
			m.input = ""
			m.message = ""
			m.loadSessions("")
			return m, nil
		}
		return m, switchView(0) // Back to main menu.
	case "enter":
		input := strings.TrimSpace(m.input)
		m.input = ""
		if strings.HasPrefix(input, "/") {
			// Search.
			query := strings.TrimSpace(input[1:])
			if query == "" {
				m.search = ""
				m.loadSessions("")
			} else {
				m.search = query
				m.loadSessions(query)
				if len(m.sessions) == 0 {
					m.message = i18n.T("search_no_results")
				} else {
					m.message = ""
				}
			}
			return m, nil
		}
		// Try as number selection.
		idx := parseNumber(input)
		if idx > 0 && idx <= len(m.sessions) {
			s := m.sessions[idx-1]
			return m, func() tea.Msg {
				return ResumeSessionMsg{SID: s.SID, CWD: s.CWD}
			}
		}
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
	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	count := len(m.sessions)
	title := i18n.T("menu_resume")
	if m.search != "" {
		title = i18n.T("search_title") + ": " + m.search
	}

	b.WriteString("\n")
	b.WriteString("  " + styles.RenderLogo(title, count))
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
		if m.search != "" {
			b.WriteString("  " + styles.Amber.Render(i18n.T("search_no_results")) + "\n")
		} else {
			b.WriteString("  " + styles.Amber.Render(i18n.T("sessions_none")) + "\n")
		}
	} else {
		table := components.RenderTable(m.rows, components.TableFull, w)
		for _, line := range strings.Split(table, "\n") {
			b.WriteString("  " + line + "\n")
		}
		legend := components.RenderLegend(m.rows)
		if legend != "" {
			b.WriteString(legend + "\n")
		}
	}

	b.WriteString("\n")
	if m.message != "" {
		b.WriteString("  " + styles.Amber.Render(m.message) + "\n")
	}

	b.WriteString("  " + styles.Dim.Render(fmt.Sprintf(
		"Enter number to resume, /keyword to search, q to return")) + "\n")
	b.WriteString("  > " + m.input + "█\n")

	return b.String()
}

func switchView(v int) tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: v}
	}
}

func parseNumber(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
