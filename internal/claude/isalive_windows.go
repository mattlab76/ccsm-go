//go:build windows

package claude

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

// isProcessAlive checks if a process with the given PID is still running.
// Uses tasklist to query the process by PID.
func isProcessAlive(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH", "/FO", "CSV")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	// tasklist returns "INFO: No tasks are running..." if PID not found.
	return !strings.Contains(string(out), "INFO:")
}
