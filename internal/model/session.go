package model

import "time"

const Version = "2.1.0"

// Session represents a Claude Code session.
type Session struct {
	SID               string
	CWD               string
	Subject           string
	CreatedAt         time.Time
	Tags              string
	TotalInputTokens  int64
	TotalOutputTokens int64
	LastInputTokens   int64
	LastOutputTokens  int64
}

// LogEntry represents an activity log entry.
type LogEntry struct {
	ID        int64
	Timestamp time.Time
	Action    string // NEW, RESUME, SAVE, DELETE, CLEANUP, SETTINGS
	Message   string
}

// Log action constants.
const (
	ActionNew      = "NEW"
	ActionResume   = "RESUME"
	ActionSave     = "SAVE"
	ActionDelete   = "DELETE"
	ActionCleanup  = "CLEANUP"
	ActionSettings = "SETTINGS"
)

// Settings holds the application configuration.
type Settings struct {
	CleanupDays int
	LogDays     int
	Lang        string
}

// DefaultSettings returns settings with default values.
func DefaultSettings() Settings {
	return Settings{
		CleanupDays: 30,
		LogDays:     90,
		Lang:        "en",
	}
}
