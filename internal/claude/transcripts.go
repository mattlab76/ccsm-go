package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RecoveredSession is reconstructed from a JSONL transcript when the
// SessionEnd hook didn't fire (Claude crashed, user killed with Ctrl+C,
// hook timed out, etc.). All fields are best-effort: Subject defaults
// to empty if no user message could be parsed, token counts default to 0.
type RecoveredSession struct {
	SID            string
	CWD            string
	Subject        string
	InputTokens    int64 // sum of input + cache_read + cache_creation
	OutputTokens   int64
	TranscriptPath string
	EndedAt        time.Time // mtime of the transcript file
}

// RecoverFromTranscript scans Claude's project directory for targetDir
// and returns the most recently modified transcript reconstructed as
// a RecoveredSession. Returns nil if no transcript exists.
//
// Used as the fallback when the SessionEnd hook didn't produce a temp
// file — common after Ctrl+C inside Claude.
func RecoverFromTranscript(targetDir string) *RecoveredSession {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	slug := strings.ReplaceAll(targetDir, "/", "-")
	projectDir := filepath.Join(home, ".claude", "projects", slug)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil
	}
	var newest os.DirEntry
	var newestTime time.Time
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
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
		return nil
	}
	path := filepath.Join(projectDir, newest.Name())
	rec := &RecoveredSession{
		SID:            strings.TrimSuffix(newest.Name(), ".jsonl"),
		CWD:            targetDir,
		TranscriptPath: path,
		EndedAt:        newestTime,
	}
	rec.Subject, rec.InputTokens, rec.OutputTokens = parseTranscript(path)
	return rec
}

// ParseTranscript reads a Claude JSONL transcript and returns the first
// user-message text (auto subject) plus the summed input/output token usage.
// Exposed for the SessionEnd hook entry point (`ccsm hook`), which needs the
// same figures the bash hook produced — without jq/python3.
func ParseTranscript(path string) (subject string, inTokens, outTokens int64) {
	return parseTranscript(path)
}

// parseTranscript walks every line of a JSONL transcript pulling out
// the first user-message text (as the auto-suggested subject) and
// summing token usage from assistant messages. Matches what the bash
// hook script does, so recovered sessions look the same as hooked ones.
func parseTranscript(path string) (subject string, inTokens, outTokens int64) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		var line struct {
			Type    string `json:"type"`
			Message struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
				Usage struct {
					InputTokens              int64 `json:"input_tokens"`
					OutputTokens             int64 `json:"output_tokens"`
					CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
					CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if subject == "" && line.Type == "user" && line.Message.Role == "user" {
			for _, c := range line.Message.Content {
				if c.Type == "text" {
					s := strings.TrimSpace(c.Text)
					s = strings.ReplaceAll(s, "\n", " ")
					s = strings.ReplaceAll(s, "\t", " ")
					if len([]rune(s)) > 120 {
						s = string([]rune(s)[:120])
					}
					if s != "" {
						subject = s
						break
					}
				}
			}
		}
		u := line.Message.Usage
		inTokens += u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
		outTokens += u.OutputTokens
	}
	return
}

// ResolveSessionCWD verifies that a transcript for sid exists at the
// project directory derived from expectedCWD. Two things can be wrong:
//
//   - The hook captured a *cwd* (e.g. after the user `cd`'d inside
//     Claude or used --add-dir) that differs from the *project dir*
//     Claude uses to store its JSONL.
//   - The session was renamed/moved by the user, leaving expectedCWD
//     stale in the ccsm DB.
//
// Strategy: if the expected location holds the transcript, return
// expectedCWD unchanged. Otherwise scan every project directory for
// {sid}.jsonl and read the cwd that Claude itself recorded inside.
// That is the path Claude will look in when --resume runs, so storing
// it in ccsm keeps resume working.
//
// If nothing is found, return expectedCWD untouched — better to keep
// the user's intent than to silently blank a directory.
func ResolveSessionCWD(sid, expectedCWD string) string {
	if expectedCWD != "" && jsonlExistsFor(sid, expectedCWD) {
		return expectedCWD
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return expectedCWD
	}
	projects := filepath.Join(home, ".claude", "projects")
	entries, err := os.ReadDir(projects)
	if err != nil {
		return expectedCWD
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		jsonl := filepath.Join(projects, e.Name(), sid+".jsonl")
		if _, err := os.Stat(jsonl); err != nil {
			continue
		}
		if cwd := cwdFromTranscript(jsonl); cwd != "" {
			return cwd
		}
	}
	return expectedCWD
}

// jsonlExistsFor checks whether the transcript for sid sits at the
// path Claude would derive from cwd.
func jsonlExistsFor(sid, cwd string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	slug := strings.ReplaceAll(cwd, "/", "-")
	path := filepath.Join(home, ".claude", "projects", slug, sid+".jsonl")
	_, err = os.Stat(path)
	return err == nil
}

// cwdFromTranscript reads the first JSONL line that carries a "cwd"
// field and returns its value. Claude tags every assistant/user
// message with the working directory it ran in.
func cwdFromTranscript(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		var r struct {
			CWD string `json:"cwd"`
		}
		if json.Unmarshal(scanner.Bytes(), &r) == nil && r.CWD != "" {
			return r.CWD
		}
	}
	return ""
}
