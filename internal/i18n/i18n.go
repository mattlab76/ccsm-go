package i18n

import (
	"fmt"
	"os"
	"strings"
)

var currentLang = "en"

// SetLang sets the active language ("en" or "de").
func SetLang(lang string) {
	switch lang {
	case "de", "en":
		currentLang = lang
	default:
		currentLang = "en"
	}
}

// Lang returns the current language code.
func Lang() string {
	return currentLang
}

// DetectLang detects the language from environment variables.
func DetectLang() string {
	for _, key := range []string{"LC_ALL", "LANG", "LC_MESSAGES"} {
		val := os.Getenv(key)
		if strings.HasPrefix(strings.ToLower(val), "de") {
			return "de"
		}
	}
	return "en"
}

// T returns the translated string for the given key.
// Supports fmt-style arguments for %d, %s etc.
func T(key string, args ...any) string {
	var m map[string]string
	if currentLang == "de" {
		m = langDE
	} else {
		m = langEN
	}
	s, ok := m[key]
	if !ok {
		// Fallback to English.
		s, ok = langEN[key]
		if !ok {
			return key
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// IsYes checks if the input is a yes confirmation (y/Y for EN, j/J for DE).
func IsYes(input string) bool {
	input = strings.TrimSpace(strings.ToLower(input))
	if currentLang == "de" {
		return input == "j" || input == "ja"
	}
	return input == "y" || input == "yes"
}

// ConfirmYes returns the localized yes character ("y" or "j").
func ConfirmYes() string {
	if currentLang == "de" {
		return "j"
	}
	return "y"
}

// ConfirmPrompt returns "y/n" or "j/n" based on language.
func ConfirmPrompt() string {
	return ConfirmYes() + "/n"
}

var langEN = map[string]string{
	// App
	"title": "Claude Code Session Manager",

	// Main menu
	"menu_new":    "Start new session",
	"menu_resume": "Browse / search sessions",
	"menu_delete": "Delete session",
	"menu_stats":  "Statistics",
	"menu_quit":   "Quit",
	"menu_recent": "Recent Sessions (enter 1-5 to resume)",
	"menu_action": "What would you like to do?",

	// New session
	"msg_starting":      "Starting new Claude session...",
	"new_subject_prompt": "Subject for this session (optional, Enter=skip):",
	"new_dir_question":  "Start in current directory?",
	"new_dir_current":   "Current:",
	"new_dir_enter_or_q": "Enter path (q=cancel):",
	"new_dir_not_found": "Directory not found. Create it?",
	"new_dir_created":   "Directory created.",

	// Sessions
	"sessions_none": "No saved sessions.",

	// Save dialog
	"save_title":           "Save Session",
	"save_want":            "Save this session?",
	"save_accept":          "Accept this subject?",
	"save_enter":           "Enter subject",
	"save_no_empty":        "Cannot save without a subject. Try again?",
	"save_tags":            "Tags (e.g. #infra #webapp)",
	"save_saved":           "Session saved",
	"save_updated":         "Session updated",
	"save_not_saved":       "Not saved.",
	"save_no_data":         "No session data received (hook not triggered?).",
	"save_recovered":       "Hook missed — recovered from transcript file.",
	"save_current_subject": "Current subject",

	// Delete
	"delete_title":   "Delete Session",
	"delete_confirm": "Really delete this session?",
	"delete_done":    "Session deleted.",

	// Usage banner above the main menu (figures derived from local JSONL transcripts).
	"usage_banner": "Today: %s · 7d: %s · Top: %s",

	// Quit confirmation modal (triggered by Ctrl+C anywhere in the app).
	"quit_title":   "⚠  Really quit ccsm?",
	"quit_hint":    "Press Ctrl+C again or [y]/Enter to quit · [n]/Esc/q to stay",
	"quit_warning": "Running Claude sessions stay alive — only ccsm closes.",

	// Startup check (invalid sessions)
	"startup_title":                 "Session Health Check",
	"startup_summary":               "%d session(s) flagged for your attention:",
	"startup_breakdown_expired":     "%d expired (transcript deleted at Claude Code)",
	"startup_breakdown_missing_dir": "%d with missing working directory (deleted locally)",
	"startup_breakdown_too_old":     "%d older than your cleanup threshold (still usable)",
	"startup_more":                  "... and %d more",
	"startup_purge":                 "Purge all (delete from ccsm)",
	"startup_dismiss":               "Dismiss all (keep, don't ask again)",
	"startup_skip":                  "Skip for now (ask again next start)",
	"startup_hint":                  "↑/↓ navigate  Enter activate  p/d/s shortcut  Esc skip",

	// Cleanup
	"cleanup_title": "Old Sessions (>%d days)",

	// Search
	"search_title":      "Search Sessions",
	"search_prompt":     "Search term",
	"search_no_results": "No matches found.",

	// Statistics
	"stats_title":         "Statistics",
	"stats_none":          "No sessions available.",
	"stats_overview":      "Overview",
	"stats_consumption":   "Token consumption (from local transcripts)",
	"stats_window_today":  "Today",
	"stats_window_24h":    "Last 24h",
	"stats_window_7d":     "Last 7 days",
	"stats_window_all":    "All time",
	"stats_by_model":      "By model (last 7 days)",
	"stats_compare_title": "API vs subscription",
	"stats_compare_intro": "Pay-per-use is what you'd owe at API rates. Compare to your subscription price.",
	"stats_compare_week":  "Last 7 days API-equivalent",
	"stats_compare_month": "Monthly extrapolation (×4.33)",
	"stats_top_sessions":  "Top sessions (last 7 days, by tokens)",
	"stats_patterns":       "Patterns (last 24h)",
	"stats_patterns_intro": "Hints for reducing your token usage.",
	"stats_long_context":   "Long-context (>150k tokens) — try /compact or /clear",
	"stats_subagent":       "Subagent overhead (Task tool spawns count separately)",
	"stats_top_dirs":      "Top directories (by session count)",
	"stats_top_tags":      "Top tags",
	"stats_all_sessions":  "All sessions",
	"stats_token_legend":  "tok = input + output, cache numbers separate",

	// Settings
	"menu_settings":            "Settings",
	"settings_title":           "Settings",
	"settings_lang":            "Language",
	"settings_current":         "Current",
	"settings_lang_prompt":     "Choose language (en/de):",
	"settings_cleanup":         "Cleanup days",
	"settings_cleanup_prompt":  "Days until session is old (0=disabled):",
	"settings_log_days":        "Log retention (days)",
	"settings_log_days_prompt": "Days to keep log entries (0=disabled):",
	"settings_currency":        "Currency",
	"settings_currency_prompt": "Currency code (USD, EUR, GBP, CHF, ...):",
	"settings_rate":            "USD→Currency rate",
	"settings_rate_prompt":     "Exchange rate (e.g. 0.92 for USD→EUR):",
	"settings_plan_name":       "Plan name",
	"settings_plan_name_prompt": "Plan name (free text, e.g. Pro / Max 5x / API):",
	"settings_plan_price":      "Plan price / month",
	"settings_plan_price_prompt": "Monthly plan price in your currency:",
	"settings_saved":           "Settings saved.",
	"settings_invalid":         "Invalid input, keeping current value.",

	// Activity Log
	"menu_log":   "Activity Log",
	"log_title":  "Activity Log",
	"log_empty":  "No log entries.",

	// Legend
	"legend_expired":     "[!] = session expired at Claude Code",
	"legend_missing_dir": "[?] = working directory deleted",

	// Table headers
	"header_nr":      "#",
	"header_subject": "Subject",
	"header_dir":     "Directory",
	"header_date":    "Date",
	"header_in":      "In",
	"header_out":     "Out",

	// Already running
	"already_running_title":      "⚠ Claude session already active!",
	"already_running_dir":        "Directory",
	"already_running_hint":       "A Claude session is already running in this directory. Please switch to that session or choose a different directory.",
	"already_running_change_dir": "Change directory",

	// General
	"press_q": "Press q to return",
}

var langDE = map[string]string{
	// App
	"title": "Claude Code Session Manager",

	// Main menu
	"menu_new":    "Neue Session starten",
	"menu_resume": "Sessions anzeigen / suchen",
	"menu_delete": "Session löschen",
	"menu_stats":  "Statistiken",
	"menu_quit":   "Beenden",
	"menu_recent": "Letzte Sessions (1-5 eingeben zum Fortsetzen)",
	"menu_action": "Was möchten Sie tun?",

	// New session
	"msg_starting":      "Starte neue Claude Session...",
	"new_subject_prompt": "Betreff für diese Session (optional, Enter=überspringen):",
	"new_dir_question":  "Im aktuellen Verzeichnis starten?",
	"new_dir_current":   "Aktuell:",
	"new_dir_enter_or_q": "Pfad eingeben (q=Abbrechen):",
	"new_dir_not_found": "Verzeichnis nicht gefunden. Anlegen?",
	"new_dir_created":   "Verzeichnis angelegt.",

	// Sessions
	"sessions_none": "Keine gespeicherten Sessions.",

	// Save dialog
	"save_title":           "Session speichern",
	"save_want":            "Session speichern?",
	"save_accept":          "Betreff übernehmen?",
	"save_enter":           "Betreff eingeben",
	"save_no_empty":        "Ohne Betreff kann nicht gespeichert werden. Nochmal?",
	"save_tags":            "Tags (z.B. #infra #webapp)",
	"save_saved":           "Session gespeichert",
	"save_updated":         "Session aktualisiert",
	"save_not_saved":       "Nicht gespeichert.",
	"save_no_data":         "Keine Session-Daten empfangen (Hook nicht ausgelöst?).",
	"save_recovered":       "Hook fehlte — aus Transcript-Datei rekonstruiert.",
	"save_current_subject": "Aktueller Betreff",

	// Delete
	"delete_title":   "Session löschen",
	"delete_confirm": "Diese Session wirklich löschen?",
	"delete_done":    "Session gelöscht.",

	// Usage banner above the main menu (figures derived from local JSONL transcripts).
	"usage_banner": "Heute: %s · 7T: %s · Top: %s",

	// Quit confirmation modal (triggered by Ctrl+C anywhere in the app).
	"quit_title":   "⚠  ccsm wirklich beenden?",
	"quit_hint":    "Nochmal Strg+C oder [y]/Enter zum Beenden · [n]/Esc/q für Abbruch",
	"quit_warning": "Laufende Claude-Sessions bleiben aktiv — nur ccsm schließt.",

	// Startup check (invalid sessions)
	"startup_title":                 "Session-Prüfung",
	"startup_summary":               "%d Session(s) benötigen deine Aufmerksamkeit:",
	"startup_breakdown_expired":     "%d abgelaufen (Transcript bei Claude Code gelöscht)",
	"startup_breakdown_missing_dir": "%d mit fehlendem Arbeitsverzeichnis (lokal gelöscht)",
	"startup_breakdown_too_old":     "%d älter als dein Cleanup-Limit (noch nutzbar)",
	"startup_more":                  "... und %d weitere",
	"startup_purge":                 "Alle löschen (aus ccsm entfernen)",
	"startup_dismiss":               "Alle ignorieren (behalten, nicht mehr fragen)",
	"startup_skip":                  "Vorerst überspringen (beim nächsten Start wieder fragen)",
	"startup_hint":                  "↑/↓ navigieren  Enter ausführen  p/d/s Shortcut  Esc überspringen",

	// Cleanup
	"cleanup_title": "Alte Sessions (>%d Tage)",

	// Search
	"search_title":      "Sessions suchen",
	"search_prompt":     "Suchbegriff",
	"search_no_results": "Keine Treffer.",

	// Statistics
	"stats_title":         "Statistiken",
	"stats_none":          "Keine Sessions vorhanden.",
	"stats_overview":      "Übersicht",
	"stats_consumption":   "Token-Verbrauch (aus lokalen Transcripts)",
	"stats_window_today":  "Heute",
	"stats_window_24h":    "Letzte 24h",
	"stats_window_7d":     "Letzte 7 Tage",
	"stats_window_all":    "Gesamt",
	"stats_by_model":      "Nach Modell (letzte 7 Tage)",
	"stats_compare_title": "API vs Abo",
	"stats_compare_intro": "Pay-per-use ist was du zu API-Preisen schuldig wärst. Vergleich mit deinem Abo.",
	"stats_compare_week":  "Letzte 7 Tage API-Äquivalent",
	"stats_compare_month": "Hochgerechnet auf Monat (×4.33)",
	"stats_top_sessions":  "Top-Sessions (letzte 7 Tage, nach Tokens)",
	"stats_patterns":       "Muster (letzte 24h)",
	"stats_patterns_intro": "Tipps zum Reduzieren deines Token-Verbrauchs.",
	"stats_long_context":   "Lange Kontexte (>150k Tokens) — probier /compact oder /clear",
	"stats_subagent":       "Subagent-Overhead (Task-Tool spawnt eigene Requests)",
	"stats_top_dirs":      "Top-Verzeichnisse (nach Session-Anzahl)",
	"stats_top_tags":      "Top-Tags",
	"stats_all_sessions":  "Alle Sessions",
	"stats_token_legend":  "tok = Input + Output, Cache-Zahlen separat",

	// Settings
	"menu_settings":            "Einstellungen",
	"settings_title":           "Einstellungen",
	"settings_lang":            "Sprache",
	"settings_current":         "Aktuell",
	"settings_lang_prompt":     "Sprache wählen (en/de):",
	"settings_currency":        "Währung",
	"settings_currency_prompt": "Währungscode (USD, EUR, GBP, CHF, ...):",
	"settings_rate":            "USD→Währung Kurs",
	"settings_rate_prompt":     "Wechselkurs (z.B. 0.92 für USD→EUR):",
	"settings_plan_name":       "Abo-Name",
	"settings_plan_name_prompt": "Abo-Name (Freitext, z.B. Pro / Max 5x / API):",
	"settings_plan_price":      "Abo-Preis / Monat",
	"settings_plan_price_prompt": "Monatlicher Abo-Preis in deiner Währung:",
	"settings_cleanup":       "Aufräumen nach Tagen",
	"settings_cleanup_prompt": "Tage bis Session als alt gilt (0=deaktiviert):",
	"settings_log_days":       "Log-Aufbewahrung (Tage)",
	"settings_log_days_prompt": "Tage für Log-Aufbewahrung (0=deaktiviert):",
	"settings_saved":          "Einstellungen gespeichert.",
	"settings_invalid":        "Ungültige Eingabe, aktueller Wert bleibt.",

	// Activity Log
	"menu_log":   "Aktivitätslog",
	"log_title":  "Aktivitätslog",
	"log_empty":  "Keine Log-Einträge.",

	// Legend
	"legend_expired":     "[!] = Session bei Claude Code abgelaufen",
	"legend_missing_dir": "[?] = Arbeitsverzeichnis gelöscht",

	// Table headers
	"header_nr":      "#",
	"header_subject": "Betreff",
	"header_dir":     "Verzeichnis",
	"header_date":    "Datum",
	"header_in":      "In",
	"header_out":     "Out",

	// Already running
	"already_running_title":      "⚠ Claude Session bereits aktiv!",
	"already_running_dir":        "Verzeichnis",
	"already_running_hint":       "In diesem Verzeichnis läuft bereits eine Claude Session. Bitte wechsle zu dieser Session oder wähle ein anderes Verzeichnis.",
	"already_running_change_dir": "Verzeichnis wechseln",

	// General
	"press_q": "q drücken zum Zurückkehren",
}
