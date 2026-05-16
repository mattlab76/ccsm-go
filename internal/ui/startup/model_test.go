package startup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/model"
)

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// setup builds an in-memory ccsm DB and points $HOME at a temp dir so
// claude.IsSessionValid finds no transcripts unless a test stages them.
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

// seedSession inserts a session with the given fields. Created defaults
// to now if zero.
func seedSession(t *testing.T, database *db.DB, sid, cwd string, created time.Time) {
	t.Helper()
	if created.IsZero() {
		created = time.Now()
	}
	s := &model.Session{
		SID:       sid,
		CWD:       cwd,
		Subject:   "test " + sid,
		CreatedAt: created,
	}
	if err := db.SaveSession(database, s); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
}

// stageJSONL creates ~/.claude/projects/<slug>/<sid>.jsonl so
// claude.IsSessionValid returns true for the SID.
func stageJSONL(t *testing.T, home, cwd, sid string) {
	t.Helper()
	slug := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(home, ".claude", "projects", slug)
	mkdirAll(t, dir)
	writeFile(t, filepath.Join(dir, sid+".jsonl"),
		`{"type":"user","cwd":"`+cwd+`"}`)
}

// ─────────────────────────────────────────────────────────────────────

func TestHasWork_EmptyDB(t *testing.T) {
	database, _ := setup(t)
	m := New(database)
	if m.HasWork() {
		t.Errorf("HasWork should be false with no sessions")
	}
}

func TestHasWork_AllValidSessions(t *testing.T) {
	database, home := setup(t)
	// Existing dir (use $HOME) + recent + valid JSONL → no flag.
	seedSession(t, database, "aaa-bbb-ccc", home, time.Now())
	stageJSONL(t, home, home, "aaa-bbb-ccc")

	m := New(database)
	if m.HasWork() {
		t.Errorf("HasWork should be false for healthy sessions")
	}
}

func TestHasWork_ExpiredSession(t *testing.T) {
	database, home := setup(t)
	// Existing dir, but no JSONL → expired.
	seedSession(t, database, "ghost-sid", home, time.Now())

	m := New(database)
	if !m.HasWork() {
		t.Errorf("HasWork should be true when transcript is missing")
	}
	expired, missing, old := m.countByReason()
	if expired != 1 || missing != 0 || old != 0 {
		t.Errorf("expected expired=1, got expired=%d missing=%d old=%d",
			expired, missing, old)
	}
}

func TestHasWork_MissingDirSession(t *testing.T) {
	database, home := setup(t)
	// JSONL exists, but the CWD is gone.
	seedSession(t, database, "valid-sid", "/does/not/exist", time.Now())
	stageJSONL(t, home, "/does/not/exist", "valid-sid")

	m := New(database)
	if !m.HasWork() {
		t.Errorf("HasWork should be true when working dir is missing")
	}
	_, missing, _ := m.countByReason()
	if missing != 1 {
		t.Errorf("expected missing=1, got %d", missing)
	}
}

func TestHasWork_TooOldSession(t *testing.T) {
	database, home := setup(t)
	// Configure cleanup_days = 30, session created 60 days ago.
	settings := model.DefaultSettings()
	settings.CleanupDays = 30
	if err := db.SaveSettings(database, &settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	old := time.Now().AddDate(0, 0, -60)
	seedSession(t, database, "old-sid", home, old)
	stageJSONL(t, home, home, "old-sid")

	m := New(database)
	if !m.HasWork() {
		t.Errorf("HasWork should be true for session older than cleanup_days")
	}
	_, _, oldCount := m.countByReason()
	if oldCount != 1 {
		t.Errorf("expected old=1, got %d", oldCount)
	}
}

func TestHasWork_CleanupDisabledByZeroSetting(t *testing.T) {
	database, home := setup(t)
	// cleanup_days = 0 → age check disabled even for very old sessions.
	settings := model.DefaultSettings()
	settings.CleanupDays = 0
	if err := db.SaveSettings(database, &settings); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	old := time.Now().AddDate(-5, 0, 0)
	seedSession(t, database, "ancient-sid", home, old)
	stageJSONL(t, home, home, "ancient-sid")

	m := New(database)
	if m.HasWork() {
		t.Errorf("HasWork should be false when cleanup_days = 0")
	}
}

func TestDismissedSessionsAreSkipped(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "dismissed-sid", home, time.Now())
	// No JSONL → would be expired, BUT it's dismissed.
	if err := db.AddDismissed(database, "dismissed-sid"); err != nil {
		t.Fatalf("AddDismissed: %v", err)
	}

	m := New(database)
	if m.HasWork() {
		t.Errorf("dismissed sessions should not surface in the check")
	}
}

