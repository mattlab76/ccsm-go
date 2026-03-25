# ccsm v2.0 — Go Rewrite

## Kontext

Dies ist ein Rewrite des Bash-Tools [ccsm](https://github.com/mattlab76/ccsm) (Claude Code Session Manager) in Go.
Die Bash-Version (v1.6.0) ist feature-complete und im Produktiveinsatz. Diese Go-Version soll die gleiche Funktionalität bieten, aber robuster, schneller und plattformunabhängiger sein.

## Referenz-Implementierung

**Bash-Version:** `/home/mha/appsDev/ccsm/ccsm` (~2060 Zeilen)
**GitHub:** https://github.com/mattlab76/ccsm
**README mit allen Features:** `/home/mha/appsDev/ccsm/README.md`
**CHANGELOG:** `/home/mha/appsDev/ccsm/CHANGELOG.md`
**Manueller Testplan:** `/home/mha/appsDev/ccsm/test/MANUAL_TESTPLAN.md`
**Session-Log Format:** TSV mit Feldern: `sid\tcwd\tbetreff\tdatum\ttags\ttokens`
**Hook-Script:** `/home/mha/appsDev/ccsm/hooks/session_end.sh`

## Features die nachgebaut werden müssen

Alle Features aus v1.6.0 — siehe `/home/mha/appsDev/ccsm/README.md` für die komplette Liste:

### Kern-Features
- Interaktives TUI-Menü mit Box-Drawing und Farben
- Session speichern (Subject, Tags, Tokens, Datum+Uhrzeit)
- Session fortsetzen (claude --resume)
- Quick Resume (1-5) im Hauptmenü
- Auto-cd ins Arbeitsverzeichnis
- Betreff-Vorschläge aus Transcript
- Token-Tracking (pro Session + Lifetime)
- Suche (nach Subject, Directory, Tag — case-insensitiv)
- Session löschen (einzeln, mehrere kommagetrennt)

### Erweiterte Features
- Settings-Menü (Sprache, Cleanup-Tage, Log-Tage)
- Activity Log mit farbcodierten Aktionen + Rotation
- Session-Validierung (JSONL-Check in ~/.claude/projects/)
- Status-Markierungen: [!] rot = expired, [?] amber = dir missing
- Legende unter Tabellen (nur wenn nötig)
- Startup-Check für ungültige Sessions (mit Purge/Dismiss)
- Dismissed-Liste (nicht nochmal fragen)
- Resume: Verzeichnis fehlt → Neu anlegen / Löschen
- Resume: Session expired → Neue starten mit gleichem Subject+Dir / Löschen
- "No conversation found" Erkennung + Recovery
- Auto-Cleanup alter Sessions
- Statistiken (Token-Balken, Top-Dirs, Top-Tags, Oldest/Newest)

### Infrastruktur
- Zweisprachig (EN/DE) mit automatischer Locale-Erkennung
- Cross-platform (Linux, macOS, FreeBSD, **NEU: Windows**)
- Installer mit Update-Erkennung + Sprachauswahl
- CLI-Flags: --search, --stats, --cleanup, --version, --help
- Zsh-Completion

## Technologie-Entscheidungen

### Go + bubbletea
- **bubbletea** für TUI (vom gleichen Team wie gum/charm)
- **lipgloss** für Styling/Farben
- **SQLite** (via modernc.org/sqlite oder mattn/go-sqlite3) statt TSV
- Single binary, keine externen Dependencies zur Laufzeit

### Datenbank
- SQLite statt TSV für Sessions, Log, Settings
- `ccsm migrate` Kommando zum Import bestehender TSV-Daten
- Schema: sessions, activity_log, settings, dismissed Tabellen

### Hook
- Hook-Script bleibt Bash (wird von Claude Code aufgerufen)
- Alternativ: Go-Binary als Hook (schneller, kein jq/python3 nötig)
- Hook schreibt JSON temp-file, Go-Binary liest es

### Distribution
- GitHub Releases mit Binaries für Linux/macOS/FreeBSD/Windows
- AUR Package
- Homebrew Formula
- Windows: Standalone .exe oder Scoop/Chocolatey

## Bekannte Bash-Limitierungen die Go löst

1. **Performance** — SQLite Queries statt Datei-Iteration, kein `find` für Validierung
2. **Terminal-Handling** — bubbletea managt Terminal-State sauber (kein stty-Hack)
3. **Tab-Completion** — bubbletea hat eingebaute Input-Widgets mit Completion
4. **Windows-Support** — Go kompiliert nativ für Windows
5. **Wartbarkeit** — Strukturierter Code statt 2000 Zeilen monolithisches Bash
6. **Datenformat** — SQLite mit Transaktionen statt TSV mit File-Locking
7. **Testing** — Go's eingebautes Test-Framework statt bats

## User-Präferenzen

- Sprache: Deutsch (Kommunikation), Code/Commits auf Englisch
- Keine Trial-and-Error Fixes — erst recherchieren, dann implementieren
- Casual Kommunikationsstil
- Nutzt Arch Linux (CachyOS), zsh als Shell
- Claude Code Power-User
