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
	"msg_switching":     "Switching to",
	"msg_resuming":      "Resuming session",

	// Errors
	"err_no_claude": "Error: Claude Code is not installed or not in PATH.",

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
	"save_current_subject": "Current subject",

	// Delete
	"delete_title":   "Delete Session",
	"delete_confirm": "Really delete this session?",
	"delete_done":    "Session deleted.",

	// Cleanup
	"cleanup_title":         "Old Sessions (>%d days)",
	"cleanup_done":          "%d session(s) deleted.",
	"cleanup_select_prompt": "Enter numbers to delete (comma-separated, e.g. 1,3) or [q] to skip:",

	// Search
	"search_title":       "Search Sessions",
	"search_prompt":      "Search term",
	"search_no_results":  "No matches found.",
	"search_no_sessions": "No sessions available.",

	// Statistics
	"stats_title":          "Statistics",
	"stats_none":           "No sessions available.",
	"stats_token_usage":    "Token Usage",
	"stats_active":         "Active Sessions",
	"stats_lifetime":       "Lifetime (all time, incl. deleted)",
	"stats_input":          "Input",
	"stats_output":         "Output",
	"stats_token_info":     "ℹ Token Info:",
	"stats_token_explain1": "Input  = your messages + system prompt + full chat history",
	"stats_token_explain2": "         + tool results (file contents) + cache",
	"stats_token_explain3": "Output = Claude's response text + tool calls",
	"stats_token_explain4": "Input >> Output is normal (Claude reads much more than it writes)",
	"stats_overview":       "Overview",
	"stats_top_dirs":       "Top Directories",
	"stats_top_tags":       "Top Tags",
	"stats_all_sessions":   "All Sessions",

	// Settings
	"menu_settings":          "Settings",
	"settings_title":         "Settings",
	"settings_lang":          "Language",
	"settings_lang_current":  "Current",
	"settings_lang_prompt":   "Choose language (en/de):",
	"settings_cleanup":       "Cleanup days",
	"settings_cleanup_current": "Current",
	"settings_cleanup_prompt": "Days until session is old (0=disabled):",
	"settings_log_days":       "Log retention (days)",
	"settings_log_days_prompt": "Days to keep log entries (0=disabled):",
	"settings_saved":          "Settings saved.",
	"settings_invalid":        "Invalid input, keeping current value.",

	// Activity Log
	"menu_log":   "Activity Log",
	"log_title":  "Activity Log",
	"log_empty":  "No log entries.",

	// Directory issues
	"dir_missing":        "Directory no longer exists",
	"dir_recreate":       "Recreate directory and continue",
	"dir_delete_session": "Delete this session",
	"dir_cancel_back":    "Back to main menu",

	// Session issues
	"session_not_found":  "Session no longer exists at Claude Code.",
	"session_restart_new": "Start new session (same directory and subject)",

	// Startup validation
	"expired_warning":     "%d session(s) no longer valid at Claude Code",
	"expired_purge":       "Remove invalid sessions now?",
	"expired_purged":      "Invalid sessions removed.",
	"missing_dir_warning": "%d session(s) with deleted working directory",

	// Legend
	"legend_expired":     "[!] = session expired at Claude Code",
	"legend_missing_dir": "[?] = working directory deleted",

	// Confirmation
	"confirm_yes": "y",

	// Table headers
	"header_nr":      "#",
	"header_subject": "Subject",
	"header_dir":     "Directory",
	"header_date":    "Date",
	"header_tags":    "Tags",
	"header_tokens":  "Tokens",
	"header_in":      "In",
	"header_out":     "Out",

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
	"msg_switching":     "Wechsle nach",
	"msg_resuming":      "Setze Session fort",

	// Errors
	"err_no_claude": "Fehler: Claude Code ist nicht installiert oder nicht im PATH.",

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
	"save_current_subject": "Aktueller Betreff",

	// Delete
	"delete_title":   "Session löschen",
	"delete_confirm": "Diese Session wirklich löschen?",
	"delete_done":    "Session gelöscht.",

	// Cleanup
	"cleanup_title":         "Alte Sessions (>%d Tage)",
	"cleanup_done":          "%d Session(s) gelöscht.",
	"cleanup_select_prompt": "Nummern zum Löschen eingeben (kommagetrennt, z.B. 1,3) oder [q] zum Überspringen:",

	// Search
	"search_title":       "Sessions suchen",
	"search_prompt":      "Suchbegriff",
	"search_no_results":  "Keine Treffer.",
	"search_no_sessions": "Keine Sessions vorhanden.",

	// Statistics
	"stats_title":          "Statistiken",
	"stats_none":           "Keine Sessions vorhanden.",
	"stats_token_usage":    "Token-Verbrauch",
	"stats_active":         "Aktive Sessions",
	"stats_lifetime":       "Lifetime (gesamt, inkl. gelöschte)",
	"stats_input":          "Input",
	"stats_output":         "Output",
	"stats_token_info":     "ℹ Token-Info:",
	"stats_token_explain1": "Input  = Nachrichten + System-Prompt + Chat-Verlauf",
	"stats_token_explain2": "         + Tool-Ergebnisse (Dateiinhalte) + Cache",
	"stats_token_explain3": "Output = Claudes Antworten + Tool-Aufrufe",
	"stats_token_explain4": "Input >> Output ist normal (Claude liest viel mehr als es schreibt)",
	"stats_overview":       "Übersicht",
	"stats_top_dirs":       "Top Verzeichnisse",
	"stats_top_tags":       "Top Tags",
	"stats_all_sessions":   "Alle Sessions",

	// Settings
	"menu_settings":          "Einstellungen",
	"settings_title":         "Einstellungen",
	"settings_lang":          "Sprache",
	"settings_lang_current":  "Aktuell",
	"settings_lang_prompt":   "Sprache wählen (en/de):",
	"settings_cleanup":       "Aufräumen nach Tagen",
	"settings_cleanup_current": "Aktuell",
	"settings_cleanup_prompt": "Tage bis Session als alt gilt (0=deaktiviert):",
	"settings_log_days":       "Log-Aufbewahrung (Tage)",
	"settings_log_days_prompt": "Tage für Log-Aufbewahrung (0=deaktiviert):",
	"settings_saved":          "Einstellungen gespeichert.",
	"settings_invalid":        "Ungültige Eingabe, aktueller Wert bleibt.",

	// Activity Log
	"menu_log":   "Aktivitätslog",
	"log_title":  "Aktivitätslog",
	"log_empty":  "Keine Log-Einträge.",

	// Directory issues
	"dir_missing":        "Verzeichnis existiert nicht mehr",
	"dir_recreate":       "Verzeichnis neu anlegen und fortfahren",
	"dir_delete_session": "Diese Session löschen",
	"dir_cancel_back":    "Zurück zum Hauptmenü",

	// Session issues
	"session_not_found":  "Session existiert bei Claude Code nicht mehr.",
	"session_restart_new": "Neue Session starten (gleiches Verzeichnis und Betreff)",

	// Startup validation
	"expired_warning":     "%d Session(s) bei Claude Code nicht mehr gültig",
	"expired_purge":       "Ungültige Sessions jetzt entfernen?",
	"expired_purged":      "Ungültige Sessions entfernt.",
	"missing_dir_warning": "%d Session(s) mit gelöschtem Arbeitsverzeichnis",

	// Legend
	"legend_expired":     "[!] = Session bei Claude Code abgelaufen",
	"legend_missing_dir": "[?] = Arbeitsverzeichnis gelöscht",

	// Confirmation
	"confirm_yes": "j",

	// Table headers
	"header_nr":      "#",
	"header_subject": "Betreff",
	"header_dir":     "Verzeichnis",
	"header_date":    "Datum",
	"header_tags":    "Tags",
	"header_tokens":  "Tokens",
	"header_in":      "In",
	"header_out":     "Out",

	// General
	"press_q": "q drücken zum Zurückkehren",
}
