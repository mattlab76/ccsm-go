# ccsm - Claude Code Session Manager

Ein TUI, das alle deine Claude-Code-Sessions an einem Ort sammelt:
weiterführen, löschen, durchsuchen, Kosten überwachen, ohne dass
du dir 36-stellige Session-IDs merken musst.

> Sprachen: Diese Seite ist auf Deutsch. English version below.

---

## Was ccsm löst

Wer Claude Code regelmäßig in mehreren Projekten nutzt, kennt das:

- Welche Session war nochmal die zum Dockhand-Repo?
- `claude --resume` will eine SID, nicht "die von gestern Vormittag"
- Wie viele Tokens hab ich diese Woche eigentlich verbraten?
- Bei welchem Subagent-Lauf war das schon wieder 200k Context?
- Sessions sammeln sich an, die längst kein gültiges Transcript mehr haben

ccsm hängt sich an einen `SessionEnd`-Hook von Claude Code, schreibt
Subject, CWD und Token-Counts in eine kleine SQLite-DB unter
`~/.claude/ccsm.db`, und bietet ein TUI plus ein paar CLI-Shortcuts
zum Verwalten.

---

## Installation

Variante A, aus Source (braucht Go 1.21+):

```bash
git clone ssh://git@git.mattlab.at/mattlab-apps/ccsm-go.git
cd ccsm-go
make install        # baut + kopiert nach ~/.local/bin/ccsm
```

Variante B, Hook verdrahten (einmalig pro Maschine). Trage in deine
Claude-Settings (`~/.claude/settings.json`) ein:

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

Das Hook-Script wird im Installer ausgeliefert und schreibt nach
Session-Ende eine kleine JSON-Datei nach `/tmp/ccsm/`, die ccsm beim
nächsten Aufruf einliest. Falls der Hook mal nicht feuert (Ctrl+C
in Claude, Crash), liest ccsm direkt aus den JSONL-Transcripts
unter `~/.claude/projects/`. Siehe Sektion Hook-Loss-Recovery unten.

---

## Erster Start

```bash
ccsm
```

zeigt das Hauptmenü:

```
   ██████╗ ██████╗ ███████╗███╗   ███╗
  ██╔════╝██╔════╝██╔════╝████╗ ████║
  ██║     ██║     ███████╗██╔████╔██║   Claude Code Session Manager v2.2.0
  ██║     ██║     ╚════██║██║╚██╔╝██║   11 session(s)
  ╚██████╗╚██████╗███████║██║ ╚═╝ ██║
   ╚═════╝ ╚═════╝╚══════╝╚═╝     ╚═╝

  Today: 1.3M · 7d: 3.9M · Top: opus 4.7 (100%)
══════════════════════════════════════════════════════════════════════

  Recent Sessions (enter 1-5 to resume)

  ┌───┬──────────────────┬──────────────────────────────┬─────...
  │ # │ Date             │ Subject                      │ Direc...
  ├───┼──────────────────┼──────────────────────────────┼─────...
  │ 1 │ 2026-05-16 10:47 │ Dockhand                     │ ..Doc...
  │ 2 │ 2026-05-16 08:29 │ HomeAssistant                │ ~/hom...
  │ 3 │ 2026-05-16 07:18 │ unifi os                     │ ..uni...
  └───┴──────────────────┴──────────────────────────────┴─────...

══════════════════════════════════════════════════════════════════════
  What would you like to do?

  [n] Start new session
  [s] Browse / search sessions
  [d] Delete session
  [i] Statistics
  [l] Activity Log
  [c] Settings
  [q] Quit

  ↑/↓ navigate  Enter activate  f fork session  shortcut for direct action
```

Oben links sitzt das CCSM-Logo mit Versions-Header. Mittig ein
Token-Banner (heute, letzte 7 Tage, dominantes Modell). Darunter die
5 zuletzt genutzten Sessions. Pfeiltasten navigieren, Enter aktiviert.

---

## Workflows

### Neue Session starten, `[n]`

Wählt einen Working-Directory aus, fragt nach optionalem Subject und
Tags, startet `claude` im gewählten Ordner. Nach Beenden öffnet sich
der Save-Dialog automatisch.