// ─── Cursor navigation ──────────────────────────────────────────────

func TestCursorDefaultsToSkip(t *testing.T) {
	database, _ := setup(t)
	m := New(database)
	if m.cursor != actionSkip {
		t.Errorf("default cursor = %d, want %d (skip)", m.cursor, actionSkip)
	}
}

func TestCursorUpDown(t *testing.T) {
	database, _ := setup(t)
	m := New(database)
	// Skip → Dismiss → Purge → (clamped at Purge)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != actionDismiss {
		t.Errorf("after up: cursor = %d, want %d", m.cursor, actionDismiss)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != actionPurge {
		t.Errorf("after second up: cursor = %d, want %d", m.cursor, actionPurge)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != actionPurge {
		t.Errorf("cursor should clamp at top, got %d", m.cursor)
	}
	// Now go back down past Skip — should clamp.
	for i := 0; i < 5; i++ {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.cursor != actionSkip {
		t.Errorf("cursor should clamp at bottom (skip), got %d", m.cursor)
	}
}

func TestHomeEndJumps(t *testing.T) {
	database, _ := setup(t)
	m := New(database)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != 0 {
		t.Errorf("Home should jump to 0, got %d", m.cursor)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursor != actionCount-1 {
		t.Errorf("End should jump to last, got %d", m.cursor)
	}
}

// ─── Action invocation ───────────────────────────────────────────────

// drainCmd runs cmd if non-nil and returns the resulting tea.Msg. nil
// in / nil out means the action did nothing.
func drainCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestSkipEmitsSwitchToMainMenu(t *testing.T) {
	database, _ := setup(t)
	m := New(database)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	msg := drainCmd(cmd)
	sw, ok := msg.(SwitchViewMsg)
	if !ok {
		t.Fatalf("expected SwitchViewMsg, got %T", msg)
	}
	if sw.View != 0 {
		t.Errorf("expected View=0 (main menu), got %d", sw.View)
	}
}

func TestPurgeDeletesFlaggedSessions(t *testing.T) {
	database, home := setup(t)
	// Two expired sessions.
	seedSession(t, database, "sid-1", home, time.Now())
	seedSession(t, database, "sid-2", home, time.Now())

	m := New(database)
	if !m.HasWork() {
		t.Fatal("setup: should have work")
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if msg := drainCmd(cmd); msg == nil {
		t.Fatal("purge should emit a switch cmd")
	}

	// Verify both rows are gone.
	if _, err := db.GetSession(database, "sid-1"); err == nil {
		t.Error("sid-1 should be deleted after purge")
	}
	if _, err := db.GetSession(database, "sid-2"); err == nil {
		t.Error("sid-2 should be deleted after purge")
	}
}

func TestDismissAddsToDismissedTable(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "sid-x", home, time.Now())

	m := New(database)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if msg := drainCmd(cmd); msg == nil {
		t.Fatal("dismiss should emit a switch cmd")
	}

	dismissed, err := db.GetDismissed(database)
	if err != nil {
		t.Fatalf("GetDismissed: %v", err)
	}
	if len(dismissed) != 1 || dismissed[0] != "sid-x" {
		t.Errorf("expected dismissed=[sid-x], got %v", dismissed)
	}
	// Session row itself should still be present (dismiss ≠ delete).
	if _, err := db.GetSession(database, "sid-x"); err != nil {
		t.Errorf("session row should remain after dismiss, got %v", err)
	}
}

func TestEnterActivatesCursorAction(t *testing.T) {
	database, home := setup(t)
	seedSession(t, database, "enter-sid", home, time.Now())

	m := New(database)
	// Move cursor to Purge, then Enter.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursor != actionPurge {
		t.Fatalf("cursor should be on purge after Home, got %d", m.cursor)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if msg := drainCmd(cmd); msg == nil {
		t.Fatal("Enter should emit a switch cmd")
	}
	// Purge action executed.
	if _, err := db.GetSession(database, "enter-sid"); err == nil {
		t.Error("Enter on purge should delete the flagged session")
	}
}

func TestEscDismissesAsSkip(t *testing.T) {
	database, _ := setup(t)
	m := New(database)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	msg := drainCmd(cmd)
	if _, ok := msg.(SwitchViewMsg); !ok {
		t.Errorf("Esc should emit SwitchViewMsg, got %T", msg)
	}
}
