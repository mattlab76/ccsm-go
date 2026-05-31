package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readSettings decodes the settings.json at the test HOME into a map.
func readSettings(t *testing.T) map[string]any {
	t.Helper()
	path, _ := SettingsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("settings not valid JSON: %v\n%s", err, data)
	}
	return m
}

// sessionEndCommands extracts every SessionEnd hook command string.
func sessionEndCommands(t *testing.T, m map[string]any) []string {
	t.Helper()
	var cmds []string
	hooks, _ := m["hooks"].(map[string]any)
	se, _ := hooks["SessionEnd"].([]any)
	for _, grp := range se {
		g, _ := grp.(map[string]any)
		inner, _ := g["hooks"].([]any)
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			if c, ok := hm["command"].(string); ok {
				cmds = append(cmds, c)
			}
		}
	}
	return cmds
}

func TestInstall_FreshCreatesHook(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	action, path, err := InstallSessionEndHook("/usr/local/bin/ccsm")
	if err != nil {
		t.Fatal(err)
	}
	if action != "added" {
		t.Errorf("action = %q, want added", action)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings.json not written: %v", err)
	}
	cmds := sessionEndCommands(t, readSettings(t))
	if len(cmds) != 1 || cmds[0] != "/usr/local/bin/ccsm hook" {
		t.Errorf("commands = %v, want one '/usr/local/bin/ccsm hook'", cmds)
	}
}

func TestInstall_MigratesBashHookAndPreservesOtherKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing settings: an OLD bash ccsm hook + an unrelated key.
	pre := `{
  "effortLevel": "xhigh",
  "hooks": {
    "SessionEnd": [
      { "matcher": "", "hooks": [
        { "type": "command", "command": "bash /home/x/.claude/hooks/ccsm_session_end.sh", "timeout": 5 }
      ] }
    ]
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(pre), 0644); err != nil {
		t.Fatal(err)
	}

	action, _, err := InstallSessionEndHook("/home/x/.local/bin/ccsm")
	if err != nil {
		t.Fatal(err)
	}
	if action != "updated" {
		t.Errorf("action = %q, want updated", action)
	}

	m := readSettings(t)
	// Unrelated key preserved.
	if m["effortLevel"] != "xhigh" {
		t.Errorf("effortLevel = %v, want xhigh (must be preserved)", m["effortLevel"])
	}
	// Exactly one ccsm command, now the self-hook form (bash replaced, not duplicated).
	cmds := sessionEndCommands(t, m)
	if len(cmds) != 1 || cmds[0] != "/home/x/.local/bin/ccsm hook" {
		t.Errorf("commands = %v, want one '/home/x/.local/bin/ccsm hook'", cmds)
	}
}

func TestInstall_Idempotent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, _, err := InstallSessionEndHook("/bin/ccsm"); err != nil {
		t.Fatal(err)
	}
	action, _, err := InstallSessionEndHook("/bin/ccsm")
	if err != nil {
		t.Fatal(err)
	}
	if action != "unchanged" {
		t.Errorf("second install action = %q, want unchanged", action)
	}
	if cmds := sessionEndCommands(t, readSettings(t)); len(cmds) != 1 {
		t.Errorf("commands = %v, want exactly one (no duplicate)", cmds)
	}
}

func TestInstall_PreservesUnrelatedSessionEndHook(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	pre := `{"hooks":{"SessionEnd":[{"matcher":"","hooks":[
		{"type":"command","command":"/opt/other-tool notify","timeout":3}]}]}}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(pre), 0644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := InstallSessionEndHook("/bin/ccsm"); err != nil {
		t.Fatal(err)
	}
	cmds := sessionEndCommands(t, readSettings(t))
	var hasOther, hasCCSM bool
	for _, c := range cmds {
		if c == "/opt/other-tool notify" {
			hasOther = true
		}
		if c == "/bin/ccsm hook" {
			hasCCSM = true
		}
	}
	if !hasOther {
		t.Error("unrelated SessionEnd hook was dropped")
	}
	if !hasCCSM {
		t.Error("ccsm hook was not added alongside the unrelated one")
	}
}

func TestInstall_QuotesPathWithSpaces(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if _, _, err := InstallSessionEndHook("/Applications/My Tools/ccsm"); err != nil {
		t.Fatal(err)
	}
	cmds := sessionEndCommands(t, readSettings(t))
	if len(cmds) != 1 || cmds[0] != `"/Applications/My Tools/ccsm" hook` {
		t.Errorf("commands = %v, want quoted path", cmds)
	}
}

func TestHookInstalled_DetectsAndAbsent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// No settings.json yet → not installed, no error.
	if ok, _, err := HookInstalled(); err != nil || ok {
		t.Errorf("HookInstalled (no file) = (%v, err=%v), want (false, nil)", ok, err)
	}

	if _, _, err := InstallSessionEndHook("/bin/ccsm"); err != nil {
		t.Fatal(err)
	}
	ok, cmd, err := HookInstalled()
	if err != nil || !ok || cmd != "/bin/ccsm hook" {
		t.Errorf("HookInstalled = (%v, %q, %v), want (true, '/bin/ccsm hook', nil)", ok, cmd, err)
	}
}

func TestHandleSessionEnd_WritesSpoolFile(t *testing.T) {
	dir := withTempHookDir(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Stage a transcript with one user message + assistant usage.
	cwd := "/work/proj"
	slug := "-work-proj"
	pdir := filepath.Join(home, ".claude", "projects", slug)
	if err := os.MkdirAll(pdir, 0755); err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(pdir, "sid123.jsonl")
	body := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"build a thing"}]}}` + "\n" +
		`{"type":"assistant","message":{"usage":{"input_tokens":40,"cache_read_input_tokens":2,"output_tokens":17}}}` + "\n"
	if err := os.WriteFile(transcript, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	payload := `{"session_id":"sid123","cwd":"` + cwd + `","transcript_path":"` + transcript + `"}`
	if err := HandleSessionEnd(strings.NewReader(payload)); err != nil {
		t.Fatalf("HandleSessionEnd: %v", err)
	}

	h, err := ReadSessionData("sid123")
	if err != nil {
		t.Fatalf("ReadSessionData: %v", err)
	}
	if h.CWD != cwd {
		t.Errorf("CWD = %q, want %q", h.CWD, cwd)
	}
	if h.Subject != "build a thing" {
		t.Errorf("Subject = %q, want %q", h.Subject, "build a thing")
	}
	if h.InputTokens != 42 || h.OutputTokens != 17 { // 40 input + 2 cache_read
		t.Errorf("tokens = %d/%d, want 42/17", h.InputTokens, h.OutputTokens)
	}
	_ = dir
}