### Session weiterführen

Drei Wege:

- Cursor auf Session-Zeile, dann Enter
- Zahl `1` bis `5` im Hauptmenü springt direkt in die N-te Session
- `ccsm 1` auf der Shell resumed die neueste Session ohne TUI:

  ```
  $ ccsm 1
  Resuming #1: Dockhand (/home/mha/IT-Infrastruktur-projekte/open/Dockhand)
  ...Claude startet...
  ```

- `[s]` öffnet den Browser für ältere Sessions mit Suche (`/`) und
  Sort (`s`).

### Session forken, `[f]` im Hauptmenü

Cursor auf eine bestehende Session, dann `f`, startet eine neue
Claude-Session im gleichen Verzeichnis, vorausgefüllt mit Subject
und Tags der Quelle. Die neue Session bekommt eine eigene SID, keine
`--resume`-Beziehung. Nützlich, wenn du nach einem `/clear` weiter
am gleichen Projekt arbeitest und die DB-Hygiene behalten willst.

### Sessions löschen, `[d]`

```
  ↑/↓ navigate  Space mark  a all expired  A all  Enter delete  q back
```

Mehrere Sessions auf einmal löschen:

- `Space` markiert die aktuelle Zeile
- `a` markiert alle mit `[!]` (expired) auf einmal, praktisch wenn
  du viele alte Leichen wegräumst
- `A` markiert alles
- `Enter` zeigt einen Confirm-Dialog, der die ersten 10 Markierten
  auflistet

Beim Löschen werden zugehörige Einträge in der `dismissed`-Tabelle
automatisch mit aufgeräumt.

### Statistiken, `[i]`

Sieben Sektionen, alle als saubere Tabellen:

```
  ╔═╗ ╔═╗ ╔═╗ ╔╦╗   Statistics v2.2.0
║   ║   ╚═╗ ║║║   11 session(s)
╚═╝ ╚═╝ ╚═╝ ╩ ╩

══════════════════════════════════════════════════════════════════

  Overview
  ┌──────────────┬──────────────────────────────────────┐
  │ Field        │ Value                                │
  ├──────────────┼──────────────────────────────────────┤
  │ Sessions     │ 11                                   │
  │ Plan         │ Max 20x · 184.00 EUR/mo              │
  └──────────────┴──────────────────────────────────────┘

  Token consumption (from local transcripts)
  ┌────────────────┬────────────┬────────────┬────────────┬────────────┐
  │ Window         │     Tokens │      Input │     Output │       Cost │
  ├────────────────┼────────────┼────────────┼────────────┼────────────┤
  │ Today          │     579.2k │       4.3k │     574.9k │       €242 │
  │ Last 24h       │     902.3k │       5.1k │     897.2k │       €315 │
  │ Last 7 days    │       3.3M │      18.2k │       3.3M │      €1.2k │
  │ All time       │      14.7M │     209.4k │      14.5M │      €5.1k │
  └────────────────┴────────────┴────────────┴────────────┴────────────┘

  API vs Abo
  ┌──────────────────────────────────┬─────────────────────────────────┐
  │ Item                             │ Amount                          │
  ├──────────────────────────────────┼─────────────────────────────────┤
  │ Last 7 days API-equivalent       │ €1.2k                           │
  │ Monthly extrapolation (×4.33)    │ €5.3k                           │
  │ Plan: Max 20x                    │ 184.00 EUR/mo                   │
  │ Diff (API - Plan)                │ +5206 EUR  -> subscription is c.│
  └──────────────────────────────────┴─────────────────────────────────┘
```

Sticky Header bleibt oben, der Body scrollt mit Pfeiltasten,
PgUp/PgDn und g/G. Werte stammen direkt aus den lokalen
JSONL-Transcripts, gleiche Quelle wie Claudes eigenes `/usage`. Die
Kosten sind API-äquivalent (was du bei pay-per-use bezahlen würdest),
nicht dein tatsächlicher Abo-Preis.

### Settings, `[c]`

Sieben Felder mit Pfeil/Enter-Navigation:

