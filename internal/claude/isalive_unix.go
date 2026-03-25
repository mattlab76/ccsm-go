//go:build linux || darwin || freebsd

package claude

import "syscall"

// isProcessAlive checks if a process with the given PID is still running.
// Sending signal 0 checks existence without actually sending a signal.
func isProcessAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}
