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

type mode int

const (
	modeNavigate mode = iota // Arrow keys to navigate, Enter to select
	modeSearch               // Typing search query with live filter
)

// Model is the session browser view.
type Model struct {
	db       *db.DB
	sessions []model.Session
	rows     []components.TableRow
	width    int
	height   int
	mode     mode
	cursor   int    // 0-based index of selected row
	input    string // search input
	search   string // active search query
	message  string
	offset   int // scroll offset for long lists
}

// New creates a new browser model.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100, mode: modeNavigate}
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
	// Reset cursor.
	m.cursor = 0
	m.offset = 0
	m.updateSelection()
}

func (m *Model) updateSelection() {
	for i := range m.rows {
		m.rows[i].Selected = (i == m.cursor)
	}
}

// visibleRows returns how many data rows fit on screen.
func (m Model) visibleRows() int {
	// Reserve lines for: logo(3) + separator(1) + header(3) + bottom_border(1) + hints(2) + search(1) + padding(4)
	available := m.height - 15
	if available < 3 {
		available = 3
	}
	return available
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

	switch m.mode {
	case modeNavigate:
		return m.handleNavigate(key)
	case modeSearch:
		return m.handleSearch(key, msg)
	}
	return m, nil
}

func (m Model) handleNavigate(key string) (Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.updateSelection()
			// Scroll up if needed.
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
			m.updateSelection()
			// Scroll down if needed.
			vis := m.visibleRows()
			if m.cursor >= m.offset+vis {
				m.offset = m.cursor - vis + 1
			}
		}
	case "home", "g":
		m.cursor = 0
		m.offset = 0
		m.updateSelection()
	case "end", "G":
		if len(m.rows) > 0 {
			m.cursor = len(m.rows) - 1
			vis := m.visibleRows()
			if m.cursor >= vis {
				m.offset = m.cursor - vis + 1
			}
			m.updateSelection()
		}
	case "pgup":
		vis := m.visibleRows()
		m.cursor -= vis
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.offset -= vis
		if m.offset < 0 {
			m.offset = 0
		}
		m.updateSelection()
	case "pgdown":
		vis := m.visibleRows()
		m.cursor += vis
		if m.cursor >= len(m.rows) {
			m.cursor = len(m.rows) - 1
		}
		if m.cursor >= m.offset+vis {
			m.offset = m.cursor - vis + 1
		}
		m.updateSelection()
	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.sessions) {
			s := m.sessions[m.cursor]
			return m, func() tea.Msg {
				return ResumeSessionMsg{SID: s.SID, CWD: s.CWD}
			}
		}
	case "/":
		// Switch to search mode.
		m.mode = modeSearch
		m.input = ""
		m.message = ""
	case "esc", "q":
		if m.search != "" {
			// Clear search, show all.
			m.search = ""
			m.input = ""
			m.message = ""
			m.loadSessions("")
			return m, nil
		}
		return m, switchView(0)
	}
	return m, nil
}

func (m Model) handleSearch(key string, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch key {
	case "enter":
		// Accept search and switch to navigate mode.
		m.mode = modeNavigate
		m.search = strings.TrimSpace(m.input)
		if m.search == "" {
			m.loadSessions("")
		}
		m.message = ""
	case "esc":
		// Cancel search, restore all sessions.
		m.mode = modeNavigate
		m.input = ""
		m.search = ""
		m.message = ""
		m.loadSessions("")
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		// Live filter on every keystroke.
		m.liveFilter()
	default:
		if len(key) == 1 {
			m.input += key
		} else if key == "space" {
			m.input += " "
		}
		// Live filter on every keystroke.
		m.liveFilter()
	}
	return m, nil
}

func (m *Model) liveFilter() {
	query := strings.TrimSpace(m.input)
	m.search = query
	m.loadSessions(query)
	if len(m.sessions) == 0 && query != "" {
		m.message = i18n.T("search_no_results")
	} else {
		m.message = ""
	}
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
		// Render visible rows with scroll offset.
		vis := m.visibleRows()
		end := m.offset + vis
		if end > len(m.rows) {
			end = len(m.rows)
		}
		visibleRows := m.rows[m.offset:end]

		table := components.RenderTable(visibleRows, components.TableFull, w)
		for _, line := range strings.Split(table, "\n") {
			b.WriteString("  " + line + "\n")
		}
		legend := components.RenderLegend(m.rows)
		if legend != "" {
			b.WriteString(legend + "\n")
		}

		// Scroll indicator.
		if len(m.rows) > vis {
			b.WriteString(fmt.Sprintf("  "+styles.Dim.Render("(%d-%d of %d)")+"\n",
				m.offset+1, end, len(m.rows)))
		}
	}

	b.WriteString("\n")
	if m.message != "" {
		b.WriteString("  " + styles.Amber.Render(m.message) + "\n")
	}

	// Hints and input based on mode.
	switch m.mode {
	case modeNavigate:
		b.WriteString("  " + styles.Dim.Render("↑/↓ navigate  Enter resume  / search  q back") + "\n")
	case modeSearch:
		b.WriteString("  " + i18n.T("search_prompt") + ": " + m.input + "█\n")
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
