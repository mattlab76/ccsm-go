# ccsm - Claude Code Session Manager

Kleines Go-TUI, das alle deine Claude-Code-Sessions an einem Ort
sammelt: weiterführen, durchsuchen, löschen, Kosten überwachen,
ohne dass du dir 36-stellige Session-IDs merken musst.

```
   ██████╗ ██████╗ ███████╗███╗   ███╗
  ██╔════╝██╔════╝██╔════╝████╗ ████║
  ██║     ██║     ███████╗██╔████╔██║   Claude Code Session Manager v2.2.1
  ██║     ██║     ╚════██║██║╚██╔╝██║   11 session(s)
  ╚██████╗╚██████╗███████║██║ ╚═╝ ██║
   ╚═════╝ ╚═════╝╚══════╝╚═╝     ╚═╝

  Today: 1.3M · 7d: 3.9M · Top: opus 4.7 (100%)
```

## Installation

```bash
git clone ssh://git@git.mattlab.at/mattlab-apps/ccsm-go.git
cd ccsm-go
make install        # baut + kopiert nach ~/.local/bin/ccsm
```

Einmalig den SessionEnd-Hook in `~/.claude/settings.json` eintragen:

```json
{
  "hooks": {
    "SessionEnd": [
      {
        "matcher": "",
        "hooks": [
          { "type": "command",
            "command": "bash /home/USER/.claude/hooks/ccsm_session_end.sh",
            "timeout": 5 }
        ]
      }
    ]
  }
}
```

## Benutzung

```bash
ccsm                 # TUI
ccsm 1               # neueste Session resumen, ohne TUI
ccsm search foo      # alle Sessions mit "foo" listen
ccsm stats           # Token-Summary
```

Im TUI:

| Taste | Wirkung |
|-------|---------|
| Pfeiltasten, `j` `k` | Auswahl bewegen |
| Enter | Aktivieren (Session resumen oder Menü öffnen) |
| `1` bis `5` | Quick-Resume Session N |
| `f` | Aktuelle Session forken (gleicher Ordner, neue SID) |
| `n` | Neue Session |
| `s` | Browser mit Suche (`/`) und Sort (`s`) |
| `d` | Löschen (`Space` markiert, `a` alle expired, `A` alle) |
| `i` | Statistik (Tabellen, Plan-Vergleich, scrollbar) |
| `l` | Aktivitätslog |
| `c` | Settings (Sprache, Cleanup, Currency, Plan) |
| `q` | Beenden |
| Ctrl+C | Quit-Confirm Dialog (verhindert versehentliches Abbrechen) |

## Was es besonders macht

**Startup Health Check.** Beim Start zeigt ccsm Sessions die nicht
mehr nutzbar sind (Transcript bei Claude gelöscht, Arbeitsverzeichnis
lokal weg, oder älter als dein Cleanup-Limit). Drei Aktionen: alle
löschen, alle ignorieren, später entscheiden.

**Hook-Loss-Recovery.** Wenn Claude abrupt endet (Ctrl+C, Crash)
und der SessionEnd-Hook nicht feuert, rekonstruiert ccsm Subject und
Token-Counts direkt aus dem JSONL-Transcript unter
`~/.claude/projects/`.

**CWD-Self-Heal.** Wenn das Transcript nicht da liegt wo erwartet
(passiert nach `cd` oder `--add-dir` in Claude), durchsucht ccsm
alle Projekt-Ordner, liest den echten CWD aus dem JSONL und heilt
die DB-Row beim nächsten Resume.

**Personalisierter Kosten-Vergleich.** Stats-View zeigt
API-Äquivalent-Kosten aus den lokalen Transcripts. Wenn du in den
Settings deinen Plan-Preis hinterlegst, kommt eine Diff-Zeile dazu:
"Pay-per-use kostet diesen Monat 5300 EUR, dein Plan 184 EUR,
Ersparnis 5116 EUR/mo".

## Daten

ccsm legt alles unter `~/.claude/` ab:

- `ccsm.db` — SQLite mit Sessions, Settings, Aktivitätslog
- `ccsm-locks/` — Lock-Files für laufende Sessions
- `hooks/ccsm_session_end.sh` — vom Installer

## Architektur

```
                         ┌──────────────────────────┐
                         │     ~/.claude/ccsm.db    │
                         │  sessions / log / dismissed │
                         └────────────┬─────────────┘
                                      │
   ┌──────────────────────┐     ┌─────┴─────┐    ┌──────────────────┐
   │  ~/.claude/projects/ │ ─── │   ccsm    │ ── │   ~/.claude/     │
   │  *.jsonl transcripts │     │ (Go/TUI)  │    │  hooks/...sh     │
   │  (Kosten + Recovery) │     └─────┬─────┘    │  → /tmp/ccsm/    │
   └──────────────────────┘           │          └──────────────────┘
                                ┌─────┴──────┐
                                │  claude    │
                                │  --resume  │
                                └────────────┘
```

TUI mit bubbletea + lipgloss. DB via modernc.org/sqlite (pure Go,
kein cgo). Daten aus dem SessionEnd-Hook für die aktuelle Session,
aus den JSONL-Transcripts für historische Statistiken.

Code unter `internal/`: `app/`, `claude/`, `db/`, `hook/`, `i18n/`,
`model/`, `ui/`, `usage/`.

## Tests

```bash
make test            # ein Lauf
make test-report     # läuft + Protokoll an TESTRESULTS.md
```

Aktuell 222 Tests, 47% Coverage. Jeder `test-report`-Lauf hängt
einen Eintrag mit Timestamp, Git-SHA und Per-Paket-Coverage an
`TESTRESULTS.md` an.

## Lizenz

Wie das ursprüngliche Bash-`ccsm`, siehe `LICENSE`.
