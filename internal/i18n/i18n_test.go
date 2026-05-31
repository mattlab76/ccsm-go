package i18n

import (
	"testing"
)

func TestT_English(t *testing.T) {
	SetLang("en")
	if got := T("title"); got != "Claude Code Session Manager" {
		t.Errorf("T(title) = %q, want %q", got, "Claude Code Session Manager")
	}
	if got := T("menu_new"); got != "Start new session" {
		t.Errorf("T(menu_new) = %q, want %q", got, "Start new session")
	}
}

func TestT_German(t *testing.T) {
	SetLang("de")
	defer SetLang("en")
	if got := T("menu_new"); got != "Neue Session starten" {
		t.Errorf("T(menu_new) = %q, want %q", got, "Neue Session starten")
	}
	if got := T("menu_quit"); got != "Beenden" {
		t.Errorf("T(menu_quit) = %q, want %q", got, "Beenden")
	}
}

func TestT_FallbackToEnglish(t *testing.T) {
	SetLang("de")
	defer SetLang("en")
	// If a key is missing in DE, it should fall back to EN.
	// All keys are present in both, so test unknown key returns key itself.
	if got := T("nonexistent_key"); got != "nonexistent_key" {
		t.Errorf("T(nonexistent_key) = %q, want %q", got, "nonexistent_key")
	}
}

func TestT_WithArgs(t *testing.T) {
	SetLang("en")
	got := T("cleanup_title", 30)
	want := "Old Sessions (>30 days)"
	if got != want {
		t.Errorf("T(cleanup_title, 30) = %q, want %q", got, want)
	}
}

func TestSetLang_Invalid(t *testing.T) {
	SetLang("fr")
	if got := Lang(); got != "en" {
		t.Errorf("SetLang(fr) => Lang() = %q, want %q", got, "en")
	}
}

func TestIsYes(t *testing.T) {
	SetLang("en")
	if !IsYes("y") {
		t.Error("IsYes(y) should be true for EN")
	}
	if !IsYes("Y") {
		t.Error("IsYes(Y) should be true for EN")
	}
	if !IsYes("yes") {
		t.Error("IsYes(yes) should be true for EN")
	}
	if IsYes("j") {
		t.Error("IsYes(j) should be false for EN")
	}

	SetLang("de")
	defer SetLang("en")
	if !IsYes("j") {
		t.Error("IsYes(j) should be true for DE")
	}
	if !IsYes("ja") {
		t.Error("IsYes(ja) should be true for DE")
	}
	if IsYes("y") {
		t.Error("IsYes(y) should be false for DE")
	}
}

func TestConfirmYes(t *testing.T) {
	SetLang("en")
	if got := ConfirmYes(); got != "y" {
		t.Errorf("ConfirmYes() = %q for EN, want %q", got, "y")
	}
	SetLang("de")
	defer SetLang("en")
	if got := ConfirmYes(); got != "j" {
		t.Errorf("ConfirmYes() = %q for DE, want %q", got, "j")
	}
}

func TestDetectLang(t *testing.T) {
	// DetectLang reads env vars, so we just check it returns a valid value.
	lang := DetectLang()
	if lang != "en" && lang != "de" {
		t.Errorf("DetectLang() = %q, want en or de", lang)
	}
}

func TestStatsCLIKeysDefined(t *testing.T) {
	// Emitted by `ccsm stats` (cmd/ccsm/main.go). A missing definition makes
	// T() return the raw key, so `ccsm stats` would print "stats_input:"
	// instead of a label — the regression this guards against.
	SetLang("en")
	for _, key := range []string{"stats_active", "stats_lifetime", "stats_input", "stats_output"} {
		if T(key) == key {
			t.Errorf("key %q is undefined — `ccsm stats` would render the raw key", key)
		}
	}
}

func TestAllKeysConsistent(t *testing.T) {
	// Every EN key should exist in DE.
	for key := range langEN {
		if _, ok := langDE[key]; !ok {
			t.Errorf("key %q exists in EN but not in DE", key)
		}
	}
	// Every DE key should exist in EN.
	for key := range langDE {
		if _, ok := langEN[key]; !ok {
			t.Errorf("key %q exists in DE but not in EN", key)
		}
	}
}
