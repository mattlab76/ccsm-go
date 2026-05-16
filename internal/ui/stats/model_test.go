package stats

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/model"
)

func setup(t *testing.T) *db.DB {
	t.Helper()
	// Use a temp $HOME so usage.Compute() doesn't scan the real
	// ~/.claude/projects (it would still work but be much slower).
	t.Setenv("HOME", t.TempDir())
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func seedSession(t *testing.T, d *db.DB, sid, cwd string) {
	t.Helper()
	s := &model.Session{
		SID:       sid,
		CWD:       cwd,
		Subject:   "s-" + sid,
		CreatedAt: time.Now(),
	}
	if err := db.SaveSession(d, s); err != nil {
		t.Fatal(err)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == 0x1b {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ─────────────────────────────────────────────────────────────────────

func TestNew_EmptyState(t *testing.T) {
	d := setup(t)
	m := New(d).SetSize(120, 40)
	out := stripANSI(m.View())
	if !strings.Contains(out, "No sessions available") {
		t.Errorf("empty-state message missing in: %s", out)
	}
}

func TestNew_RendersOverviewWithSessions(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "s1", "/work/a")
	seedSession(t, d, "s2", "/work/b")
	m := New(d).SetSize(120, 60)
	out := stripANSI(m.View())

	if !strings.Contains(out, "Overview") {
		t.Errorf("missing Overview section in: %s", out[:min(len(out), 300)])
	}
	// Session count visible.
	if !strings.Contains(out, "Sessions") {
		t.Errorf("missing Sessions row")
	}
}

func TestPlanLineShownWhenConfigured(t *testing.T) {
	d := setup(t)
	settings := model.DefaultSettings()
	settings.PlanName = "Max 5x"
	settings.PlanMonthlyPrice = 100
	settings.Currency = "USD"
	if err := db.SaveSettings(d, &settings); err != nil {
		t.Fatal(err)
	}
	seedSession(t, d, "s1", "/work")

	m := New(d).SetSize(120, 80)
	out := stripANSI(m.View())
	if !strings.Contains(out, "Max 5x") {
		t.Errorf("plan name missing from overview in: %s", out)
	}
}

func TestMoneyConversionToEUR(t *testing.T) {
	d := setup(t)
	settings := model.DefaultSettings()
	settings.Currency = "EUR"
	settings.ExchangeRate = 0.92
	if err := db.SaveSettings(d, &settings); err != nil {
		t.Fatal(err)
	}
	seedSession(t, d, "s1", "/work")

	m := New(d).SetSize(120, 80)
	out := stripANSI(m.View())
	// Even with 0 tokens, the consumption table renders with €0.00 cells.
	if !strings.Contains(out, "€") {
		t.Errorf("expected EUR symbol in output: %s", out[:min(len(out), 500)])
	}
}

// ─── Sticky header + scroll ─────────────────────────────────────────

func TestHeaderStaysOnScroll(t *testing.T) {
	d := setup(t)
	for i := 0; i < 20; i++ {
		seedSession(t, d, "sid-"+string(rune('a'+i)), "/work")
	}
	m := New(d).SetSize(120, 25)

	out1 := stripANSI(m.View())
	if !strings.Contains(out1, "Statistics") {
		t.Errorf("header missing initially")
	}

	for i := 0; i < 50; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	out2 := stripANSI(m.View())
	if !strings.Contains(out2, "Statistics") {
		t.Errorf("header should remain on scroll")
	}
}

func TestScrollBounds(t *testing.T) {
	d := setup(t)
	for i := 0; i < 30; i++ {
		seedSession(t, d, "sid-"+string(rune('a'+i%26)), "/work")
	}
	m := New(d).SetSize(120, 25)

	// Up at start: stays at 0.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.scroll != 0 {
		t.Errorf("scroll should be 0 at top, got %d", m.scroll)
	}
	// End jumps to maxScroll.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.scroll != m.maxScroll() {
		t.Errorf("End should land at maxScroll, got %d (max=%d)",
			m.scroll, m.maxScroll())
	}
	// PgUp pulls back.
	prev := m.scroll
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.scroll >= prev {
		t.Errorf("PgUp should reduce scroll: was %d, now %d", prev, m.scroll)
	}
}

func TestQReturnsToMainMenu(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "s1", "/work")
	m := New(d).SetSize(120, 40)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should emit switch cmd")
	}
	msg := cmd()
	if _, ok := msg.(SwitchViewMsg); !ok {
		t.Errorf("expected SwitchViewMsg, got %T", msg)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
