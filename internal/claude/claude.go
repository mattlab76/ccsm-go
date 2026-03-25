package claude

import (
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// SessionFinishedMsg is sent when claude exits.
type SessionFinishedMsg struct {
	SessionID string
	Err       error
}

// StartSession returns a tea.Cmd that launches claude in the given directory.
// Creates a lock file before starting and removes it when claude exits.
func StartSession(dir string) tea.Cmd {
	c := exec.Command("claude")
	c.Dir = dir
	return tea.ExecProcess(c, func(err error) tea.Msg {
		RemoveSessionLock(dir)
		return SessionFinishedMsg{Err: err}
	})
}

// ResumeSession returns a tea.Cmd that launches claude --resume in the given directory.
// Creates a lock file before starting and removes it when claude exits.
func ResumeSession(sid, dir string) tea.Cmd {
	c := exec.Command("claude", "--resume", sid)
	c.Dir = dir
	return tea.ExecProcess(c, func(err error) tea.Msg {
		RemoveSessionLock(dir)
		return SessionFinishedMsg{SessionID: sid, Err: err}
	})
}

// IsSessionValid checks if a session's JSONL file exists in ~/.claude/projects/.
func IsSessionValid(sid string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	pattern := filepath.Join(home, ".claude", "projects", "*", sid+".jsonl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	return len(matches) > 0
}

// IsInstalled checks if the claude command is available.
func IsInstalled() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// ValidateSessions checks a list of sessions for expired (JSONL missing) and missing dir.
type ValidationResult struct {
	Expired    []string // SIDs where JSONL is gone
	MissingDir []string // SIDs where CWD doesn't exist
}

// ValidateSessions checks sessions for validity.
func ValidateSessions(sessions []struct{ SID, CWD string }) ValidationResult {
	var result ValidationResult
	for _, s := range sessions {
		if !IsSessionValid(s.SID) {
			result.Expired = append(result.Expired, s.SID)
		} else if _, err := os.Stat(s.CWD); os.IsNotExist(err) {
			result.MissingDir = append(result.MissingDir, s.SID)
		}
	}
	return result
}
