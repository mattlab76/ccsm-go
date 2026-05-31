package db

import (
	"testing"
	"time"

	"github.com/mattlab76/ccsm-go/internal/model"
)

func TestOpenMemory(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error: %v", err)
	}
	defer db.Close()
}

func TestSchemaVersion(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error: %v", err)
	}
	defer db.Close()

	var version int
	err = db.QueryRow("SELECT MAX(version) FROM schema_version").Scan(&version)
	if err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if version != 1 {
		t.Errorf("schema_version = %d, want 1", version)
	}
}

func TestTablesExist(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error: %v", err)
	}
	defer db.Close()

	tables := []string{"sessions", "activity_log", "settings", "dismissed", "lifetime_tokens", "schema_version"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestSessionsColumns(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error: %v", err)
	}
	defer db.Close()

	rows, err := db.Query("PRAGMA table_info(sessions)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	wantCols := map[string]bool{
		"sid": false, "cwd": false, "subject": false, "created_at": false,
		"tags": false, "total_input_tokens": false, "total_output_tokens": false,
		"last_input_tokens": false, "last_output_tokens": false,
	}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dfltValue *string
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if _, ok := wantCols[name]; ok {
			wantCols[name] = true
		}
	}
	for col, found := range wantCols {
		if !found {
			t.Errorf("column %q not found in sessions table", col)
		}
	}
}

func TestLifetimeTokensInitialized(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error: %v", err)
	}
	defer db.Close()

	var totalIn, totalOut int64
	err = db.QueryRow("SELECT total_input, total_output FROM lifetime_tokens WHERE id=1").Scan(&totalIn, &totalOut)
	if err != nil {
		t.Fatalf("query lifetime_tokens: %v", err)
	}
	if totalIn != 0 || totalOut != 0 {
		t.Errorf("lifetime_tokens = (%d, %d), want (0, 0)", totalIn, totalOut)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	db, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error: %v", err)
	}
	defer db.Close()

	// Running migrate again should not error.
	if err := db.migrate(); err != nil {
		t.Errorf("second migrate() failed: %v", err)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error: %v", err)
	}
	defer database.Close()

	// Save settings.
	s := &model.Settings{CleanupDays: 60, LogDays: 180, Lang: "de"}
	if err := SaveSettings(database, s); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	// Read them back.
	got, err := GetSettings(database)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.CleanupDays != 60 {
		t.Errorf("CleanupDays = %d, want 60", got.CleanupDays)
	}
	if got.LogDays != 180 {
		t.Errorf("LogDays = %d, want 180", got.LogDays)
	}
	if got.Lang != "de" {
		t.Errorf("Lang = %q, want %q", got.Lang, "de")
	}
}

func TestDismissedRoundTrip(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory() error: %v", err)
	}
	defer database.Close()

	// Add dismissed sessions.
	if err := AddDismissed(database, "sid-1"); err != nil {
		t.Fatalf("AddDismissed: %v", err)
	}
	if err := AddDismissed(database, "sid-2"); err != nil {
		t.Fatalf("AddDismissed: %v", err)
	}
	// Duplicate should not error.
	if err := AddDismissed(database, "sid-1"); err != nil {
		t.Fatalf("AddDismissed duplicate: %v", err)
	}

	sids, err := GetDismissed(database)
	if err != nil {
		t.Fatalf("GetDismissed: %v", err)
	}
	if len(sids) != 2 {
		t.Errorf("GetDismissed() returned %d items, want 2", len(sids))
	}

	// Remove one.
	if err := RemoveDismissed(database, []string{"sid-1"}); err != nil {
		t.Fatalf("RemoveDismissed: %v", err)
	}
	sids, err = GetDismissed(database)
	if err != nil {
		t.Fatalf("GetDismissed after remove: %v", err)
	}
	if len(sids) != 1 {
		t.Errorf("GetDismissed() returned %d items after remove, want 1", len(sids))
	}
}

// --- Session CRUD Tests ---

