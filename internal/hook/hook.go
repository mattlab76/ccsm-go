package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const tmpDir = "/tmp/ccsm"

// HookData holds the session metadata written by the hook script.
type HookData struct {
	SessionID    string `json:"session_id"`
	CWD          string `json:"cwd"`
	Subject      string `json:"betreff"`
	Transcript   string `json:"transcript"`
	Tokens       string `json:"tokens"` // "input/output"
	InputTokens  int64
	OutputTokens int64
}

// ReadSessionData reads the hook temp file for a given session ID.
// It waits up to 5 seconds for the file to appear (hook runs async).
func ReadSessionData(sid string) (*HookData, error) {
	path := filepath.Join(tmpDir, fmt.Sprintf("session-%s.json", sid))

	// Wait up to 5 seconds for the file.
	var data []byte
	var err error
	for i := range 10 {
		data, err = os.ReadFile(path)
		if err == nil {
			break
		}
		if i < 9 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("hook data not found after 5s: %w", err)
	}

	var h HookData
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("invalid hook data: %w", err)
	}

	// Parse tokens string "input/output".
	h.InputTokens, h.OutputTokens = parseTokens(h.Tokens)

	return &h, nil
}

// FindLatestSessionData finds the most recently modified temp file.
func FindLatestSessionData() (*HookData, error) {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, err
	}

	var newest os.DirEntry
	var newestTime time.Time
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "session-") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if newest == nil || info.ModTime().After(newestTime) {
			newest = e
			newestTime = info.ModTime()
		}
	}
	if newest == nil {
		return nil, fmt.Errorf("no session data files found")
	}

	// Extract SID from filename: session-{sid}.json
	name := newest.Name()
	sid := strings.TrimPrefix(strings.TrimSuffix(name, ".json"), "session-")
	return ReadSessionData(sid)
}

// CleanupTempFile removes the temp file for a given session.
func CleanupTempFile(sid string) {
	path := filepath.Join(tmpDir, fmt.Sprintf("session-%s.json", sid))
	os.Remove(path)
}

// CleanupOldTempFiles removes temp files older than 1 day.
func CleanupOldTempFiles() {
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "session-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(tmpDir, e.Name()))
		}
	}
}

func parseTokens(s string) (int64, int64) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	in, _ := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	out, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	return in, out
}
