package log

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/styles"
)

// SwitchViewMsg is sent to the parent app to change views.
type SwitchViewMsg struct {
	View int
}

// Model is the activity log view.
//
// Output is split into a fixed sticky header (logo + separator) and a
// body that scrolls. Keeps the title anchored while paging through
// long log histories.
type Model struct {
	db          *db.DB
	entries     []model.LogEntry
	width       int
	height      int
	scroll      int
	headerLines []string
	bodyLines   []string
}

// New creates a new activity log model.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100}
	m.load()
	return m
}

func (m *Model) load() {
	m.entries, _ = db.ListLog(m.db)
	m.buildContent()
}

func (m *Model) buildContent() {
	var header, lines []string

	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - 8
	if innerW < 56 {
		innerW = 56
	}

	// Sticky header: blank + logo (split per visual line) + blank + separator.
	header = append(header, "")
	logo := "  " + styles.RenderLogo(i18n.T("log_title"), len(m.entries))
	for _, line := range strings.Split(logo, "\n") {
		header = append(header, line)
	}
	header = append(header, "")
	header = append(header, styles.DoubleLine(innerW))
	m.headerLines = header

	// Body: log entries.
	add := func(s string) { lines = append(lines, s) }
	add("")
	if len(m.entries) == 0 {
		add("  " + styles.Amber.Render(i18n.T("log_empty")))
	} else {
		for _, e := range m.entries {
			ts := e.Timestamp.Format("2006-01-02 15:04")
			action := colorAction(e.Action)
			add(fmt.Sprintf("  %s  %s  %s", ts, action, e.Message))
		}
	}
	add("")
	add("  " + styles.Dim.Render(i18n.T("press_q")))
	m.bodyLines = lines
}

func colorAction(action string) string {
	tag := fmt.Sprintf("[%s]", action)
	padded := fmt.Sprintf("%-10s", tag)
	switch action {
	case model.ActionNew:
		return styles.Green.Render(padded)
	case model.ActionResume:
		return styles.Teal.Render(padded)
	case model.ActionSave:
		return styles.Violet.Render(padded)
	case model.ActionDelete:
		return styles.Amber.Render(padded)
	case model.ActionCleanup:
		return styles.Amber.Render(padded)
	case model.ActionSettings:
		return styles.Dim.Render(padded)
	default:
		return styles.Dim.Render(padded)
	}
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	m.buildContent()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) bodyHeight() int {
	h := m.height - len(m.headerLines) - 2
	if h < 5 {
		h = 5
	}
	return h
}

func (m Model) maxScroll() int {
	max := len(m.bodyLines) - m.bodyHeight()
	if max < 0 {
		return 0
	}
	return max
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return SwitchViewMsg{View: 0} }
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			if m.scroll < m.maxScroll() {
				m.scroll++
			}
		case "home", "g":
			m.scroll = 0
		case "end", "G":
			m.scroll = m.maxScroll()
		case "pgup":
			step := m.bodyHeight() - 2
			if step < 1 {
				step = 1
			}
			m.scroll -= step
			if m.scroll < 0 {
				m.scroll = 0
			}
		case "pgdown":
			step := m.bodyHeight() - 2
			if step < 1 {
				step = 1
			}
			m.scroll += step
			if m.scroll > m.maxScroll() {
				m.scroll = m.maxScroll()
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	bodyH := m.bodyHeight()
	start := m.scroll
	end := start + bodyH
	if end > len(m.bodyLines) {
		end = len(m.bodyLines)
	}
	if start > end {
		start = end
	}

	var b strings.Builder
	b.WriteString(strings.Join(m.headerLines, "\n"))
	b.WriteString("\n")
	b.WriteString(strings.Join(m.bodyLines[start:end], "\n"))

	if m.maxScroll() > 0 {
		b.WriteString("\n  " + styles.Dim.Render(fmt.Sprintf(
			"(%d-%d of %d  ·  ↑↓ scroll  PgUp/PgDn page  g/G top/bottom)",
			start+1, end, len(m.bodyLines))))
	}
	return b.String()
}
