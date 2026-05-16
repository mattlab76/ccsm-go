// Package stats renders the in-app Statistics view.
//
// Data comes from two sources:
//   - the ccsm SQLite DB (sessions, tags, dirs)
//   - the local Claude Code JSONL transcripts via internal/usage
//     (per-message token + model granularity, cost estimates)
//
// All money values are rendered in the user's configured currency
// (Settings → 4 Currency / 5 Rate). The "API vs Abo" section uses the
// configured plan name + monthly price for a personalised comparison.
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
	"github.com/mattlab76/ccsm-go/internal/usage"
)

// SwitchViewMsg is sent to the parent app to change views.
type SwitchViewMsg struct {
	View int
}

// Model is the statistics view.
//
// The rendered output is split into a fixed sticky header (logo +
// separator) that stays visible at the top of the view, and a body
// that scrolls with the cursor. This keeps the title/count anchored
// while the user pages through the data tables.
type Model struct {
	db     *db.DB
	width  int
	height int
	scroll int

	sessions    []model.Session
	rows        []components.TableRow
	topDirs     []dirCount
	topTags     []tagCount
	settings    *model.Settings
	snap        usage.Snapshot
	headerLines []string // fixed at top, never scrolls
	bodyLines   []string // scrolls
}

type dirCount struct {
	Dir   string
	Count int
}

type tagCount struct {
	Tag   string
	Count int
}

// New creates a new statistics model and loads all data synchronously.
func New(database *db.DB) Model {
	m := Model{db: database, width: 100}
	m.load()
	return m
}

func (m *Model) load() {
	sessions, _ := db.ListSessions(m.db, 0)
	m.sessions = sessions
	m.settings, _ = db.GetSettings(m.db)
	m.snap, _ = usage.Compute()

	m.rows = make([]components.TableRow, len(sessions))
	dirMap := make(map[string]int)
	tagMap := make(map[string]int)
	for i, s := range sessions {
		status := components.StatusOK
		if !claude.IsSessionValid(s.SID) {
			status = components.StatusExpired
		}
		m.rows[i] = components.TableRow{Num: i + 1, Session: s, Status: status}
		dirMap[s.CWD]++
		if s.Tags != "" && s.Tags != "-" {
			for _, tag := range strings.Fields(s.Tags) {
				tagMap[tag]++
			}
		}
	}

	for dir, count := range dirMap {
		m.topDirs = append(m.topDirs, dirCount{Dir: dir, Count: count})
	}
	sort.Slice(m.topDirs, func(i, j int) bool { return m.topDirs[i].Count > m.topDirs[j].Count })
	if len(m.topDirs) > 5 {
		m.topDirs = m.topDirs[:5]
	}

	for tag, count := range tagMap {
		m.topTags = append(m.topTags, tagCount{Tag: tag, Count: count})
	}
	sort.Slice(m.topTags, func(i, j int) bool { return m.topTags[i].Count > m.topTags[j].Count })
	if len(m.topTags) > 10 {
		m.topTags = m.topTags[:10]
	}

	m.buildContent()
}

// money formats a USD amount in the user's configured currency.
func (m *Model) money(usd float64) string {
	cur, rate := "USD", 1.0
	if m.settings != nil {
		if m.settings.Currency != "" {
			cur = m.settings.Currency
		}
		if m.settings.ExchangeRate > 0 {
			rate = m.settings.ExchangeRate
		}
	}
	return usage.FormatMoney(usd, cur, rate)
}

