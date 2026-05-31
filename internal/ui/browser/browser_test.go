package browser

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
)

func setupBrowser(t *testing.T) Model {
	t.Helper()
	database, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}

	sessions := []model.Session{
		{SID: "s1", CWD: "/tmp/alpha", Subject: "Alpha project", CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), Tags: "#go", TotalInputTokens: 500000},
		{SID: "s2", CWD: "/tmp/beta", Subject: "Beta bugfix", CreatedAt: time.Date(2026, 3, 22, 14, 0, 0, 0, time.UTC), Tags: "#python", TotalInputTokens: 100000},
		{SID: "s3", CWD: "/tmp/gamma", Subject: "Gamma feature", CreatedAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.UTC), Tags: "#go", TotalInputTokens: 300000},
	}
	for _, s := range sessions {
		db.SaveSession(database, &s)
	}

	m := New(database)
	m = m.SetSize(120, 40)
	return m
}

// --- Navigation Tests ---

func TestBrowser_InitialState(t *testing.T) {
	m := setupBrowser(t)
	if m.mode != modeNavigate {
		t.Errorf("initial mode = %d, want modeNavigate", m.mode)
	}
	if len(m.sessions) != 3 {
		t.Errorf("sessions = %d, want 3", len(m.sessions))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestBrowser_ArrowNavigation(t *testing.T) {
	m := setupBrowser(t)

	// Move down.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", m.cursor)
	}

	// Move down again.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("after 2x down: cursor = %d, want 2", m.cursor)
	}

	// Move down at bottom — should not go past end.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("down at bottom: cursor = %d, want 2", m.cursor)
	}

	// Move up.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("after up: cursor = %d, want 1", m.cursor)
	}
}

func TestBrowser_SelectionHighlight(t *testing.T) {
	m := setupBrowser(t)

	// Initial selection should be row 0.
	if !m.rows[0].Selected {
		t.Error("row 0 should be selected initially")
	}
	if m.rows[1].Selected {
		t.Error("row 1 should not be selected initially")
	}

	// Move down.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.rows[0].Selected {
		t.Error("row 0 should not be selected after down")
	}
	if !m.rows[1].Selected {
		t.Error("row 1 should be selected after down")
	}
}

// --- Preview Tests ---

func TestBrowser_EnterShowsPreview(t *testing.T) {
	m := setupBrowser(t)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modePreview {
		t.Errorf("after Enter: mode = %d, want modePreview", m.mode)
	}
}

func TestBrowser_PreviewEscReturns(t *testing.T) {
	m := setupBrowser(t)

	// Enter preview.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != modePreview {
		t.Fatal("should be in preview mode")
	}

	// Esc goes back to navigate.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.mode != modeNavigate {
		t.Errorf("after Esc in preview: mode = %d, want modeNavigate", m.mode)
	}
}

func TestBrowser_PreviewRender(t *testing.T) {
	m := setupBrowser(t)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := m.renderPreview()
	if view == "" {
		t.Error("preview should not be empty")
	}
}

// --- Sort Tests ---

func TestBrowser_SortByDate(t *testing.T) {
	m := setupBrowser(t)

	// Default sort is by date (newest first).
	if m.sortMode != sortByDate {
		t.Errorf("initial sortMode = %d, want sortByDate", m.sortMode)
	}
	// Newest session (s2, 2026-03-22) should be first.
	if m.sessions[0].SID != "s2" {
		t.Errorf("first session = %q, want s2 (newest)", m.sessions[0].SID)
	}
}

func TestBrowser_SortByName(t *testing.T) {
	m := setupBrowser(t)

	// Press 's' to cycle to sortByName.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.sortMode != sortByName {
		t.Errorf("sortMode = %d, want sortByName", m.sortMode)
	}
	// Alphabetically: Alpha, Beta, Gamma.
	if m.sessions[0].Subject != "Alpha project" {
		t.Errorf("first by name = %q, want Alpha project", m.sessions[0].Subject)
	}
}