func seedSessions(t *testing.T, database *DB) {
	t.Helper()
	sessions := []model.Session{
		{SID: "s1", CWD: "/home/user/project-a", Subject: "Docker setup", CreatedAt: time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC), Tags: "#infra #docker", TotalInputTokens: 100000, TotalOutputTokens: 5000, LastInputTokens: 50000, LastOutputTokens: 2500},
		{SID: "s2", CWD: "/home/user/project-b", Subject: "React frontend", CreatedAt: time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC), Tags: "#webapp", TotalInputTokens: 200000, TotalOutputTokens: 10000, LastInputTokens: 100000, LastOutputTokens: 5000},
		{SID: "s3", CWD: "/home/user/project-a", Subject: "Docker compose update", CreatedAt: time.Date(2026, 3, 20, 11, 0, 0, 0, time.UTC), Tags: "#infra #docker", TotalInputTokens: 50000, TotalOutputTokens: 2000, LastInputTokens: 25000, LastOutputTokens: 1000},
	}
	for _, s := range sessions {
		if err := SaveSession(database, &s); err != nil {
			t.Fatalf("SaveSession(%s): %v", s.SID, err)
		}
	}
}

func TestSessionListOrder(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	sessions, err := ListSessions(database, 0)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("got %d sessions, want 3", len(sessions))
	}
	// Newest first.
	if sessions[0].SID != "s3" {
		t.Errorf("first session = %q, want s3 (newest)", sessions[0].SID)
	}
	if sessions[2].SID != "s1" {
		t.Errorf("last session = %q, want s1 (oldest)", sessions[2].SID)
	}
}

