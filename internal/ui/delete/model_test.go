package delete

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
)

func setup(t *testing.T) (*db.DB, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database, home
}

func seedSession(t *testing.T, database *db.DB, sid, cwd string) {
	t.Helper()
	s := &model.Session{
		SID:       sid,
		CWD:       cwd,
		Subject:   "subj-" + sid,
		CreatedAt: time.Now(),
	}
	if err := db.SaveSession(database, s); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
}

func stageJSONL(t *testing.T, home, cwd, sid string) {
	t.Helper()
	slug := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(home, ".claude", "projects", slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, sid+".jsonl"),
		[]byte(`{"type":"user"}`), 0644); err != nil {
		t.Fatal(err)
	}
}

// ─────────────────────────────────────────────────────────────────────

func TestNew_LoadsAllSessions(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "sid-a", home)
	seedSession(t, database, "sid-b", home)
	seedSession(t, database, "sid-c", home)

	m := New(database)
	if len(m.sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(m.sessions))
	}
}

func TestNew_MarksExpiredSessions(t *testing.T) {
	database, home := setup(t)
	// One with JSONL (valid), one without (expired).
	seedSession(t, database, "valid-sid", home)
	stageJSONL(t, home, home, "valid-sid")
	seedSession(t, database, "ghost-sid", home)

	m := New(database)
	var expired int
	for _, r := range m.rows {
		if r.Status == components.StatusExpired {
			expired++
		}
	}
	if expired != 1 {
		t.Errorf("expected 1 expired row, got %d", expired)
	}
}

// ─── Cursor navigation ──────────────────────────────────────────────

func TestCursorUpDown(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "s1", home)
	seedSession(t, database, "s2", home)
	seedSession(t, database, "s3", home)
	m := New(database).SetSize(120, 40)

	if m.cursor != 0 {
		t.Errorf("cursor should start at 0, got %d", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("after two downs: cursor = %d, want 2", m.cursor)
	}
	// Clamped at the bottom.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("cursor should clamp at last row, got %d", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("after up: cursor = %d, want 1", m.cursor)
	}
}

func TestHomeEnd(t *testing.T) {
	database, home := setup(t)
	for _, s := range []string{"s1", "s2", "s3", "s4", "s5"} {
		seedSession(t, database, s, home)
	}
	m := New(database).SetSize(120, 40)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursor != 4 {
		t.Errorf("End should land on last row (4), got %d", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Errorf("Home should land on 0, got %d", m.cursor)
	}
}

// ─── Marking ────────────────────────────────────────────────────────

func TestSpaceTogglesMark(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "s1", home)
	m := New(database).SetSize(120, 40)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !m.marked[0] {
		t.Errorf("row 0 should be marked after space")
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.marked[0] {
		t.Errorf("row 0 should be unmarked after second space")
	}
}

func TestMarkAllExpired(t *testing.T) {
	database, home := setup(t)
	// Three sessions: two expired, one valid.
	seedSession(t, database, "exp-1", home)
	seedSession(t, database, "exp-2", home)
	seedSession(t, database, "valid-1", home)
	stageJSONL(t, home, home, "valid-1")

	m := New(database).SetSize(120, 40)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if len(m.marked) != 2 {
		t.Errorf("expected 2 marked (both expired), got %d", len(m.marked))
	}
	// Pressing 'a' again should undo (all matching already marked).
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(m.marked) != 0 {
		t.Errorf("'a' twice should unmark; got %d still marked", len(m.marked))
	}
}

func TestMarkAll(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "s1", home)
	seedSession(t, database, "s2", home)
	seedSession(t, database, "s3", home)
	m := New(database).SetSize(120, 40)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if len(m.marked) != 3 {
		t.Errorf("'A' should mark all 3, got %d", len(m.marked))
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	if len(m.marked) != 0 {
		t.Errorf("'A' twice should unmark all, got %d", len(m.marked))
	}
}

// ─── Delete confirmation flow ───────────────────────────────────────

func TestEnterOnUnmarkedMarksCurrentAndGoesToConfirm(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "s1", home)
	m := New(database).SetSize(120, 40)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateConfirm {
		t.Errorf("expected stateConfirm, got %v", m.state)
	}
	if !m.marked[0] {
		t.Errorf("row 0 should be auto-marked when Enter pressed with nothing marked")
	}
}

func TestConfirmYesDeletesFromDB(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "doomed", home)
	m := New(database).SetSize(120, 40)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // → confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if _, err := db.GetSession(database, "doomed"); err == nil {
		t.Error("session should be gone after y confirm")
	}
	if m.state != stateDone {
		t.Errorf("expected stateDone after delete, got %v", m.state)
	}
}

func TestConfirmEnterAlsoDeletes(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "doomed2", home)
	m := New(database).SetSize(120, 40)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // → confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // Enter on confirm = yes
	if _, err := db.GetSession(database, "doomed2"); err == nil {
		t.Error("Enter on confirm should delete the session")
	}
}

func TestConfirmNoCancels(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "spared", home)
	m := New(database).SetSize(120, 40)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if _, err := db.GetSession(database, "spared"); err != nil {
		t.Errorf("session should still exist after n, got %v", err)
	}
	if m.state != stateSelect {
		t.Errorf("n should return to stateSelect, got %v", m.state)
	}
	if len(m.marked) != 0 {
		t.Errorf("marks should be cleared after cancel, got %d", len(m.marked))
	}
}

func TestQReturnsToMainMenu(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "s1", home)
	m := New(database).SetSize(120, 40)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q should emit a switch cmd")
	}
	msg := cmd()
	sw, ok := msg.(SwitchViewMsg)
	if !ok {
		t.Fatalf("expected SwitchViewMsg, got %T", msg)
	}
	if sw.View != 0 {
		t.Errorf("expected View=0 (main menu), got %d", sw.View)
	}
}

// ─── Viewport ───────────────────────────────────────────────────────

func TestViewportScrollsWhenManyRows(t *testing.T) {
	database, home := setup(t)
	for i := 0; i < 50; i++ {
		seedSession(t, database, "s-"+string(rune('a'+i%26))+string(rune('a'+(i/26)%26)), home)
	}
	m := New(database).SetSize(120, 25) // small terminal

	// Move cursor past the visible window.
	for i := 0; i < 30; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.offset == 0 {
		t.Errorf("offset should have moved, still 0")
	}
	if m.cursor < m.offset {
		t.Errorf("cursor %d should be at or below offset %d", m.cursor, m.offset)
	}
}