| # | Feld | Beispiel |
|---|------|----------|
| 1 | Language | `de` oder `en` |
| 2 | Cleanup days | `180` (0 = aus) |
| 3 | Log retention | `90` Tage |
| 4 | Currency | `EUR`, `USD`, `GBP`, ... |
| 5 | USD-Currency rate | `0.92` |
| 6 | Plan name | `Max 5x`, `Pro`, `API` |
| 7 | Plan price/month | `92.00` |

Die letzten vier Werte aktivieren den personalisierten
Plan-Vergleich in der Stats-View ("Diff API - Plan").

### Aktivitätslog, `[l]`

Chronologie aller Aktionen: New, Resume, Save, Delete, Cleanup,
Settings. Sticky-Header, scrollbar.

---

## Startup Health Check

Wenn ccsm beim Start Sessions findet, die

- bei Claude Code expired sind (Transcript gelöscht, Marker `[!]` rot),
- ein fehlendes Arbeitsverzeichnis haben (Marker `[?]` amber),
- oder älter als dein konfiguriertes `cleanup_days`-Limit sind (Marker `[⌛]`),

öffnet sich ein Dialog vor dem Hauptmenü:

```
  Session Health Check        3 session(s)
══════════════════════════════════════════════
  3 session(s) flagged for your attention:
    [⌛] 3 older than your cleanup threshold (still usable)

    [⌛] 2026-04-24  Energi App
    [⌛] 2026-04-21  Lobster Partnerkanal
    [⌛] 2026-04-17  lfs7 tool

  [p] Purge all (delete from ccsm)
  [d] Dismiss all (keep, don't ask again)
▶ [s] Skip for now (ask again next start)
```

`[s]` Skip ist Default. Enter ohne Auswahl ist die sichere Option.

---

## Hook-Loss-Recovery

Wenn Claude abrupt endet (Ctrl+C, Crash, Hook-Timeout), feuert der
SessionEnd-Hook eventuell nicht und es liegen keine Daten in
`/tmp/ccsm/`. Statt blind die letzte fremde Session anzufassen (was
früher zu CWD/SID-Mismatches führte), liest ccsm dann direkt das
JSONL-Transcript unter `~/.claude/projects/{slug(cwd)}/{sid}.jsonl`
aus und rekonstruiert Subject und Token-Counts daraus. Der
Save-Dialog zeigt dann zusätzlich:

```
  ⚠ Hook missed - recovered from transcript file.
```

---

## CWD-Self-Heal

Wenn Claudes Hook eine andere `cwd` meldet als das
Projekt-Verzeichnis, in dem das JSONL gespeichert wird (passiert
nach `cd` oder `--add-dir`), verbiegt sich der gespeicherte Pfad.
Vor jedem Resume prüft ccsm jetzt:

1. Liegt das Transcript wirklich da wo wir glauben?
2. Wenn nein, durchsuche alle Projekt-Ordner nach `{sid}.jsonl`
3. Lies die erste `cwd`-Zeile aus dem Transcript (Claude tagged jede
   Message damit)
4. Heile die DB-Row stillschweigend, log den Fix im Aktivitätslog

Damit verschwindet die "No conversation found"-Meldung beim Resume.

---

## Ctrl+C Schutz

Ein versehentliches `Ctrl+C` (z.B. wenn du eigentlich `Ctrl+Shift+V`
zum Pasten wolltest) öffnet einen roten Modal-Dialog:

```
╔════════════════════════════════════════════╗
║   ⚠  Really quit ccsm?                     ║
║                                            ║
║   Running Claude sessions stay alive       ║
║   only ccsm closes.                        ║
║                                            ║
║   Press Ctrl+C again or [y]/Enter to quit  ║
║   [n]/Esc/q to stay                        ║
╚════════════════════════════════════════════╝
```

`n` oder `Esc` schließt den Dialog wieder. Greift nur im ccsm-TUI
selbst. Während Claude läuft, hat Claude die volle Kontrolle übers
Terminal.

---

## CLI-Referenz

| Befehl | Zweck |
|--------|-------|
| `ccsm` | Interaktives TUI |
| `ccsm <N>` | Resume Nth recent session direkt (1 = neueste) |
| `ccsm search [query]` | Sessions als Tabelle ausgeben |
| `ccsm stats` | Token-Summary als Text |
| `ccsm cleanup` | Alte Sessions (älter `cleanup_days`) auflisten |
| `ccsm migrate` | Daten aus der Bash-v1.x-Version importieren |
| `ccsm version` | Version |
| `ccsm completion zsh` | Zsh-Autocompletion |

