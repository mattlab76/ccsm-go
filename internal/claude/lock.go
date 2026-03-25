package claude

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// lockDir returns the directory for lock files (~/.claude/ccsm-locks/).
func lockDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "ccsm-locks")
}

// lockPath returns the lock file path for a given working directory.
// Uses a SHA-256 hash of the absolute path as filename.
func lockPath(dir string) string {
	ld := lockDir()
	if ld == "" {
		return ""
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(absDir)))
	return filepath.Join(ld, hash+".lock")
}

// CreateSessionLock creates a lock file for the given directory containing the PID.
func CreateSessionLock(dir string, pid int) error {
	ld := lockDir()
	if ld == "" {
		return fmt.Errorf("cannot determine lock directory")
	}
	if err := os.MkdirAll(ld, 0755); err != nil {
		return err
	}
	path := lockPath(dir)
	if path == "" {
		return fmt.Errorf("cannot determine lock path")
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

// RemoveSessionLock removes the lock file for the given directory.
func RemoveSessionLock(dir string) {
	path := lockPath(dir)
	if path != "" {
		os.Remove(path)
	}
}

// IsClaudeRunningInDir checks if a Claude session is active in the given directory
// by reading the lock file and verifying the process is still alive.
func IsClaudeRunningInDir(dir string) bool {
	path := lockPath(dir)
	if path == "" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		// Invalid lock file — clean it up.
		os.Remove(path)
		return false
	}
	if !isProcessAlive(pid) {
		// Stale lock — process no longer running, clean up.
		os.Remove(path)
		return false
	}
	return true
}

// CleanupStaleLocks removes all lock files with dead PIDs.
func CleanupStaleLocks() {
	ld := lockDir()
	if ld == "" {
		return
	}
	entries, err := os.ReadDir(ld)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".lock") {
			continue
		}
		path := filepath.Join(ld, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			os.Remove(path)
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil || pid <= 0 || !isProcessAlive(pid) {
			os.Remove(path)
		}
	}
}
