package model

import "testing"

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.CleanupDays != 30 {
		t.Errorf("CleanupDays = %d, want 30", s.CleanupDays)
	}
	if s.LogDays != 90 {
		t.Errorf("LogDays = %d, want 90", s.LogDays)
	}
	if s.Lang != "en" {
		t.Errorf("Lang = %q, want %q", s.Lang, "en")
	}
	if s.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", s.Currency, "USD")
	}
	if s.ExchangeRate != 1.0 {
		t.Errorf("ExchangeRate = %v, want 1.0", s.ExchangeRate)
	}
	if s.PlanName != "" {
		t.Errorf("PlanName should be empty by default, got %q", s.PlanName)
	}
	if s.PlanMonthlyPrice != 0 {
		t.Errorf("PlanMonthlyPrice should be 0 by default, got %v", s.PlanMonthlyPrice)
	}
}

func TestVersionConstant(t *testing.T) {
	if Version == "" {
		t.Error("Version constant is empty")
	}
	// Crude SemVer sanity check.
	if Version[0] < '0' || Version[0] > '9' {
		t.Errorf("Version should start with a digit, got %q", Version)
	}
}

func TestLogActionConstants(t *testing.T) {
	// All action codes must be non-empty and uppercase so the activity
	// log stays consistent across views.
	actions := map[string]string{
		"NEW":      ActionNew,
		"RESUME":   ActionResume,
		"SAVE":     ActionSave,
		"DELETE":   ActionDelete,
		"CLEANUP":  ActionCleanup,
		"SETTINGS": ActionSettings,
	}
	for want, got := range actions {
		if got != want {
			t.Errorf("Action constant mismatch: got %q, want %q", got, want)
		}
	}
}
