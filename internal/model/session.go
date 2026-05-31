package model

import "time"

const Version = "2.3.1"

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
	CleanupDays      int
	LogDays          int
	Lang             string
	Currency         string  // ISO 4217 code, e.g. "EUR". Empty/USD = no conversion.
	ExchangeRate     float64 // USD → Currency rate, e.g. 0.92 for USD→EUR
	PlanName         string  // free text, e.g. "Max 5x" / "Pro" / "API"
	PlanMonthlyPrice float64 // monthly price in the target Currency
}

// DefaultSettings returns settings with default values.
func DefaultSettings() Settings {
	return Settings{
		CleanupDays:  30,
		LogDays:      90,
		Lang:         "en",
		Currency:     "USD",
		ExchangeRate: 1.0,
	}
}
