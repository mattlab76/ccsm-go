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
	"github.com/mattlab76/ccsm-go/internal/usage"
)

// usageReadyMsg carries an async-computed usage snapshot to the model.
type usageReadyMsg struct {
	snap usage.Snapshot
	err  error
}

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
	db         *db.DB
	sessions   []model.Session
	rows       []components.TableRow
	actions    []menuAction
	width      int
	height     int
	cursor     int
	err        error
	totalCount int            // all sessions in DB; cached at load so View() needn't query per repaint
	usageSnap  usage.Snapshot // populated async; zero-value until ready
	usageReady bool
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
	// Cache the full count once; the menu can't gain/lose sessions while
	// it's on screen, so re-querying it on every repaint was wasteful.
	if n, err := db.CountSessions(m.db); err == nil {
		m.totalCount = n
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
	// Kick off async usage computation so the banner can populate
	// without blocking the initial menu render.
	return func() tea.Msg {
		snap, err := usage.Compute()
		return usageReadyMsg{snap: snap, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if ur, ok := msg.(usageReadyMsg); ok {
		m.usageSnap = ur.snap
		m.usageReady = true
		return m, nil
	}
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
	case "f":
		// Fork: start a fresh Claude session in the cursor session's
		// working directory, pre-filling subject + tags. New SID, no
		// `--resume`. Only meaningful when the cursor is on a session.
		if m.cursor < len(m.sessions) {
			s := m.sessions[m.cursor]
			return m, func() tea.Msg {
				return ForkSessionMsg{
					CWD:     s.CWD,
					Subject: s.Subject,
					Tags:    s.Tags,
				}
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

	// Logo + title (count cached at load — no per-repaint query).
	b.WriteString("\n")
	b.WriteString("  " + styles.RenderLogo(
		fmt.Sprintf("%s v%s", i18n.T("title"), model.Version),
		m.totalCount,
	))
	b.WriteString("\n\n")

	// Usage banner (shown once the async computation has returned).
	b.WriteString("  " + m.renderUsageBanner() + "\n")

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
	b.WriteString("  " + styles.Dim.Render("↑/↓ navigate  Enter activate  f fork session  shortcut for direct action") + "\n")

	return b.String()
}

// renderUsageBanner produces the one-line summary above the menu.
// Shows total tokens used today / over the rolling 7 days, and the
// model that dominated this week by token volume. Costs intentionally
// live in the Statistics view, not here.
func (m Model) renderUsageBanner() string {
	if !m.usageReady {
		return styles.Dim.Render("Usage: computing…")
	}
	todayTok := m.usageSnap.Today.InputTokens + m.usageSnap.Today.OutputTokens
	weekTok := m.usageSnap.ThisWeek.InputTokens + m.usageSnap.ThisWeek.OutputTokens

	topModel := "–"
	if len(m.usageSnap.ThisWeek.ByModel) > 0 {
		// Pick the heaviest model by total tokens — the snapshot sorts
		// by cost, which can rank differently when mixing Opus/Sonnet/Haiku.
		var top usage.ModelStats
		var topTok int64
		for _, ms := range m.usageSnap.ThisWeek.ByModel {
			t := ms.InputTokens + ms.OutputTokens
			if t > topTok {
				topTok = t
				top = ms
			}
		}
		share := ""
		if weekTok > 0 {
			share = fmt.Sprintf(" (%.0f%%)", float64(topTok)/float64(weekTok)*100)
		}
		topModel = usage.ModelShort(top.Model) + share
	}

	return styles.Dim.Render(fmt.Sprintf(i18n.T("usage_banner"),
		styles.Green.Render(usage.FormatTokens(todayTok)),
		styles.Green.Render(usage.FormatTokens(weekTok)),
		styles.Violet.Render(topModel)))
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

// ForkSessionMsg is sent to the parent app to start a fresh Claude
// session in an existing session's working directory, pre-filling
// the save dialog with that session's subject and tags. Triggered by
// the `f` shortcut on a cursor-selected session row.
type ForkSessionMsg struct {
	CWD     string
	Subject string
	Tags    string
}

func switchView(v int) tea.Cmd {
	return func() tea.Msg {
		return SwitchViewMsg{View: v}
	}
}
