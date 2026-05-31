# ccsm

Tool, das deine Claude-Code-Sessions verwaltet. Du arbeitest mit
Claude in mehreren Projekten parallel und willst nicht jedes Mal
36-stellige SIDs raussuchen, abgelaufene Sessions manuell aufräumen
oder im Kopf addieren wie viel Tokens du diese Woche verbraucht
hast. ccsm sitzt dazwischen: eine kleine SQLite-DB unter
`~/.claude/`, ein TUI mit Hotkeys, ein paar CLI-Shortcuts.

Geschrieben in Go, ein statisches Binary, kein cgo, kein Daemon.


## Installation

Aus Source, braucht Go 1.21 oder neuer:

```
git clone ssh://git@git.mattlab.at/mattlab-apps/ccsm-go.git
cd ccsm-go
make install
```

`make install` baut das Binary und legt es unter `~/.local/bin/ccsm`
ab. Wenn der Pfad in deiner `$PATH` steht (sollte er), funktioniert
`ccsm` ab sofort von überall.

Damit ccsm beim Session-Ende automatisch Subject, CWD und
Token-Counts erfährt, registrierst du ccsm einmalig als
SessionEnd-Hook in Claude Code:

```
ccsm install-hook
```

Das trägt `ccsm hook` idempotent in `~/.claude/settings.json` ein,
ohne andere Einstellungen anzufassen, und ersetzt dabei eine
eventuell vorhandene ältere Hook-Variante. Danach Claude Code einmal
neu starten. Mit `ccsm doctor` prüfst du anschließend, ob Hook,
`claude` im PATH und die Datenbank in Ordnung sind.

Der Hook braucht keine externen Programme: `ccsm hook` liest Claudes
Payload selbst und parst das Transcript in Go (kein jq, kein
python3). Und auch ganz ohne Hook funktioniert ccsm — fehlende Daten
zieht es direkt aus den JSONL-Transcripts unter `~/.claude/projects/`,
der Hook ist nur etwas schneller.


## Erste Schritte

Einfach `ccsm` ausführen. Beim ersten Start ist die Liste leer. Nach
ein paar Claude-Sessions sieht der Hauptbildschirm so aus:

```
   ██████╗ ██████╗ ███████╗███╗   ███╗
  ██╔════╝██╔════╝██╔════╝████╗ ████║
  ██║     ██║     ███████╗██╔████╔██║   Claude Code Session Manager v2.3.0
  ██║     ██║     ╚════██║██║╚██╔╝██║   11 sessions
  ╚██████╗╚██████╗███████║██║ ╚═╝ ██║
   ╚═════╝ ╚═════╝╚══════╝╚═╝     ╚═╝

  Today: 1.3M · 7d: 3.9M · Top: opus 4.7 (100%)
  ────────────────────────────────────────────────────────────────

  Recent Sessions (enter 1-5 to resume)
  ┌───┬──────────────────┬─────────────────────────┬──────────┐
  │ # │ Date             │ Subject                 │ Tokens   │
  ├───┼──────────────────┼─────────────────────────┼──────────┤
  │ 1 │ 2026-05-16 10:47 │ Dockhand                │  327.1M  │
  │ 2 │ 2026-05-16 08:29 │ HomeAssistant           │  287.2M  │
  │ 3 │ 2026-05-16 07:18 │ unifi os                │  125.9M  │
  └───┴──────────────────┴─────────────────────────┴──────────┘

  [n] Neue Session     [s] Browse / Search    [d] Löschen
  [i] Statistik        [l] Aktivitätslog      [c] Settings    [q] Quit
```

Oben das Token-Banner: was du heute und in den letzten sieben Tagen
verbraucht hast und welches Modell den Großteil ausmacht. Darunter
die fünf zuletzt benutzten Sessions; `1` bis `5` setzt direkt eine
davon fort. Mit den Buchstaben unten kommst du in die anderen
Bereiche, oder du navigierst per Pfeiltasten und Enter.


## Sessions verwalten

**Neue Session.** `n` im Hauptmenü öffnet einen Dialog für Working
Directory, Subject und Tags. Subject und Tags sind optional und
lassen sich am Ende der Session noch ergänzen oder ändern.

**Fortsetzen.** Drei Wege:

- Cursor auf eine Session, dann Enter
- Zahl `1` bis `5` für die fünf neuesten Sessions
- `ccsm 1` auf der Shell startet die neueste Session direkt, ohne
  Umweg übers TUI. `ccsm 2` die zweitneueste, und so weiter

**Forken.** Wenn du nach einem `/clear` weiterarbeiten willst oder
eine Variante des gleichen Themas im selben Verzeichnis öffnest,
drück `f` auf einer Session. Das startet eine neue Claude-Session
mit dem Verzeichnis, Subject und Tags der Quelle — aber eigener SID
und ohne Resume-Beziehung.

**Browsen.** `s` öffnet eine durchsuchbare Liste aller Sessions, mit
`/` zum Filtern und `s` zum Wechseln der Sortierung
(Datum, Subject, Tokens).

**Löschen.** `d` öffnet einen Auswahldialog mit der vollen Liste.
`Space` markiert Zeilen, `a` markiert alle abgelaufenen auf einmal,
`A` markiert alles. Enter zeigt einen Confirm-Dialog der bis zu
zehn Markierte als Vorschau auflistet.


## Statistik und Kosten

`i` öffnet die Statistik. Sticky Header, scrollbarer Body, alle
Werte aus den lokalen JSONL-Transcripts (gleiche Datenbasis wie
Claudes eigenes `/usage`).

