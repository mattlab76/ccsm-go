package stats

import (
	"fmt"
	"sort"
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

// Model is the statistics view.
type Model struct {
	db     *db.DB
	width  int
	height int
	scroll int

	sessions       []model.Session
	rows           []components.TableRow
	lifetimeIn     int64
	lifetimeOut    int64
	activeIn       int64
	activeOut      int64
	topDirs        []dirCount
	topTags        []tagCount
	settings       *model.Settings
	contentLines   []string
}

type dirCount struct {
	Dir   string
	Count int
}

type tagCount struct {
	Tag   string
	Count int
}

// New creates a new statistics model.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100}
	m.load()
	return m
}

func (m *Model) load() {
	sessions, _ := db.ListSessions(m.db, 0)
	m.sessions = sessions
	m.lifetimeIn, m.lifetimeOut, _ = db.GetLifetimeTokens(m.db)
	m.settings, _ = db.GetSettings(m.db)

	// Build table rows and calculate active totals.
	m.rows = make([]components.TableRow, len(sessions))
	dirMap := make(map[string]int)
	tagMap := make(map[string]int)

	for i, s := range sessions {
		status := components.StatusOK
		if !claude.IsSessionValid(s.SID) {
			status = components.StatusExpired
		}
		m.rows[i] = components.TableRow{Num: i + 1, Session: s, Status: status}

		m.activeIn += s.TotalInputTokens
		m.activeOut += s.TotalOutputTokens
		dirMap[s.CWD]++
		if s.Tags != "" && s.Tags != "-" {
			for _, tag := range strings.Fields(s.Tags) {
				tagMap[tag]++
			}
		}
	}

	// Top 5 directories.
	for dir, count := range dirMap {
		m.topDirs = append(m.topDirs, dirCount{Dir: dir, Count: count})
	}
	sort.Slice(m.topDirs, func(i, j int) bool { return m.topDirs[i].Count > m.topDirs[j].Count })
	if len(m.topDirs) > 5 {
		m.topDirs = m.topDirs[:5]
	}

	// Top tags.
	for tag, count := range tagMap {
		m.topTags = append(m.topTags, tagCount{Tag: tag, Count: count})
	}
	sort.Slice(m.topTags, func(i, j int) bool { return m.topTags[i].Count > m.topTags[j].Count })
	if len(m.topTags) > 10 {
		m.topTags = m.topTags[:10]
	}

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

	// Logo.
	add("")
	add("  " + styles.RenderLogo(
		fmt.Sprintf("%s v%s", i18n.T("stats_title"), model.Version),
		len(m.sessions),
	))
	add("")
	add(styles.DoubleLine(innerW))

	// Overview.
	add("")
	add("  " + styles.TealBold.Render(i18n.T("stats_overview")))
	add("")
	add(fmt.Sprintf("  Sessions: %d", len(m.sessions)))
	if len(m.sessions) > 0 {
		oldest := m.sessions[len(m.sessions)-1].CreatedAt.Format("2006-01-02 15:04")
		newest := m.sessions[0].CreatedAt.Format("2006-01-02 15:04")
		add(fmt.Sprintf("  Oldest:   %s", oldest))
		add(fmt.Sprintf("  Newest:   %s", newest))
	}
	if m.settings != nil && m.settings.CleanupDays > 0 {
		add(fmt.Sprintf("  Cleanup:  %d days", m.settings.CleanupDays))
	}

	// Token usage.
	add("")
	add(styles.DoubleLine(innerW))
	add("")
	add("  " + styles.TealBold.Render(i18n.T("stats_token_usage")))
	add("")

	// Active sessions tokens.
	add("  " + styles.Bold.Render(i18n.T("stats_active")))
	add(fmt.Sprintf("    %s: %s  %s", i18n.T("stats_input"), components.FormatTokens(m.activeIn), renderBar(m.activeIn, m.activeOut, true)))
	add(fmt.Sprintf("    %s: %s  %s", i18n.T("stats_output"), components.FormatTokens(m.activeOut), renderBar(m.activeOut, m.activeIn, false)))

	// Lifetime tokens.
	if m.lifetimeIn > 0 || m.lifetimeOut > 0 {
		add("")
		add("  " + styles.Bold.Render(i18n.T("stats_lifetime")))
		add(fmt.Sprintf("    %s: %s  %s", i18n.T("stats_input"), components.FormatTokens(m.lifetimeIn), renderBar(m.lifetimeIn, m.lifetimeOut, true)))
		add(fmt.Sprintf("    %s: %s  %s", i18n.T("stats_output"), components.FormatTokens(m.lifetimeOut), renderBar(m.lifetimeOut, m.lifetimeIn, false)))
	}

	// Token info.
	add("")
	add("  " + styles.Amber.Render(i18n.T("stats_token_info")))
	add("  " + styles.Dim.Render(i18n.T("stats_token_explain1")))
	add("  " + styles.Dim.Render(i18n.T("stats_token_explain2")))
	add("  " + styles.Dim.Render(i18n.T("stats_token_explain3")))
	add("  " + styles.Dim.Render(i18n.T("stats_token_explain4")))

	// Top directories.
	if len(m.topDirs) > 0 {
		add("")
		add(styles.DoubleLine(innerW))
		add("")
		add("  " + styles.TealBold.Render(i18n.T("stats_top_dirs")))
		add("")
		for _, d := range m.topDirs {
			short := components.ShortenPath(d.Dir, 50)
			add(fmt.Sprintf("    %3d  %s", d.Count, short))
		}
	}

	// Top tags.
	if len(m.topTags) > 0 {
		add("")
		add("  " + styles.TealBold.Render(i18n.T("stats_top_tags")))
		add("")
		for _, t := range m.topTags {
			add(fmt.Sprintf("    %3d  %s", t.Count, t.Tag))
		}
	}

	// All sessions table.
	if len(m.rows) > 0 {
		add("")
		add(styles.DoubleLine(innerW))
		add("")
		add("  " + styles.TealBold.Render(i18n.T("stats_all_sessions")))
		add("")
		table := components.RenderTable(m.rows, components.TableFull, w)
		for _, line := range strings.Split(table, "\n") {
			add("  " + line)
		}
		legend := components.RenderLegend(m.rows)
		if legend != "" {
			add(legend)
		}
	}

	add("")
	add("  " + styles.Dim.Render(i18n.T("press_q")))
	add("")

	m.contentLines = lines
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
	if len(m.sessions) == 0 {
		return "\n  " + styles.Amber.Render(i18n.T("stats_none")) + "\n\n  " + styles.Dim.Render(i18n.T("press_q")) + "\n"
	}

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

// renderBar creates a Unicode bar: █ filled, ░ empty.
func renderBar(value, other int64, isInput bool) string {
	barWidth := 24
	maxVal := value
	if other > maxVal {
		maxVal = other
	}
	if maxVal <= 0 {
		return ""
	}
	filled := int(float64(value) / float64(maxVal) * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	if isInput {
		return styles.Violet.Render(bar)
	}
	return styles.Teal.Render(bar)
}
