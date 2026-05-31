package newsession

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mattlab76/ccsm-go/internal/claude"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/hook"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
)

// setup gives us a clean in-memory DB, fresh $HOME, and an isolated
// /tmp/ccsm via hook.SetTmpDirForTest. Locale forced to English.
func setup(t *testing.T) (*db.DB, string) {
	t.Helper()
	prev := i18n.Lang()
	i18n.SetLang("en")
	t.Cleanup(func() { i18n.SetLang(prev) })

	home := t.TempDir()
	t.Setenv("HOME", home)

	hookTmp := t.TempDir()
	hook.SetTmpDirForTest(hookTmp)
	// hook.SetTmpDirForTest is added below in hook helper; for cleanup
	// it'll restore on test end via t.Cleanup inside the helper itself.

	database, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database, home
}

// writeHookFile drops a JSON spool file under hook.tmpDir() so
// hook.FindSessionDataForDir / hook.ReadSessionData can pick it up.
func writeHookFile(t *testing.T, sid, cwd, subject string, in, out int64) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"session_id": sid,
		"cwd":        cwd,
		"betreff":    subject,
		"transcript": "",
		"tokens":     "0/0", // unused — InputTokens/OutputTokens are set below
	})
	// Use hook's internal helper indirectly: spool dir is the tmp the
	// helper set up. We need its path to write the file ourselves.
	path := filepath.Join(hook.TmpDirForTest(), "session-"+sid+".json")
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatal(err)
	}
	// The tokens field is parsed by hook.parseTokens, so encode them as
	// "in/out" if we want them populated.
	if in > 0 || out > 0 {
		body, _ = json.Marshal(map[string]any{
			"session_id": sid,
			"cwd":        cwd,
			"betreff":    subject,
			"transcript": "",
			"tokens":     itoa(in) + "/" + itoa(out),
		})
		_ = os.WriteFile(path, body, 0644)
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func stageTranscript(t *testing.T, home, cwd, sid string) {
	t.Helper()
	slug := strings.ReplaceAll(cwd, "/", "-")
	dir := filepath.Join(home, ".claude", "projects", slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	body := `{"type":"user","cwd":"` + cwd + `","message":{"role":"user","content":[{"type":"text","text":"hello"}]}}` + "\n" +
		`{"type":"assistant","message":{"usage":{"input_tokens":5,"output_tokens":10}}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, sid+".jsonl"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// ─────────────────────────────────────────────────────────────────────

func TestNew_DefaultsToCurrentDirAndSubjectStep(t *testing.T) {
	d, _ := setup(t)
	m := New(d)
	if m.step != stepSubject {
		t.Errorf("step = %v, want stepSubject", m.step)
	}
	if m.targetDir == "" {
		t.Errorf("targetDir should default to current working dir")
	}
}

func TestNewAlreadyRunning_LandsOnWarningStep(t *testing.T) {
	d, _ := setup(t)
	m := NewAlreadyRunning(d, "/work/repo")
	if m.step != stepAlreadyRunning {
		t.Errorf("step = %v, want stepAlreadyRunning", m.step)
	}
	if m.targetDir != "/work/repo" {
		t.Errorf("targetDir = %q, want /work/repo", m.targetDir)
	}
}

func TestNewForResume_LoadsExistingSessionMetadata(t *testing.T) {
	d, _ := setup(t)
	s := &model.Session{
		SID:       "resume-sid",
		CWD:       "/work/repo",
		Subject:   "original",
		CreatedAt: time.Now(),
	}
	if err := db.SaveSession(d, s); err != nil {
		t.Fatal(err)
	}
	m := NewForResume(d, "resume-sid")
	if !m.isUpdate {
		t.Error("isUpdate should be true for existing session")
	}
	if m.targetDir != "/work/repo" {
		t.Errorf("targetDir = %q, want /work/repo", m.targetDir)
	}
	if m.existingSession == nil || m.existingSession.SID != "resume-sid" {
		t.Errorf("existingSession not loaded: %+v", m.existingSession)
	}
}

// ─── Hook-driven save flow ──────────────────────────────────────────

func TestSessionFinished_UsesHookData(t *testing.T) {
	d, _ := setup(t)
	writeHookFile(t, "hook-sid", "/work/repo", "from hook", 42, 17)

	m := New(d)
	m.targetDir = "/work/repo"
	m, _ = m.handleSessionFinished(claude.SessionFinishedMsg{})

	if m.hookData == nil {
		t.Fatal("hookData should be populated from spool file")
	}
	if m.hookData.SessionID != "hook-sid" {
		t.Errorf("SessionID = %q, want hook-sid", m.hookData.SessionID)
	}
	if m.recovered {
		t.Error("recovered should be false when hook data is present")
	}
}

func TestSessionFinished_FallsBackToTranscriptRecovery(t *testing.T) {
	d, home := setup(t)
	// NO hook file. But a transcript exists for this targetDir.
	stageTranscript(t, home, "/work/recovered", "recovered-sid")

	m := New(d)
	m.targetDir = "/work/recovered"
	m, _ = m.handleSessionFinished(claude.SessionFinishedMsg{})

	if m.hookData == nil {
		t.Fatal("hookData should be reconstructed from transcript")
	}
	if m.hookData.SessionID != "recovered-sid" {
		t.Errorf("SessionID = %q, want recovered-sid", m.hookData.SessionID)
	}
	if !m.recovered {
		t.Error("recovered flag should be set when falling back to transcript")
	}
}

func TestSessionFinished_NoHookNoTranscriptShowsNoData(t *testing.T) {
	d, _ := setup(t)
	m := New(d)
	m.targetDir = "/nowhere"
	m, _ = m.handleSessionFinished(claude.SessionFinishedMsg{})

	if m.step != stepDone {
		t.Errorf("step = %v, want stepDone", m.step)
	}
	if m.message == "" {
		t.Error("expected a 'no data' message")
	}
}

func TestSessionFinished_DetectsUpdateForExistingSID(t *testing.T) {
	d, _ := setup(t)
	// Existing session in DB with same SID the hook will report.
	existing := &model.Session{
		SID:       "update-sid",
		CWD:       "/work/repo",
		Subject:   "previous subject",
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}
	if err := db.SaveSession(d, existing); err != nil {
		t.Fatal(err)
	}
	writeHookFile(t, "update-sid", "/work/repo", "from hook", 1, 2)

	m := New(d)
	m.targetDir = "/work/repo"
	m, _ = m.handleSessionFinished(claude.SessionFinishedMsg{})

	if !m.isUpdate {
		t.Error("isUpdate should be true when SID already exists in DB")
	}
	if m.step != stepSaveSubjectConfirm {
		t.Errorf("update flow should go to subject-confirm, got %v", m.step)
	}
}

// ─── Fork constructor ───────────────────────────────────────────────

func TestNewForFork_PrefillsSubjectAndTags(t *testing.T) {
	d, _ := setup(t)
	m := NewForFork(d, "/work/repo", "carried subject", "tag1 tag2")
	if m.step != stepLaunching {
		t.Errorf("step = %v, want stepLaunching", m.step)
	}
	if m.targetDir != "/work/repo" {
		t.Errorf("targetDir = %q, want /work/repo", m.targetDir)
	}
	if m.presetSubject != "carried subject" {
		t.Errorf("presetSubject = %q, want %q", m.presetSubject, "carried subject")
	}
	if m.chosenTags != "tag1 tag2" {
		t.Errorf("chosenTags = %q, want %q", m.chosenTags, "tag1 tag2")
	}
	if m.isUpdate {
		t.Error("fork must NOT be flagged as update — new SID will be assigned")
	}
	if m.existingSession != nil {
		t.Error("fork must not pre-load an existingSession — it's a brand-new session")
	}
}

func TestSessionFinished_AutoSubjectFallsBackToDirName(t *testing.T) {
	d, _ := setup(t)
	// Hook returns no subject — model should auto-build one from dir.
	writeHookFile(t, "sid-x", "/work/myproj", "", 1, 2)

	m := New(d)
	m.targetDir = "/work/myproj"
	m, _ = m.handleSessionFinished(claude.SessionFinishedMsg{})

	if !strings.Contains(m.autoSubject, "myproj") {
		t.Errorf("autoSubject should contain dir name, got %q", m.autoSubject)
	}
}

// ─── Token reconciliation (#3) ──────────────────────────────────────

func TestTokenTotals(t *testing.T) {
	cases := []struct {
		name                                   string
		tin, tout, pin, pout                   int64
		wTotalIn, wTotalOut, wLastIn, wLastOut int64
	}{
		{"fresh session: last==total", 300, 400, 0, 0, 300, 400, 300, 400},
		{"resume stores cumulative, last is delta", 150, 260, 100, 200, 150, 260, 50, 60},
		{"shorter transcript clamps last to zero", 80, 90, 100, 200, 80, 90, 0, 0},
		{"prior inflation self-heals on total", 120, 130, 999, 999, 120, 130, 0, 0},
	}
	for _, c := range cases {
		ti, to, li, lo := tokenTotals(c.tin, c.tout, c.pin, c.pout)
		if ti != c.wTotalIn || to != c.wTotalOut || li != c.wLastIn || lo != c.wLastOut {
			t.Errorf("%s: got total %d/%d last %d/%d; want total %d/%d last %d/%d",
				c.name, ti, to, li, lo, c.wTotalIn, c.wTotalOut, c.wLastIn, c.wLastOut)
		}
	}
}

// TestDoSave_ResumeDoesNotDoubleCount drives the full save path on an update.
// The hook reports the WHOLE transcript (prior 100/200 + this run 50/60 =
// 150/260). The stored Total must equal the whole-transcript figure (not be
// added on top of the prior total), Last must be this run's delta, and the
// lifetime counter must grow only by that delta.
func TestDoSave_ResumeDoesNotDoubleCount(t *testing.T) {
	d, _ := setup(t)
	existing := &model.Session{
		SID: "sid", CWD: "/work/repo", Subject: "subj",
		CreatedAt:        time.Now().Add(-time.Hour),
		TotalInputTokens: 100, TotalOutputTokens: 200,
		LastInputTokens: 100, LastOutputTokens: 200,
	}
	if err := db.SaveSession(d, existing); err != nil {
		t.Fatal(err)
	}

	m := New(d)
	m.isUpdate = true
	m.existingSession = existing
	m.chosenSubject = "subj"
	m.hookData = &hook.HookData{
		SessionID: "sid", CWD: "/work/repo",
		InputTokens: 150, OutputTokens: 260,
	}

	m, _ = m.doSave()

	got, err := db.GetSession(d, "sid")
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalInputTokens != 150 || got.TotalOutputTokens != 260 {
		t.Errorf("Total = %d/%d, want 150/260 (no double-count)",
			got.TotalInputTokens, got.TotalOutputTokens)
	}
	if got.LastInputTokens != 50 || got.LastOutputTokens != 60 {
		t.Errorf("Last = %d/%d, want 50/60 (this run's delta)",
			got.LastInputTokens, got.LastOutputTokens)
	}
	in, out, _ := db.GetLifetimeTokens(d)
	if in != 50 || out != 60 {
		t.Errorf("lifetime = %d/%d, want 50/60 (delta only)", in, out)
	}
}

// TestDoSave_FreshSessionStoresWholeTranscript checks the new-session path:
// no prior totals, so Total and Last both equal the whole-transcript figure
// and lifetime grows by the full amount.
func TestDoSave_FreshSessionStoresWholeTranscript(t *testing.T) {
	d, _ := setup(t)
	m := New(d)
	m.isUpdate = false
	m.chosenSubject = "brand new"
	m.hookData = &hook.HookData{
		SessionID: "fresh", CWD: "/work/new",
		InputTokens: 300, OutputTokens: 400,
	}

	m, _ = m.doSave()

	got, err := db.GetSession(d, "fresh")
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalInputTokens != 300 || got.LastInputTokens != 300 {
		t.Errorf("fresh: total in=%d last in=%d, want 300/300",
			got.TotalInputTokens, got.LastInputTokens)
	}
	in, out, _ := db.GetLifetimeTokens(d)
	if in != 300 || out != 400 {
		t.Errorf("lifetime = %d/%d, want 300/400", in, out)
	}
}
