package log

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/model"
)

func setup(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// stripANSI removes color escapes so we can substring-check rendered text.
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

func TestNew_EmptyState(t *testing.T) {
	d := setup(t)
	m := New(d).SetSize(120, 40)
	out := stripANSI(m.View())
	if !strings.Contains(out, "Activity Log") {
		t.Errorf("missing title in: %s", out)
	}
}

func TestNew_RendersEntries(t *testing.T) {
	d := setup(t)
	db.LogAction(d, model.ActionNew, "subject1")
	db.LogAction(d, model.ActionResume, "subject2")
	db.LogAction(d, model.ActionDelete, "subject3")

	m := New(d).SetSize(120, 40)
	out := stripANSI(m.View())
	for _, want := range []string{"subject1", "subject2", "subject3"} {
		if !strings.Contains(out, want) {
			t.Errorf("entry %q missing in: %s", want, out)
		}
	}
	// Action tags rendered.
	for _, want := range []string{"NEW", "RESUME", "DELETE"} {
		if !strings.Contains(out, want) {
			t.Errorf("action tag %q missing in: %s", want, out)
		}
	}
}

func TestColorAction_AllKnownActions(t *testing.T) {
	// Just verify it doesn't panic and returns non-empty for each.
	for _, action := range []string{
		model.ActionNew, model.ActionResume, model.ActionSave,
		model.ActionDelete, model.ActionCleanup, model.ActionSettings,
		"UNKNOWN",
	} {
		out := colorAction(action)
		if out == "" {
			t.Errorf("colorAction(%q) returned empty", action)
		}
		if !strings.Contains(stripANSI(out), action) {
			t.Errorf("colorAction(%q) does not contain action label in: %q", action, out)
		}
	}
}

// ─── Sticky header + scroll ─────────────────────────────────────────

func TestStickyHeaderAlwaysPresent(t *testing.T) {
	d := setup(t)
	for i := 0; i < 50; i++ {
		db.LogAction(d, model.ActionNew, "entry")
	}
	m := New(d).SetSize(120, 30) // small height, content overflows
	out := stripANSI(m.View())
	// The logo line is part of the sticky header.
	if !strings.Contains(out, "CCSM") && !strings.Contains(out, "Activity Log") {
		t.Errorf("header missing from view: %s", out[:min(len(out), 200)])
	}

	// Scroll down; header should still be there.
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	out2 := stripANSI(m.View())
	if !strings.Contains(out2, "Activity Log") {
		t.Errorf("header lost after scrolling")
	}
}

func TestScrollBoundsRespected(t *testing.T) {
	d := setup(t)
	for i := 0; i < 100; i++ {
		db.LogAction(d, model.ActionNew, "entry")
	}
	m := New(d).SetSize(120, 30)

	// Up at start: no-op, no panic.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.scroll != 0 {
		t.Errorf("scroll should stay at 0, got %d", m.scroll)
	}
	// End jumps to max.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.scroll != m.maxScroll() {
		t.Errorf("End should land at maxScroll (%d), got %d",
			m.maxScroll(), m.scroll)
	}
	// Down past max: clamps.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.scroll != m.maxScroll() {
		t.Errorf("scroll should clamp at maxScroll, got %d", m.scroll)
	}
}

func TestQReturnsToMainMenu(t *testing.T) {
	m := New(setup(t)).SetSize(120, 40)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should emit switch cmd")
	}
	msg := cmd()
	sw, ok := msg.(SwitchViewMsg)
	if !ok {
		t.Fatalf("expected SwitchViewMsg, got %T", msg)
	}
	if sw.View != 0 {
		t.Errorf("expected View=0, got %d", sw.View)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
