package mainmenu

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/model"
)

func setup(t *testing.T) *db.DB {
	t.Helper()
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
		Subject:   "subj-" + sid,
		CreatedAt: time.Now(),
	}
	if err := db.SaveSession(d, s); err != nil {
		t.Fatal(err)
	}
}

// ─── Cursor: sessions + menu items ─────────────────────────────────

func TestCursorStartsOnFirstRow(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "s1", "/work")
	m := New(d)
	if m.cursor != 0 {
		t.Errorf("cursor should start at 0, got %d", m.cursor)
	}
}

func TestCursorWalksOverSessionsAndIntoMenuItems(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "s1", "/work/a")
	seedSession(t, d, "s2", "/work/b")
	m := New(d)
	// 2 sessions + 7 menu actions = 9 rows. Walk all the way down.
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.cursor != len(m.sessions)+len(m.actions)-1 {
		t.Errorf("cursor should clamp at last row (%d), got %d",
			len(m.sessions)+len(m.actions)-1, m.cursor)
	}
	// Walk back to top.
	for i := 0; i < 20; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	if m.cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", m.cursor)
	}
}

func TestHomeEndJumps(t *testing.T) {
	d := setup(t)
	for _, s := range []string{"a", "b", "c"} {
		seedSession(t, d, s, "/w")
	}
	m := New(d)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursor != len(m.sessions)+len(m.actions)-1 {
		t.Errorf("End should jump to last row, got %d", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Errorf("Home should land at 0, got %d", m.cursor)
	}
}

// ─── Enter activation ───────────────────────────────────────────────

func TestEnterOnSessionEmitsResume(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "resume-me", "/work/here")
	m := New(d)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on session should emit cmd")
	}
	msg := cmd()
	rsm, ok := msg.(ResumeSessionMsg)
	if !ok {
		t.Fatalf("expected ResumeSessionMsg, got %T", msg)
	}
	if rsm.SID != "resume-me" || rsm.CWD != "/work/here" {
		t.Errorf("wrong resume target: %+v", rsm)
	}
}

func TestEnterOnMenuItemEmitsSwitchView(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "s1", "/w")
	m := New(d)
	// Walk past sessions onto first menu action ("n" → newSession).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // onto menu item 0
	if m.cursor != len(m.sessions) {
		t.Fatalf("cursor should be at first menu item (%d), got %d",
			len(m.sessions), m.cursor)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on menu item should emit cmd")
	}
	sw, ok := cmd().(SwitchViewMsg)
	if !ok {
		t.Fatalf("expected SwitchViewMsg, got %T", cmd())
	}
	if sw.View != viewNewSession {
		t.Errorf("first menu item should switch to newSession, got View=%d", sw.View)
	}
}

// ─── Letter shortcuts ───────────────────────────────────────────────

func TestLetterShortcutsTriggerCorrectAction(t *testing.T) {
	tests := map[rune]int{
		'n': viewNewSession,
		's': viewBrowser,
		'd': viewDelete,
		'i': viewStats,
		'l': viewLog,
		'c': viewSettings,
	}
	for letter, wantView := range tests {
		d := setup(t)
		seedSession(t, d, "x", "/w")
		m := New(d)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{letter}})
		if cmd == nil {
			t.Errorf("key %c: no cmd emitted", letter)
			continue
		}
		msg := cmd()
		sw, ok := msg.(SwitchViewMsg)
		if !ok {
			t.Errorf("key %c: expected SwitchViewMsg, got %T", letter, msg)
			continue
		}
		if sw.View != wantView {
			t.Errorf("key %c: View = %d, want %d", letter, sw.View, wantView)
		}
	}
}

func TestQuitShortcutEmitsTeaQuit(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "s", "/w")
	m := New(d)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should emit cmd")
	}
	msg := cmd()
	// tea.QuitMsg is the concrete msg returned by tea.Quit.
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

// ─── Number-key quick resume ────────────────────────────────────────

func TestNumberKeyResumesNthSession(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "first", "/a")
	seedSession(t, d, "second", "/b")
	seedSession(t, d, "third", "/c")
	m := New(d)

	// Sessions are returned newest-first; verify by inspecting m.sessions.
	if len(m.sessions) < 2 {
		t.Fatalf("expected at least 2 sessions, got %d", len(m.sessions))
	}
	wantSID := m.sessions[1].SID

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if cmd == nil {
		t.Fatal("'2' should emit cmd")
	}
	rsm, ok := cmd().(ResumeSessionMsg)
	if !ok {
		t.Fatalf("expected ResumeSessionMsg, got %T", cmd())
	}
	if rsm.SID != wantSID {
		t.Errorf("'2' should resume 2nd session (%s), got %s", wantSID, rsm.SID)
	}
}

func TestNumberKeyOutOfRangeIsNoOp(t *testing.T) {
	d := setup(t)
	seedSession(t, d, "only", "/w")
	m := New(d)
	// Only one session; pressing '5' should not panic and should emit nothing.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if cmd != nil {
		t.Errorf("out-of-range number should be a no-op, got cmd")
	}
}

// ─── Async usage banner ─────────────────────────────────────────────

func TestInitReturnsCmdForUsageCompute(t *testing.T) {
	d := setup(t)
	m := New(d)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init should return a cmd to compute usage in background")
	}
	msg := cmd()
	if _, ok := msg.(usageReadyMsg); !ok {
		t.Errorf("Init cmd should produce usageReadyMsg, got %T", msg)
	}
}

func TestUsageReadyMsgFlipsBannerToReady(t *testing.T) {
	d := setup(t)
	m := New(d)
	if m.usageReady {
		t.Error("usageReady should start false")
	}
	cmd := m.Init()
	msg := cmd()
	m, _ = m.Update(msg)
	if !m.usageReady {
		t.Error("usageReady should be true after the cmd returns")
	}
}