func TestSessionListLimit(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	sessions, err := ListSessions(database, 2)
	if err != nil {
		t.Fatalf("ListSessions(2): %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("got %d sessions, want 2", len(sessions))
	}
}

func TestSessionGet(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	s, err := GetSession(database, "s2")
	if err != nil {
		t.Fatalf("GetSession(s2): %v", err)
	}
	if s.Subject != "React frontend" {
		t.Errorf("Subject = %q, want %q", s.Subject, "React frontend")
	}
	if s.TotalInputTokens != 200000 {
		t.Errorf("TotalInputTokens = %d, want 200000", s.TotalInputTokens)
	}
}

func TestSessionGetNotFound(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()

	_, err = GetSession(database, "nonexistent")
	if err == nil {
		t.Error("GetSession(nonexistent) should return error")
	}
}

func TestSessionUpsert(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	// Update existing session.
	updated := &model.Session{
		SID: "s2", CWD: "/home/user/project-b", Subject: "React frontend v2",
		CreatedAt: time.Date(2026, 2, 15, 14, 30, 0, 0, time.UTC),
		Tags:      "#webapp #react", TotalInputTokens: 300000, TotalOutputTokens: 15000,
	}
	if err := SaveSession(database, updated); err != nil {
		t.Fatalf("SaveSession upsert: %v", err)
	}

	s, _ := GetSession(database, "s2")
	if s.Subject != "React frontend v2" {
		t.Errorf("Subject after upsert = %q", s.Subject)
	}
	if s.TotalInputTokens != 300000 {
		t.Errorf("Tokens after upsert = %d", s.TotalInputTokens)
	}

	// Count should still be 3.
	count, _ := CountSessions(database)
	if count != 3 {
		t.Errorf("count after upsert = %d, want 3", count)
	}
}

func TestSessionDelete(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	if err := DeleteSession(database, "s2"); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	count, _ := CountSessions(database)
	if count != 2 {
		t.Errorf("count after delete = %d, want 2", count)
	}
}

func TestSessionDeleteMultiple(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	if err := DeleteSessions(database, []string{"s1", "s3"}); err != nil {
		t.Fatalf("DeleteSessions: %v", err)
	}
	count, _ := CountSessions(database)
	if count != 1 {
		t.Errorf("count after batch delete = %d, want 1", count)
	}
}

func TestSessionSearch(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	// Search by subject.
	results, err := SearchSessions(database, "docker")
	if err != nil {
		t.Fatalf("SearchSessions: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("search 'docker' returned %d, want 2", len(results))
	}

	// Search by tag.
	results, _ = SearchSessions(database, "#webapp")
	if len(results) != 1 {
		t.Errorf("search '#webapp' returned %d, want 1", len(results))
	}

	// Search by directory.
	results, _ = SearchSessions(database, "project-a")
	if len(results) != 2 {
		t.Errorf("search 'project-a' returned %d, want 2", len(results))
	}

	// Case insensitive.
	results, _ = SearchSessions(database, "REACT")
	if len(results) != 1 {
		t.Errorf("search 'REACT' (case insensitive) returned %d, want 1", len(results))
	}

	// No match.
	results, _ = SearchSessions(database, "nonexistent")
	if len(results) != 0 {
		t.Errorf("search 'nonexistent' returned %d, want 0", len(results))
	}
}

func TestSessionGetOld(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	// All 3 sessions are from 2026, so with days=0 from now they're all old.
	// Use a very large number to get none.
	old, _ := GetOldSessions(database, 99999)
	if len(old) != 0 {
		t.Errorf("GetOldSessions(99999) returned %d, want 0", len(old))
	}
}

func TestSessionMoveToEnd(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()
	seedSessions(t, database)

	// s1 is the oldest. Move it to end.
	if err := MoveToEnd(database, "s1"); err != nil {
		t.Fatalf("MoveToEnd: %v", err)
	}

	sessions, _ := ListSessions(database, 0)
	if sessions[0].SID != "s1" {
		t.Errorf("after MoveToEnd, first session = %q, want s1", sessions[0].SID)
	}
}

// --- Activity Log Tests ---

func TestActivityLog(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()

	if err := LogAction(database, "NEW", "test-session (/tmp/test)"); err != nil {
		t.Fatalf("LogAction NEW: %v", err)
	}
	if err := LogAction(database, "RESUME", "test-session"); err != nil {
		t.Fatalf("LogAction RESUME: %v", err)
	}
	if err := LogAction(database, "DELETE", "test-session"); err != nil {
		t.Fatalf("LogAction DELETE: %v", err)
	}

	entries, err := ListLog(database)
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("log entries = %d, want 3", len(entries))
	}
	// Newest first.
	if entries[0].Action != "DELETE" {
		t.Errorf("newest action = %q, want DELETE", entries[0].Action)
	}
	if entries[2].Action != "NEW" {
		t.Errorf("oldest action = %q, want NEW", entries[2].Action)
	}
}

func TestActivityLogRotation(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()

	// Insert an old entry directly.
	old := time.Now().AddDate(0, 0, -100).Format("2006-01-02 15:04:05")
	database.Exec("INSERT INTO activity_log (timestamp, action, message) VALUES (?, 'OLD', 'old entry')", old)
	// Insert a recent entry.
	LogAction(database, "NEW", "recent entry")

	removed, err := RotateLog(database, 90)
	if err != nil {
		t.Fatalf("RotateLog: %v", err)
	}
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}

	entries, _ := ListLog(database)
	if len(entries) != 1 {
		t.Errorf("after rotation: %d entries, want 1", len(entries))
	}
}

func TestActivityLogRotationDisabled(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()

	LogAction(database, "NEW", "entry")
	removed, _ := RotateLog(database, 0)
	if removed != 0 {
		t.Errorf("rotation disabled but removed %d", removed)
	}
}

// --- Lifetime Tokens Tests ---

func TestLifetimeTokens(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()

	// Initial state.
	in, out, err := GetLifetimeTokens(database)
	if err != nil {
		t.Fatalf("GetLifetimeTokens: %v", err)
	}
	if in != 0 || out != 0 {
		t.Errorf("initial lifetime = (%d, %d), want (0, 0)", in, out)
	}

	// Add tokens.
	if err := AddLifetimeTokens(database, 100000, 5000); err != nil {
		t.Fatalf("AddLifetimeTokens: %v", err)
	}
	if err := AddLifetimeTokens(database, 200000, 10000); err != nil {
		t.Fatalf("AddLifetimeTokens second: %v", err)
	}

	in, out, _ = GetLifetimeTokens(database)
	if in != 300000 {
		t.Errorf("lifetime input = %d, want 300000", in)
	}
	if out != 15000 {
		t.Errorf("lifetime output = %d, want 15000", out)
	}
}
