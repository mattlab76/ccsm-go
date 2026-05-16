# ccsm

Kleines Terminal-Tool, das deine Claude-Code-Sessions ordnet.
Statt SIDs zu merken, suchst, fortführst, löschst und vergleichst
du Sessions per Hotkey.

```
   ██████╗ ██████╗ ███████╗███╗   ███╗
  ██╔════╝██╔════╝██╔════╝████╗ ████║
  ██║     ██║     ███████╗██╔████╔██║
  ██║     ██║     ╚════██║██║╚██╔╝██║
  ╚██████╗╚██████╗███████║██║ ╚═╝ ██║
   ╚═════╝ ╚═════╝╚══════╝╚═╝     ╚═╝
```

## Warum

Wer mehrere Projekte parallel mit Claude Code bearbeitet, sammelt
schnell zwanzig anonyme Sessions an. ccsm gibt jeder einen Namen,
zeigt was sie kostet, und räumt abgelaufene auf.

## Install

```bash
git clone ssh://git@git.mattlab.at/mattlab-apps/ccsm-go.git
cd ccsm-go && make install
```

Plus einmalig den SessionEnd-Hook in `~/.claude/settings.json`
verdrahten. Snippet steht in der
[README im Repo](https://git.mattlab.at/mattlab-apps/ccsm-go).

## Loslegen

```bash
ccsm        # TUI: Pfeile, Enter, Buchstaben-Hotkeys
ccsm 1      # neueste Session direkt fortführen, ohne TUI
```

## Vollständige Doku

Die komplette Anleitung (Workflows, Tastatur-Cheat-Sheet, Architektur)
steht als [README im Repo](https://git.mattlab.at/mattlab-apps/ccsm-go).

Quellcode, Issues, Releases:
**https://git.mattlab.at/mattlab-apps/ccsm-go**