func TestBrowser_SortByTokens(t *testing.T) {
	m := setupBrowser(t)

	// Press 's' twice to get to sortByTokens.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.sortMode != sortByTokens {
		t.Errorf("sortMode = %d, want sortByTokens", m.sortMode)
	}
	// Most tokens first: s1 (500k), s3 (300k), s2 (100k).
	if m.sessions[0].SID != "s1" {
		t.Errorf("first by tokens = %q, want s1", m.sessions[0].SID)
	}
}

func TestBrowser_SortCyclesBack(t *testing.T) {
	m := setupBrowser(t)

	// 3 presses cycles back to sortByDate.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.sortMode != sortByDate {
		t.Errorf("after 3 sort cycles: sortMode = %d, want sortByDate", m.sortMode)
	}
}

func TestBrowser_SortResetsCursor(t *testing.T) {
	m := setupBrowser(t)

	// Move cursor down.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatal("cursor should be 1")
	}

	// Sort resets cursor to 0.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if m.cursor != 0 {
		t.Errorf("sort should reset cursor to 0, got %d", m.cursor)
	}
}

// --- Search Tests ---

func TestBrowser_SearchMode(t *testing.T) {
	m := setupBrowser(t)

	// '/' enters search mode.
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if m.mode != modeSearch {
		t.Errorf("after /: mode = %d, want modeSearch", m.mode)
	}
}

func TestBrowser_SearchEscCancels(t *testing.T) {
	m := setupBrowser(t)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.mode != modeNavigate {
		t.Errorf("after Esc: mode = %d, want modeNavigate", m.mode)
	}
}

// --- View Tests ---

func TestBrowser_ViewNotEmpty(t *testing.T) {
	m := setupBrowser(t)
	view := m.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestBrowser_SortModeString(t *testing.T) {
	if sortByDate.String() == "" {
		t.Error("sortByDate.String() should not be empty")
	}
	if sortByName.String() == "" {
		t.Error("sortByName.String() should not be empty")
	}
	if sortByTokens.String() == "" {
		t.Error("sortByTokens.String() should not be empty")
	}
}

// TestBrowser_SortKeepsStatusWithSession is a regression test: sorting used
// to reorder m.sessions but reassign m.rows[i].Session without moving the
// row's Status, so [!]/[?] markers ended up on the wrong rows after a
// name/token sort. After the fix, each row's Status must stay paired with
// its session, and m.sessions must stay aligned with the displayed rows.
func TestBrowser_SortKeepsStatusWithSession(t *testing.T) {
	mk := func(sid, subject string, tok int64, st components.SessionStatus) components.TableRow {
		return components.TableRow{
			Session: model.Session{SID: sid, Subject: subject, TotalInputTokens: tok,
				CreatedAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)},
			Status: st,
		}
	}
	m := Model{
		rows: []components.TableRow{
			mk("s1", "Zebra", 100, components.StatusExpired),
			mk("s2", "Alpha", 300, components.StatusOK),
			mk("s3", "Mango", 200, components.StatusMissingDir),
		},
	}
	m.sessions = make([]model.Session, len(m.rows))
	want := map[string]components.SessionStatus{}
	for i, r := range m.rows {
		m.sessions[i] = r.Session
		want[r.Session.SID] = r.Status
	}

	m.sortMode = sortByName
	m.applySortOrder()

	// Rows now in alphabetical order: Alpha, Mango, Zebra.
	gotOrder := []string{m.rows[0].Session.Subject, m.rows[1].Session.Subject, m.rows[2].Session.Subject}
	wantOrder := []string{"Alpha", "Mango", "Zebra"}
	for i := range wantOrder {
		if gotOrder[i] != wantOrder[i] {
			t.Fatalf("sorted order = %v, want %v", gotOrder, wantOrder)
		}
	}
	// Each row's status must still match its own session, and m.sessions
	// must mirror the rows.
	for i, r := range m.rows {
		if r.Status != want[r.Session.SID] {
			t.Errorf("row %d (%s): status = %v, want %v",
				i, r.Session.SID, r.Status, want[r.Session.SID])
		}
		if m.sessions[i].SID != r.Session.SID {
			t.Errorf("row %d: m.sessions SID = %s, row SID = %s (out of sync)",
				i, m.sessions[i].SID, r.Session.SID)
		}
	}
}
