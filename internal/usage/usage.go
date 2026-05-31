// Package usage parses Claude Code's per-session JSONL transcripts and
// aggregates token usage and approximate cost. It mirrors what the
// in-session `/usage` slash command shows, but works from outside any
// active session — same source data, same caveat: only sessions on this
// machine are seen.
package usage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PricingPerMTok lists the rates we charge against, per million tokens, in USD.
// Update when Anthropic changes published pricing. Unknown models contribute
// zero cost (better to under-report than to invent numbers).
type pricing struct {
	in, out, cacheRead, cacheWrite float64
}

// Family prefixes are matched longest-first; first match wins.
var modelPricing = map[string]pricing{
	"claude-opus":   {in: 15.00, out: 75.00, cacheRead: 1.50, cacheWrite: 18.75},
	"claude-sonnet": {in: 3.00, out: 15.00, cacheRead: 0.30, cacheWrite: 3.75},
	"claude-haiku":  {in: 1.00, out: 5.00, cacheRead: 0.10, cacheWrite: 1.25},
}

func pricingFor(model string) pricing {
	for prefix, p := range modelPricing {
		if strings.HasPrefix(model, prefix) {
			return p
		}
	}
	return pricing{}
}

// ModelStats is the per-model accumulation for a time window.
type ModelStats struct {
	Model        string
	InputTokens  int64
	OutputTokens int64
	CacheRead    int64
	CacheWrite   int64
	CostUSD      float64
}

// PeriodStats aggregates everything for one time window.
type PeriodStats struct {
	CostUSD      float64
	InputTokens  int64
	OutputTokens int64
	CacheRead    int64
	CacheWrite   int64
	ByModel      []ModelStats // sorted by CostUSD desc
}

// SessionCost is a per-session roll-up, used for top-sessions panels.
type SessionCost struct {
	SID     string
	CWD     string
	CostUSD float64
	Last    time.Time
}

// Snapshot is the full result of one aggregation pass.
type Snapshot struct {
	Today          PeriodStats
	Last24h        PeriodStats
	ThisWeek       PeriodStats // rolling 7 days
	Session        PeriodStats // rolling 5h (matches Claude Code's "current session")
	AllTime        PeriodStats
	TopSessions    []SessionCost // top 5 by cost in the last 7 days
	LongContextPct float64       // share of last-24h messages where cache_read > 150k tokens
	SubagentPct    float64       // share of last-24h cost coming from sidechain messages
	ComputedAt     time.Time
	FileCount      int

	// Reset predictions: the oldest message timestamp inside each rolling
	// window. The window "resets" at oldest + window-length, which is
	// what Claude Code's /usage displays as "Resets <time>".
	SessionWindowStart time.Time // oldest message in last 5h (zero if no msgs)
	WeekWindowStart    time.Time // oldest message in last 7d (zero if no msgs)
}

// rawLine is a partial decode of a JSONL message line. Only the fields we
// actually need are listed — json.Unmarshal silently drops the rest.
type rawLine struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	SessionID   string `json:"sessionId"`
	CWD         string `json:"cwd"`
	IsSidechain bool   `json:"isSidechain"`
	Message     struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

