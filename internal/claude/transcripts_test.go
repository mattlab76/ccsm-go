package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withFakeHome redirects $HOME to a temp dir so the test stages its
// own ~/.claude/projects layout without touching the user's real one.
func withFakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// writeTranscript creates ~/.claude/projects/<slug(cwd)>/<sid>.jsonl
// with one minimal message that has the recorded cwd field.
func writeTranscript(t *testing.T, home, cwd, sid, content string) string {
	t.Helper()
	slug := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(home, ".claude", "projects", slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, sid+".jsonl")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	return path
}

// ─── ResolveSessionCWD ──────────────────────────────────────────────

func TestResolveSessionCWD_PassesThroughWhenCorrect(t *testing.T) {
	home := withFakeHome(t)
	sid := "11111111-1111-1111-1111-111111111111"
	cwd := "/work/repo"
	writeTranscript(t, home, cwd, sid, `{"cwd":"/work/repo","type":"user"}`+"\n")

	got := ResolveSessionCWD(sid, cwd)
	if got != cwd {
		t.Errorf("expected unchanged cwd, got %q", got)
	}
}

func TestResolveSessionCWD_FindsRealDirWhenStoredIsWrong(t *testing.T) {
	home := withFakeHome(t)
	sid := "22222222-2222-2222-2222-222222222222"
	realCWD := "/real/project/path"
	storedCWD := "/wrong/place"
	// Transcript is stored under realCWD's slug, with cwd recorded inside.
	writeTranscript(t, home, realCWD, sid,
		`{"cwd":"/real/project/path","type":"user","message":{}}`+"\n")

	got := ResolveSessionCWD(sid, storedCWD)
	if got != realCWD {
		t.Errorf("expected resolved cwd %q, got %q", realCWD, got)
	}
}

func TestResolveSessionCWD_KeepsInputWhenSIDMissing(t *testing.T) {
	withFakeHome(t)
	got := ResolveSessionCWD("ghost-sid", "/some/cwd")
	if got != "/some/cwd" {
		t.Errorf("missing SID should return input unchanged, got %q", got)
	}
}

func TestResolveSessionCWD_KeepsInputWhenTranscriptHasNoCWD(t *testing.T) {
	home := withFakeHome(t)
	sid := "33333333-3333-3333-3333-333333333333"
	// Transcript exists at the WRONG location and has no cwd field anywhere.
	writeTranscript(t, home, "/some/dir", sid, `{"type":"permission-mode"}`+"\n")

	got := ResolveSessionCWD(sid, "/expected/dir")
	// No cwd recoverable → return the input unchanged.
	if got != "/expected/dir" {
		t.Errorf("expected input unchanged when no cwd in transcript, got %q", got)
	}
}

func TestResolveSessionCWD_EmptyInputStillSearches(t *testing.T) {
	home := withFakeHome(t)
	sid := "44444444-4444-4444-4444-444444444444"
	realCWD := "/somewhere"
	writeTranscript(t, home, realCWD, sid,
		`{"cwd":"/somewhere","type":"user"}`+"\n")

	got := ResolveSessionCWD(sid, "")
	if got != realCWD {
		t.Errorf("expected to find real cwd even with empty input, got %q", got)
	}
}

// ─── RecoverFromTranscript ─────────────────────────────────────────

func TestRecoverFromTranscript_ReturnsNilWhenProjectDirMissing(t *testing.T) {
	withFakeHome(t)
	got := RecoverFromTranscript("/does/not/exist")
	if got != nil {
		t.Errorf("expected nil for missing project dir, got %+v", got)
	}
}

func TestRecoverFromTranscript_PicksNewestJSONL(t *testing.T) {
	home := withFakeHome(t)
	cwd := "/work/myproj"

	// Two transcripts. The newer one should win regardless of file
	// system listing order.
	older := writeTranscript(t, home, cwd, "11111111-1111-1111-1111-111111111111",
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"old prompt"}]}}`+"\n")
	newer := writeTranscript(t, home, cwd, "22222222-2222-2222-2222-222222222222",
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"new prompt"}]}}`+"\n"+
			`{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50}}}`+"\n")
	// Force "newer" to be more recent than "older" via mtime.
	past := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(older, past, past); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	now := time.Now()
	if err := os.Chtimes(newer, now, now); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	rec := RecoverFromTranscript(cwd)
	if rec == nil {
		t.Fatal("expected non-nil recovery")
	}
	if rec.SID != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("expected newest SID, got %q", rec.SID)
	}
	if rec.CWD != cwd {
		t.Errorf("CWD = %q, want %q", rec.CWD, cwd)
	}
	if rec.Subject != "new prompt" {
		t.Errorf("Subject = %q, want %q", rec.Subject, "new prompt")
	}
	if rec.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", rec.OutputTokens)
	}
}

func TestRecoverFromTranscript_SumsAllTokenKinds(t *testing.T) {
	home := withFakeHome(t)
	cwd := "/work/repo"
	// One assistant message with all four kinds of input tokens —
	// the recovered InputTokens should be the sum of input +
	// cache_read + cache_creation.
	writeTranscript(t, home, cwd, "55555555-5555-5555-5555-555555555555",
		`{"type":"assistant","message":{"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":100,"cache_creation_input_tokens":40}}}`+"\n")

	rec := RecoverFromTranscript(cwd)
	if rec == nil {
		t.Fatal("expected non-nil recovery")
	}
	if rec.InputTokens != 150 {
		t.Errorf("InputTokens = %d, want 150 (10+100+40)", rec.InputTokens)
	}
	if rec.OutputTokens != 20 {
		t.Errorf("OutputTokens = %d, want 20", rec.OutputTokens)
	}
}

func TestRecoverFromTranscript_HandlesMissingSubjectGracefully(t *testing.T) {
	home := withFakeHome(t)
	cwd := "/work/repo"
	// Only an assistant message — no user text to extract.
	writeTranscript(t, home, cwd, "66666666-6666-6666-6666-666666666666",
		`{"type":"assistant","message":{"usage":{"input_tokens":1,"output_tokens":2}}}`+"\n")

	rec := RecoverFromTranscript(cwd)
	if rec == nil {
		t.Fatal("expected non-nil recovery")
	}
	if rec.Subject != "" {
		t.Errorf("Subject should be empty when no user text, got %q", rec.Subject)
	}
}

func TestRecoverFromTranscript_TruncatesLongSubject(t *testing.T) {
	home := withFakeHome(t)
	cwd := "/work/repo"
	longText := strings.Repeat("a", 200)
	writeTranscript(t, home, cwd, "77777777-7777-7777-7777-777777777777",
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"`+longText+`"}]}}`+"\n")

	rec := RecoverFromTranscript(cwd)
	if rec == nil {
		t.Fatal("expected non-nil recovery")
	}
	if got := len([]rune(rec.Subject)); got > 120 {
		t.Errorf("Subject should be truncated to <= 120 runes, got %d", got)
	}
}

func TestRecoverFromTranscript_NormalisesWhitespaceInSubject(t *testing.T) {
	home := withFakeHome(t)
	cwd := "/work/repo"
	writeTranscript(t, home, cwd, "88888888-8888-8888-8888-888888888888",
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"hello\nworld\twith tabs"}]}}`+"\n")

	rec := RecoverFromTranscript(cwd)
	if rec == nil {
		t.Fatal("expected non-nil recovery")
	}
	if strings.ContainsAny(rec.Subject, "\n\t") {
		t.Errorf("Subject should not contain newlines or tabs, got %q", rec.Subject)
	}
}
