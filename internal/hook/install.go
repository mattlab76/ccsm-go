package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// hookTimeout is the seconds Claude Code waits for the SessionEnd hook.
const hookTimeout = 5

// SettingsPath returns the path to Claude Code's settings.json.
func SettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// desiredCommand is the command string we register: "<ccsm> hook", quoted
// if the binary path contains whitespace so the shell parses it as one arg.
func desiredCommand(execPath string) string {
	if strings.ContainsAny(execPath, " \t") {
		return `"` + execPath + `" hook`
	}
	return execPath + " hook"
}

// isCCSMHookCommand reports whether a registered hook command belongs to
// ccsm — covers both the new "<ccsm> hook" form and any older bash-script
// registration (".../ccsm_session_end.sh"), so installing migrates it.
func isCCSMHookCommand(cmd string) bool {
	return strings.Contains(cmd, "ccsm")
}

// InstallSessionEndHook registers (or updates) ccsm as Claude Code's
// SessionEnd hook in settings.json, pointing at execPath. It is idempotent:
//
//   - no ccsm hook yet        → appends a new matcher group ("added")
//   - an old/other ccsm hook  → rewrites its command in place ("updated")
//   - already exactly current → leaves the file untouched ("unchanged")
//
// Every other key in settings.json (effortLevel, unrelated hooks, …) is
// preserved: the file is decoded into a generic map, mutated minimally, and
// re-encoded.
func InstallSessionEndHook(execPath string) (action, path string, err error) {
	path, err = SettingsPath()
	if err != nil {
		return "", "", err
	}

	root := map[string]any{}
	if data, rerr := os.ReadFile(path); rerr == nil {
		if len(data) > 0 {
			if jerr := json.Unmarshal(data, &root); jerr != nil {
				return "", path, fmt.Errorf("%s is not valid JSON: %w", path, jerr)
			}
		}
	} else if !os.IsNotExist(rerr) {
		return "", path, rerr
	}

	cmd := desiredCommand(execPath)

	hooks, _ := root["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
		root["hooks"] = hooks
	}
	sessionEnd, _ := hooks["SessionEnd"].([]any)

	// Update any existing ccsm command in place.
	updated, alreadyCurrent := false, false
	for _, grp := range sessionEnd {
		g, ok := grp.(map[string]any)
		if !ok {
			continue
		}
		inner, ok := g["hooks"].([]any)
		if !ok {
			continue
		}
		for _, h := range inner {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			c, _ := hm["command"].(string)
			if !isCCSMHookCommand(c) {
				continue
			}
			if c == cmd {
				alreadyCurrent = true
				continue
			}
			hm["type"] = "command"
			hm["command"] = cmd
			hm["timeout"] = hookTimeout
			updated = true
		}
	}

	switch {
	case updated:
		action = "updated"
	case alreadyCurrent:
		return "unchanged", path, nil
	default:
		sessionEnd = append(sessionEnd, map[string]any{
			"matcher": "",
			"hooks": []any{
				map[string]any{"type": "command", "command": cmd, "timeout": hookTimeout},
			},
		})
		hooks["SessionEnd"] = sessionEnd
		action = "added"
	}

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", path, err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", path, err
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return "", path, err
	}
	return action, path, nil
}

// HookInstalled reports whether a ccsm SessionEnd hook is registered, and
// returns its command string. Used by `ccsm doctor`. A missing settings.json
// is not an error — it just means "not installed".
func HookInstalled() (installed bool, command string, err error) {
	path, err := SettingsPath()
	if err != nil {
		return false, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", err
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return false, "", fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	hooks, _ := root["hooks"].(map[string]any)
	sessionEnd, _ := hooks["SessionEnd"].([]any)
	for _, grp := range sessionEnd {
		g, _ := grp.(map[string]any)
		inner, _ := g["hooks"].([]any)
		for _, h := range inner {
			hm, _ := h.(map[string]any)
			if c, _ := hm["command"].(string); isCCSMHookCommand(c) {
				return true, c, nil
			}
		}
	}
	return false, "", nil
}
