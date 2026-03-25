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
type Model struct {
	db           *db.DB
	entries      []model.LogEntry
	width        int
	height       int
	scroll       int
	contentLines []string
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
	var lines []string
	add := func(s string) { lines = append(lines, s) }

	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - 8
	if innerW < 56 {
		innerW = 56
	}

	add("")
	add("  " + styles.RenderLogo(i18n.T("log_title"), len(m.entries)))
	add("")
	add(styles.DoubleLine(innerW))
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
	add("")

	m.contentLines = lines
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
			maxScroll := len(m.contentLines) - m.height + 2
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scroll < maxScroll {
				m.scroll++
			}
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	visibleHeight := m.height
	if visibleHeight <= 0 {
		visibleHeight = 40
	}

	start := m.scroll
	end := start + visibleHeight
	if end > len(m.contentLines) {
		end = len(m.contentLines)
	}
	if start >= end {
		start = 0
		if end == 0 {
			end = len(m.contentLines)
		}
	}

	return strings.Join(m.contentLines[start:end], "\n")
}