---

## Keyboard Cheat Sheet

| Wo? | Taste | Was? |
|-----|-------|------|
| Überall | `Ctrl+C` | Quit-Confirm öffnen |
| Hauptmenü | Pfeil hoch/runter, `jk` | Session oder Menüpunkt wählen |
| Hauptmenü | `Enter` | Aktivieren |
| Hauptmenü | `1`-`5` | Quick-Resume Session N |
| Hauptmenü | `f` | Session forken (Cursor auf Session) |
| Hauptmenü | `n s d i l c q` | Direktwahl der Menüpunkte |
| Browser | `/` | Suche. `s` Sort wechseln |
| Browser | `g`, `G` | An den Anfang, ans Ende |
| Delete | `Space` | Markieren. `a` alle expired. `A` alle |
| Delete | `Enter` | Löschen-Confirm öffnen |
| Stats/Log | `PgUp`, `PgDn` | Body scrollen (Header bleibt) |
| Confirm-Dialoge | `y`, `j`, `Enter` | Ja. `n`, `q`, `Esc` Abbrechen |

---

## Architektur

```
                         ┌──────────────────────────┐
                         │     ~/.claude/ccsm.db    │
                         │  sessions / log / dirty  │
                         └────────────┬─────────────┘
                                      │
   ┌──────────────────────┐     ┌─────┴─────┐    ┌──────────────────┐
   │  ~/.claude/projects/ │◄────│   ccsm    │◄───│   ~/.claude/     │
   │  *.jsonl transcripts │     │ (Go/TUI)  │    │  hooks/...sh     │
   │  (read for usage     │     └─────┬─────┘    │  -> /tmp/ccsm/   │
   │   stats + recovery)  │           │          └──────────────────┘
   └──────────────────────┘           │
                                ┌─────┴──────┐
                                │  claude    │
                                │  --resume  │
                                └────────────┘
```

- TUI: `bubbletea` und `lipgloss`
- DB: SQLite (via `modernc.org/sqlite`, pure-Go, kein cgo)
- Daten-Source: Hook-JSON für aktuelle Session, JSONL-Transcripts
  für alles andere

Code-Layout (alles unter `internal/`):

```
app/        Bubbletea root model, view routing, Ctrl+C confirm
claude/     Resume/Start, lock-files, transcript helpers, cwd self-heal
db/         SQLite schema + Sessions/Settings/Log/Tokens accessors
hook/       Reader für /tmp/ccsm/session-*.json
i18n/       EN + DE Strings, Locale-Detection
model/      Session/Settings/Version
ui/         Bubbletea Child-Models pro View
usage/      JSONL-Parser, Pricing-Tabelle, Aggregation, Money-Formatter
```

---

## Tests und Mitwirken

```bash
make test            # ein Lauf
make test-v          # mit -v
make test-report     # läuft + appendet ein Protokoll an TESTRESULTS.md
```

Aktuell 222 Tests, ca. 47% Coverage. Jeder `make test-report`-Lauf
fügt einen Eintrag mit Timestamp, Git-SHA und Per-Paket-Coverage in
`TESTRESULTS.md` ein, sodass Regressionen in der Git-Historie
sichtbar bleiben.

---

## Lizenz

Wie das ursprüngliche Bash-`ccsm`. Siehe `LICENSE`.

---

# English Quickref

ccsm is a small TUI that catalogues your Claude Code sessions across
projects, so you can resume by name instead of by 36-character SID,
spot expired/orphaned sessions, see weekly token cost, and compare
pay-per-use against your subscription. Pure Go, single static binary,
SQLite-backed.

Install: `make install` (needs Go 1.21+). Wire the SessionEnd hook
into `~/.claude/settings.json` as shown above; ccsm also reads
`~/.claude/projects/*.jsonl` directly as a fallback.

Run `ccsm` for the TUI, `ccsm 1` to resume the newest session
without it. See the Keyboard Cheat Sheet above for everything else.