// Compute walks ~/.claude/projects/, parses every JSONL line, and returns
// a fully-aggregated snapshot. Cost on a fresh machine with no transcripts
// returns a zero-value snapshot, not an error.
func Compute() (Snapshot, error) {
	root, err := projectsDir()
	if err != nil {
		return Snapshot{ComputedAt: time.Now()}, err
	}

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	cutoff5h := now.Add(-5 * time.Hour)
	cutoff24h := now.Add(-24 * time.Hour)
	cutoffWeek := now.Add(-7 * 24 * time.Hour)

	snap := Snapshot{ComputedAt: now}

	todayByModel := map[string]*ModelStats{}
	last24hByModel := map[string]*ModelStats{}
	weekByModel := map[string]*ModelStats{}
	sessionByModel := map[string]*ModelStats{}
	allByModel := map[string]*ModelStats{}
	sessionTotals := map[string]*SessionCost{} // SID -> rolling 7d cost

	var last24hCount, last24hLongCtx int
	var last24hCost, last24hSidechainCost float64

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable subtrees rather than aborting
		}
		if d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		snap.FileCount++

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		// Some transcript lines are large (tool outputs); raise the buffer.
		scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
		for scanner.Scan() {
			var r rawLine
			if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
				continue
			}
			if r.Type != "assistant" || r.Message.Model == "" || r.Message.Model == "<synthetic>" {
				continue
			}
			ts, err := time.Parse(time.RFC3339Nano, r.Timestamp)
			if err != nil {
				continue
			}
			price := pricingFor(r.Message.Model)
			u := r.Message.Usage
			cost := float64(u.InputTokens)/1e6*price.in +
				float64(u.OutputTokens)/1e6*price.out +
				float64(u.CacheReadInputTokens)/1e6*price.cacheRead +
				float64(u.CacheCreationInputTokens)/1e6*price.cacheWrite

			addToModel(allByModel, r.Message.Model, u.InputTokens, u.OutputTokens,
				u.CacheReadInputTokens, u.CacheCreationInputTokens, cost)
			if ts.After(cutoffWeek) {
				addToModel(weekByModel, r.Message.Model, u.InputTokens, u.OutputTokens,
					u.CacheReadInputTokens, u.CacheCreationInputTokens, cost)
				if snap.WeekWindowStart.IsZero() || ts.Before(snap.WeekWindowStart) {
					snap.WeekWindowStart = ts
				}
				// Roll up by session for the top-sessions panel.
				sid := r.SessionID
				if sid == "" {
					sid = strings.TrimSuffix(filepath.Base(path), ".jsonl")
				}
				sc, ok := sessionTotals[sid]
				if !ok {
					sc = &SessionCost{SID: sid, CWD: r.CWD}
					sessionTotals[sid] = sc
				}
				sc.CostUSD += cost
				if ts.After(sc.Last) {
					sc.Last = ts
				}
				if sc.CWD == "" {
					sc.CWD = r.CWD
				}
			}
			if ts.After(cutoff5h) {
				addToModel(sessionByModel, r.Message.Model, u.InputTokens, u.OutputTokens,
					u.CacheReadInputTokens, u.CacheCreationInputTokens, cost)
				if snap.SessionWindowStart.IsZero() || ts.Before(snap.SessionWindowStart) {
					snap.SessionWindowStart = ts
				}
			}
			if ts.After(cutoff24h) {
				addToModel(last24hByModel, r.Message.Model, u.InputTokens, u.OutputTokens,
					u.CacheReadInputTokens, u.CacheCreationInputTokens, cost)
				last24hCount++
				last24hCost += cost
				if u.CacheReadInputTokens > 150_000 {
					last24hLongCtx++
				}
				if r.IsSidechain {
					last24hSidechainCost += cost
				}
			}
			if ts.After(startOfDay) {
				addToModel(todayByModel, r.Message.Model, u.InputTokens, u.OutputTokens,
					u.CacheReadInputTokens, u.CacheCreationInputTokens, cost)
			}
		}
		return nil
	})
	if err != nil {
		return snap, err
	}

	snap.Today = finalize(todayByModel)
	snap.Last24h = finalize(last24hByModel)
	snap.ThisWeek = finalize(weekByModel)
	snap.Session = finalize(sessionByModel)
	snap.AllTime = finalize(allByModel)
	snap.TopSessions = topSessions(sessionTotals, 5)

	if last24hCount > 0 {
		snap.LongContextPct = float64(last24hLongCtx) / float64(last24hCount) * 100
	}
	if last24hCost > 0 {
		snap.SubagentPct = last24hSidechainCost / last24hCost * 100
	}
	return snap, nil
}

func addToModel(m map[string]*ModelStats, model string, in, out, cr, cw int64, cost float64) {
	s, ok := m[model]
	if !ok {
		s = &ModelStats{Model: model}
		m[model] = s
	}
	s.InputTokens += in
	s.OutputTokens += out
	s.CacheRead += cr
	s.CacheWrite += cw
	s.CostUSD += cost
}

