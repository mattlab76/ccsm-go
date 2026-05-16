# ccsm — Claude Code Session Manager

A small terminal UI that keeps track of your Claude Code sessions
across projects: resume by name instead of 36-character SID, spot
expired/orphaned sessions, see weekly token cost, and compare what
pay-per-use API would cost vs. your subscription. Pure Go, single
static binary, SQLite-backed.

```
   ██████╗ ██████╗ ███████╗███╗   ███╗
  ██╔════╝██╔════╝██╔════╝████╗ ████║
  ██║     ██║     ███████╗██╔████╔██║   Claude Code Session Manager v2.2.0
  ██║     ██║     ╚════██║██║╚██╔╝██║   11 session(s)
  ╚██████╗╚██████╗███████║██║ ╚═╝ ██║
   ╚═════╝ ╚═════╝╚══════╝╚═╝     ╚═╝

  Today: 1.3M · 7d: 3.9M · Top: opus 4.7 (100%)
```

---

## Quick install

```bash
git clone ssh://git@git.mattlab.at/mattlab-apps/ccsm-go.git
cd ccsm-go
make install        # → ~/.local/bin/ccsm
```

Plus a one-time hook entry in `~/.claude/settings.json` — see
[docs/WIKI.md](docs/WIKI.md#installation) for the snippet.

## Quick use

```bash
ccsm            # interactive TUI
ccsm 1          # resume the newest session, no TUI
ccsm search     # list all sessions as a table
ccsm stats      # text summary
```

In the TUI: `↑↓` navigate · `Enter` activate · `f` fork the cursor
session · letter shortcuts (`n` new, `s` search, `d` delete,
`i` stats, `l` log, `c` settings, `q` quit) · `Ctrl+C` brings up a
confirm dialog (no accidental quits).

## Full docs

→ **[docs/WIKI.md](docs/WIKI.md)** — workflows, screenshots,
architecture, keyboard cheat sheet, German + English.

## Tests

```bash
make test-report
```

Runs the suite, appends a dated entry to
[TESTRESULTS.md](TESTRESULTS.md) with per-package coverage so
regressions stay visible in git history.

## License

See `LICENSE`.