Was du dort siehst:

- Übersicht: Anzahl Sessions, älteste und neueste, dein Plan
- Token-Verbrauch heute, letzte 24 Stunden, letzte sieben Tage,
  insgesamt — jeweils mit Input-, Output- und Kostenwerten
- Aufschlüsselung nach Modell für die letzten sieben Tage
- Top-Sessions nach Tokens
- Mustererkennung für die letzten 24 Stunden: wie viel deiner
  Nutzung lief mit großem Kontext, wie viel über Subagenten

Die Kostenanzeige rechnet zu Anthropic-API-Preisen. Wenn du in den
Settings deinen Plan-Preis und die Währung hinterlegst, kommt
darüber ein direkter Vergleich: "Pay-per-use diese Woche 1.200 EUR,
dein Max-20x-Plan kostet 184 EUR im Monat, Ersparnis ungefähr 5.000
EUR pro Monat".


## Was im Hintergrund passiert

Drei Mechanismen sparen dir Ärger, ohne dass du etwas tun musst:

**Startup-Health-Check.** Beim Start prüft ccsm, ob Sessions im DB
nicht mehr nutzbar sind, weil Claude das Transcript gelöscht hat,
das Arbeitsverzeichnis lokal weg ist oder die Session älter als
dein Cleanup-Limit ist. Wenn ja, kommt ein Dialog mit drei
Optionen: alle löschen, alle ignorieren, später entscheiden.

**Hook-Loss-Recovery.** Wenn Claude abrupt beendet wird, etwa durch
Ctrl-C oder einen Crash, feuert der SessionEnd-Hook nicht. Statt
dann blind irgendwelche Daten anzuhängen, liest ccsm das
JSONL-Transcript direkt aus `~/.claude/projects/` und rekonstruiert
Subject und Token-Counts daraus. Der Save-Dialog markiert solche
Fälle mit einem Hinweis.

**CWD-Self-Heal.** Wenn der Hook eine andere `cwd` meldet als das
Projekt-Verzeichnis, in dem das JSONL liegt (passiert nach `cd` oder
`--add-dir` innerhalb von Claude), verbiegt sich der gespeicherte
Pfad. Beim nächsten Resume prüft ccsm, ob das Transcript wirklich
dort liegt, durchsucht andernfalls alle Projekt-Ordner, liest die
echte `cwd` aus dem Transcript und heilt die DB-Row stillschweigend.


## Einstellungen

`c` im Hauptmenü, sieben Felder mit Pfeil-Navigation oder
Zahl-Shortcuts:

| Nr | Feld | Beispiel |
|----|------|----------|
| 1 | Sprache | `de` oder `en` |
| 2 | Cleanup-Schwelle in Tagen | `180`, oder `0` für aus |
| 3 | Log-Aufbewahrung in Tagen | `90` |
| 4 | Währungs-Code | `EUR`, `USD`, `GBP`, `CHF` |
| 5 | Wechselkurs USD zu Währung | `0.92` |
| 6 | Abo-Name | `Max 5x`, `Pro`, `API` |
| 7 | Abo-Preis pro Monat | `184.00` |

Die Felder 4 bis 7 aktivieren den personalisierten Plan-Vergleich
in der Statistik. Wenn sie leer bleiben, zeigt die Statistik die
API-Kosten in USD an, ohne Vergleich.


## CLI-Übersicht

| Befehl | Zweck |
|--------|-------|
| `ccsm` | Interaktives TUI starten |
| `ccsm 1` | N-te neueste Session direkt fortsetzen |
| `ccsm search [text]` | Sessions filtern, Ausgabe als Tabelle |
| `ccsm stats` | Statistik als Textausgabe |
| `ccsm cleanup` | Sessions älter als Cleanup-Schwelle auflisten |
| `ccsm migrate` | Daten aus der Bash-Version v1.x importieren |
| `ccsm install-hook` | ccsm als SessionEnd-Hook in Claude Code eintragen |
| `ccsm doctor` | Umgebung prüfen (claude, Hook, Datenbank) |
| `ccsm version` | Version anzeigen |
| `ccsm completion zsh` | Zsh-Autocompletion ausgeben |


## Tastatur

| Wo | Taste | Wirkung |
|----|-------|---------|
| Überall | Ctrl-C | Quit-Confirm Dialog |
| Hauptmenü | Pfeile, j, k | Auswahl bewegen |
| Hauptmenü | Enter | Aktivieren |
| Hauptmenü | 1 bis 5 | Quick-Resume Session N |
| Hauptmenü | f | Session forken |
| Hauptmenü | n, s, d, i, l, c, q | Menüpunkte direkt |
| Browser | / | Suche, s wechselt Sortierung |
| Browser | g, G | An den Anfang, ans Ende |
| Delete | Space | Markieren |
| Delete | a, A | Alle expired markieren, alle markieren |
| Delete | Enter | Löschen-Confirm öffnen |
| Stats, Log | PgUp, PgDn | Body scrollen, Header bleibt |
| Confirm-Dialoge | y, j, Enter | Bestätigen |
| Confirm-Dialoge | n, q, Esc | Abbrechen |


## Mehr

Quellcode, Issues, Releases:
https://git.mattlab.at/mattlab-apps/ccsm-go

Mirror auf GitHub: https://github.com/mattlab76/ccsm-go

Lizenz wie das ursprüngliche Bash-Projekt, siehe `LICENSE` im
Repository.
