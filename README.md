# ccsm

Tool zum Verwalten deiner Claude-Code-Sessions: weiterführen,
durchsuchen, löschen, Token-Kosten überwachen — alles per Hotkey,
ohne SIDs zu merken.

```
ccsm           # TUI
ccsm 1         # neueste Session direkt fortsetzen
ccsm search    # alle Sessions als Tabelle
```

Installation:

```
git clone ssh://git@git.mattlab.at/mattlab-apps/ccsm-go.git
cd ccsm-go
make install
ccsm install-hook   # registriert den SessionEnd-Hook in Claude Code
```

Volle Anleitung, Tastatur-Übersicht, Architektur und Konfiguration:
**https://git.mattlab.at/mattlab-apps/ccsm-go/wiki**

Tests: `make test` oder `make test-report` (letzteres hängt einen
Eintrag mit Coverage an `TESTRESULTS.md` an).

Lizenz: siehe `LICENSE`.