func (m *Model) buildContent() {
	var header, lines []string
	add := func(s string) { lines = append(lines, s) }
	addTable := func(s string) {
		for _, line := range strings.Split(s, "\n") {
			add("  " + line)
		}
	}

	w := m.width
	if w < 60 {
		w = 60
	}
	innerW := w - 8
	if innerW < 56 {
		innerW = 56
	}
	sep := styles.DoubleLine(innerW)

	// ── Sticky header (never scrolls) ──────────────────────────────
	header = append(header, "")
	logo := "  " + styles.RenderLogo(
		fmt.Sprintf("%s v%s", i18n.T("stats_title"), model.Version),
		len(m.sessions))
	for _, line := range strings.Split(logo, "\n") {
		header = append(header, line)
	}
	header = append(header, "")
	header = append(header, sep)
	m.headerLines = header

	// ── Overview ───────────────────────────────────────────────────
	add("")
	add("  " + styles.TealBold.Render(i18n.T("stats_overview")))
	add("")
	overviewRows := [][]string{
		{"Sessions", fmt.Sprintf("%d", len(m.sessions))},
	}
	if len(m.sessions) > 0 {
		overviewRows = append(overviewRows,
			[]string{"Oldest", m.sessions[len(m.sessions)-1].CreatedAt.Format("2006-01-02 15:04")},
			[]string{"Newest", m.sessions[0].CreatedAt.Format("2006-01-02 15:04")})
	}
	if m.settings != nil && m.settings.CleanupDays > 0 {
		overviewRows = append(overviewRows,
			[]string{"Cleanup", fmt.Sprintf("%d days", m.settings.CleanupDays)})
	}
	if m.settings != nil && m.settings.PlanName != "" {
		planLine := m.settings.PlanName
		if m.settings.PlanMonthlyPrice > 0 {
			planLine += fmt.Sprintf(" · %.2f %s/mo",
				m.settings.PlanMonthlyPrice, m.settings.Currency)
		}
		overviewRows = append(overviewRows, []string{"Plan", planLine})
	}
	addTable(components.RenderKVTable(
		[]components.Col{{Header: "Field", Width: 12}, {Header: "Value", Width: 36}},
		overviewRows))

	// ── Token consumption per window ───────────────────────────────
	add("")
	add(sep)
	add("")
	add("  " + styles.TealBold.Render(i18n.T("stats_consumption")))
	add("")
	consumptionRows := [][]string{
		periodRow(i18n.T("stats_window_today"), m.snap.Today, m.money),
		periodRow(i18n.T("stats_window_24h"), m.snap.Last24h, m.money),
		periodRow(i18n.T("stats_window_7d"), m.snap.ThisWeek, m.money),
		periodRow(i18n.T("stats_window_all"), m.snap.AllTime, m.money),
	}
	addTable(components.RenderKVTable(
		[]components.Col{
			{Header: "Window", Width: 14},
			{Header: "Tokens", Width: 10, Right: true},
			{Header: "Input", Width: 10, Right: true},
			{Header: "Output", Width: 10, Right: true},
			{Header: "Cost", Width: 10, Right: true},
		},
		consumptionRows))
	add("    " + styles.Dim.Render(i18n.T("stats_token_legend")))

	// ── Per-model breakdown ────────────────────────────────────────
	if len(m.snap.ThisWeek.ByModel) > 0 {
		add("")
		add(sep)
		add("")
		add("  " + styles.TealBold.Render(i18n.T("stats_by_model")))
		add("")
		byTok := make([]usage.ModelStats, len(m.snap.ThisWeek.ByModel))
		copy(byTok, m.snap.ThisWeek.ByModel)
		sort.Slice(byTok, func(i, j int) bool {
			return (byTok[i].InputTokens + byTok[i].OutputTokens) >
				(byTok[j].InputTokens + byTok[j].OutputTokens)
		})
		totalTok := m.snap.ThisWeek.InputTokens + m.snap.ThisWeek.OutputTokens
		modelRows := make([][]string, 0, len(byTok))
		for _, ms := range byTok {
			tok := ms.InputTokens + ms.OutputTokens
			share := 0.0
			if totalTok > 0 {
				share = float64(tok) / float64(totalTok) * 100
			}
			modelRows = append(modelRows, []string{
				usage.ModelShort(ms.Model),
				usage.FormatTokens(tok),
				fmt.Sprintf("%.1f%%", share),
				m.money(ms.CostUSD),
				usage.FormatTokens(ms.CacheRead),
				usage.FormatTokens(ms.CacheWrite),
			})
		}
		addTable(components.RenderKVTable(
			[]components.Col{
				{Header: "Model", Width: 12},
				{Header: "Tokens", Width: 10, Right: true},
				{Header: "Share", Width: 7, Right: true},
				{Header: "Cost", Width: 10, Right: true},
				{Header: "Cache R", Width: 10, Right: true},
				{Header: "Cache W", Width: 10, Right: true},
			},
			modelRows))
	}

	// ── API vs subscription ────────────────────────────────────────
	add("")
	add(sep)
	add("")
	add("  " + styles.TealBold.Render(i18n.T("stats_compare_title")))
	add("  " + styles.Dim.Render(i18n.T("stats_compare_intro")))
	add("")
	monthly := m.snap.ThisWeek.CostUSD * 4.33
	compareRows := [][]string{
		{i18n.T("stats_compare_week"), m.money(m.snap.ThisWeek.CostUSD)},
		{i18n.T("stats_compare_month"), m.money(monthly)},
	}
	if m.settings != nil && m.settings.PlanMonthlyPrice > 0 {
		// Plan price is already in target currency; un-convert for diff math
		// by converting the API estimate to the same currency space.
		monthlyTarget := monthly
		if m.settings.ExchangeRate > 0 && m.settings.Currency != "USD" && m.settings.Currency != "" {
			monthlyTarget = monthly * m.settings.ExchangeRate
		}
		diff := monthlyTarget - m.settings.PlanMonthlyPrice
		verdict := "→ subscription is cheaper"
		diffStr := fmt.Sprintf("+%.0f %s", diff, m.settings.Currency)
		if diff < 0 {
			verdict = "→ pay-per-use would be cheaper"
			diffStr = fmt.Sprintf("%.0f %s", diff, m.settings.Currency)
		}
		compareRows = append(compareRows,
			[]string{fmt.Sprintf("Plan: %s", m.settings.PlanName),
				fmt.Sprintf("%.2f %s/mo", m.settings.PlanMonthlyPrice, m.settings.Currency)},
			[]string{"Diff (API − Plan)", diffStr + "  " + verdict},
		)
	}
	addTable(components.RenderKVTable(
		[]components.Col{
			{Header: "Item", Width: 32},
			{Header: "Amount", Width: 40},
		},
		compareRows))
	if m.settings == nil || m.settings.PlanName == "" {
		add("    " + styles.Dim.Render(
			"Set plan in Settings (6/7) to see personalised pay-per-use vs plan diff."))
	}

	// ── Top sessions (7d, by cost) ─────────────────────────────────
	if len(m.snap.TopSessions) > 0 {
		add("")
		add(sep)
		add("")
		add("  " + styles.TealBold.Render(i18n.T("stats_top_sessions")))
		add("")
		dirW := 50
		topRows := make([][]string, 0, len(m.snap.TopSessions))
		for i, sc := range m.snap.TopSessions {
			topRows = append(topRows, []string{
				fmt.Sprintf("%d", i+1),
				m.money(sc.CostUSD),
				sc.Last.Format("2006-01-02"),
				components.ShortenPath(sc.CWD, dirW),
			})
		}
		addTable(components.RenderKVTable(
			[]components.Col{
				{Header: "#", Width: 2, Right: true},
				{Header: "Cost", Width: 10, Right: true},
				{Header: "Date", Width: 10},
				{Header: "Directory", Width: dirW},
			},
			topRows))
	}

	// ── Patterns ───────────────────────────────────────────────────
	add("")
	add(sep)
	add("")
	add("  " + styles.TealBold.Render(i18n.T("stats_patterns")))
	add("  " + styles.Dim.Render(i18n.T("stats_patterns_intro")))
	add("")
	patternRows := [][]string{
		{i18n.T("stats_long_context"), fmt.Sprintf("%.0f%%", m.snap.LongContextPct)},
		{i18n.T("stats_subagent"), fmt.Sprintf("%.0f%%", m.snap.SubagentPct)},
	}
	addTable(components.RenderKVTable(
		[]components.Col{
			{Header: "Metric", Width: 60},
			{Header: "Value", Width: 8, Right: true},
		},
		patternRows))

	// ── Top directories ────────────────────────────────────────────
	if len(m.topDirs) > 0 {
		add("")
		add(sep)
		add("")
		add("  " + styles.TealBold.Render(i18n.T("stats_top_dirs")))
		add("")
		dirRows := make([][]string, 0, len(m.topDirs))
		for _, d := range m.topDirs {
			dirRows = append(dirRows, []string{
				fmt.Sprintf("%d", d.Count),
				components.ShortenPath(d.Dir, innerW-12),
			})
		}
		addTable(components.RenderKVTable(
			[]components.Col{
				{Header: "Sessions", Width: 8, Right: true},
				{Header: "Directory", Width: 50},
			},
			dirRows))
	}

	// ── Top tags ───────────────────────────────────────────────────
	if len(m.topTags) > 0 {
		add("")
		add("  " + styles.TealBold.Render(i18n.T("stats_top_tags")))
		add("")
		tagRows := make([][]string, 0, len(m.topTags))
		for _, t := range m.topTags {
			tagRows = append(tagRows, []string{
				fmt.Sprintf("%d", t.Count),
				t.Tag,
			})
		}
		addTable(components.RenderKVTable(
			[]components.Col{
				{Header: "Count", Width: 8, Right: true},
				{Header: "Tag", Width: 30},
			},
			tagRows))
	}

	// ── All sessions ───────────────────────────────────────────────
	if len(m.rows) > 0 {
		add("")
		add(sep)
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
	m.bodyLines = lines
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	m.buildContent()
	return m
}

func (m Model) Init() tea.Cmd { return nil }

// bodyHeight is how many body lines fit on screen below the sticky header.
func (m Model) bodyHeight() int {
	h := m.height - len(m.headerLines) - 2 // -2 for status bar at bottom
	if h < 5 {
		h = 5
	}
	return h
}

// maxScroll caps the body scroll position so the last body line stays visible.
func (m Model) maxScroll() int {
	max := len(m.bodyLines) - m.bodyHeight()
	if max < 0 {
		return 0
	}
	return max
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
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
	if len(m.sessions) == 0 && m.snap.FileCount == 0 {
		return "\n  " + styles.Amber.Render(i18n.T("stats_none")) + "\n\n  " + styles.Dim.Render(i18n.T("press_q")) + "\n"
	}
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

	// Scroll indicator on the last line of the visible body if there is more.
	if m.maxScroll() > 0 {
		b.WriteString("\n  " + styles.Dim.Render(fmt.Sprintf(
			"(%d-%d of %d  ·  ↑↓ scroll  PgUp/PgDn page  g/G top/bottom)",
			start+1, end, len(m.bodyLines))))
	}
	return b.String()
}

// periodRow builds one row for the per-window consumption table.
func periodRow(label string, p usage.PeriodStats, money func(float64) string) []string {
	tot := p.InputTokens + p.OutputTokens
	return []string{
		label,
		usage.FormatTokens(tot),
		usage.FormatTokens(p.InputTokens),
		usage.FormatTokens(p.OutputTokens),
		money(p.CostUSD),
	}
}