func finalize(m map[string]*ModelStats) PeriodStats {
	var p PeriodStats
	for _, s := range m {
		p.ByModel = append(p.ByModel, *s)
		p.CostUSD += s.CostUSD
		p.InputTokens += s.InputTokens
		p.OutputTokens += s.OutputTokens
		p.CacheRead += s.CacheRead
		p.CacheWrite += s.CacheWrite
	}
	sort.Slice(p.ByModel, func(i, j int) bool {
		return p.ByModel[i].CostUSD > p.ByModel[j].CostUSD
	})
	return p
}

func topSessions(m map[string]*SessionCost, n int) []SessionCost {
	out := make([]SessionCost, 0, len(m))
	for _, sc := range m {
		out = append(out, *sc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CostUSD > out[j].CostUSD })
	if len(out) > n {
		out = out[:n]
	}
	return out
}

func projectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".claude", "projects")
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("claude projects dir not found: %w", err)
	}
	return dir, nil
}

// FormatUSD renders a cost amount as "$1.23" or "$1.2k" for big numbers.
func FormatUSD(v float64) string {
	switch {
	case v >= 1000:
		return fmt.Sprintf("$%.1fk", v/1000)
	case v >= 10:
		return fmt.Sprintf("$%.0f", v)
	default:
		return fmt.Sprintf("$%.2f", v)
	}
}

// FormatMoney renders a USD amount converted to the target currency.
// If currency is empty/USD or rate<=0, it falls back to FormatUSD.
// Symbol is prefixed; ISO code is used for currencies without a known
// symbol so any local currency works without listing it explicitly.
func FormatMoney(usd float64, currency string, rate float64) string {
	if currency == "" || currency == "USD" || rate <= 0 {
		return FormatUSD(usd)
	}
	v := usd * rate
	sym := currencySymbol(currency)
	switch {
	case v >= 1000:
		return fmt.Sprintf("%s%.1fk", sym, v/1000)
	case v >= 10:
		return fmt.Sprintf("%s%.0f", sym, v)
	default:
		return fmt.Sprintf("%s%.2f", sym, v)
	}
}

func currencySymbol(c string) string {
	switch c {
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "JPY":
		return "¥"
	case "CHF":
		return "CHF "
	case "USD", "":
		return "$"
	}
	return c + " "
}

// FormatTokens renders a token count compactly: 1.4M, 47.6k, 123.
func FormatTokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// ProgressBar renders a fixed-width bar filled to pct (0..100+).
// Anything above 100 caps the bar at full width but the caller can
// surface the actual percentage separately.
func ProgressBar(pct float64, width int) string {
	if width < 4 {
		width = 4
	}
	filled := int(pct / 100 * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	// Use a half-block on the boundary cell for finer granularity.
	half := ""
	if filled < width {
		frac := pct/100*float64(width) - float64(filled)
		if frac >= 0.5 {
			half = "▌"
		}
	}
	empty := width - filled - len(half)
	if empty < 0 {
		empty = 0
	}
	return strings.Repeat("█", filled) + half + strings.Repeat(" ", empty)
}

// FormatReset renders a reset timestamp relative to now in a compact form
// matching what Claude Code's /usage shows ("10:50am" same-day, "May 17, 7pm"
// otherwise). Returns "" if t is zero.
func FormatReset(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	return t.Format("Mon Jan 2, 15:04")
}

// ModelShort returns a short display name for the model id.
func ModelShort(model string) string {
	switch {
	case strings.HasPrefix(model, "claude-opus-4-7"):
		return "opus 4.7"
	case strings.HasPrefix(model, "claude-opus-4-6"):
		return "opus 4.6"
	case strings.HasPrefix(model, "claude-opus"):
		return "opus"
	case strings.HasPrefix(model, "claude-sonnet-4-6"):
		return "sonnet 4.6"
	case strings.HasPrefix(model, "claude-sonnet-4-5"):
		return "sonnet 4.5"
	case strings.HasPrefix(model, "claude-sonnet"):
		return "sonnet"
	case strings.HasPrefix(model, "claude-haiku-4-5"):
		return "haiku 4.5"
	case strings.HasPrefix(model, "claude-haiku"):
		return "haiku"
	}
	return model
}
