package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/app"
	"github.com/mattlab76/ccsm-go/internal/claude"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/hook"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
	"github.com/spf13/cobra"
)

func main() {
	// Detect and set language before any output.
	i18n.SetLang(i18n.DetectLang())

	rootCmd := &cobra.Command{
		Use:   "ccsm [N]",
		Short: i18n.T("title"),
		Long: fmt.Sprintf(`%s v%s

  ccsm            launch the interactive TUI
  ccsm <N>        resume the Nth most recent session directly (1 = newest)
`, i18n.T("title"), model.Version),
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// Single numeric argument = quick-resume the Nth session
			// without opening the TUI.
			if len(args) == 1 {
				if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
					quickResume(n)
					return
				}
			}
			runTUI()
		},
	}

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(statsCmd())
	rootCmd.AddCommand(cleanupCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(hookCmd())
	rootCmd.AddCommand(installHookCmd())
	rootCmd.AddCommand(doctorCmd())

	// Also support --version flag.
	rootCmd.Version = model.Version
	rootCmd.SetVersionTemplate(fmt.Sprintf("ccsm v%s\n", model.Version))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func openDB() *db.DB {
	database, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return database
}

func loadLang(database *db.DB) {
	settings, err := db.GetSettings(database)
	if err == nil && settings.Lang != "" {
		i18n.SetLang(settings.Lang)
	}
}

// quickResume launches `claude --resume <SID>` for the Nth most recent
// session in the DB (1 = newest) without opening the TUI. Mirrors the
// safety net app.startResume applies: cleans stale lock files, self-
// heals the stored cwd if Claude's transcript moved, refuses if a
// session is already active in that directory.
func quickResume(n int) {
	database := openDB()
	defer database.Close()
	loadLang(database)
	claude.CleanupStaleLocks()

	sessions, err := db.ListSessions(database, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if n > len(sessions) {
		fmt.Fprintf(os.Stderr,
			"Error: only %d session(s) available, asked for #%d\n",
			len(sessions), n)
		os.Exit(1)
	}
	s := sessions[n-1]

	// Self-heal stored cwd if Claude's transcript actually lives
	// elsewhere (e.g. because the hook captured a post-`cd` cwd).
	cwd := claude.ResolveSessionCWD(s.SID, s.CWD)
	if cwd != s.CWD {
		old := s.CWD
		s.CWD = cwd
		_ = db.SaveSession(database, &s)
		db.LogAction(database, "RESUME",
			"auto-fixed cwd: "+old+" → "+cwd)
	}

	if claude.IsClaudeRunningInDir(cwd) {
		fmt.Fprintf(os.Stderr,
			"A Claude session is already active in %s — refusing to start a second one.\n",
			cwd)
		os.Exit(1)
	}

	fmt.Printf("Resuming #%d: %s (%s)\n", n, s.Subject, cwd)
	_ = claude.CreateSessionLock(cwd, os.Getpid())
	defer claude.RemoveSessionLock(cwd)

	db.LogAction(database, "RESUME", s.Subject+" ("+cwd+")")
	_ = db.MoveToEnd(database, s.SID)

	// Hand off to claude. stdin/stdout/stderr stay attached so the
	// user gets a normal interactive session.
	cmd := exec.Command("claude", "--resume", s.SID)
	cmd.Dir = cwd
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Claude exits non-zero on /quit etc. — that's not a ccsm error.
		// Only flag if it never started.
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintf(os.Stderr, "Error running claude: %v\n", err)
			os.Exit(1)
		}
	}
}

func runTUI() {
	database := openDB()
	defer database.Close()

	loadLang(database)

	// Startup housekeeping: clean up stale lock files.
	claude.CleanupStaleLocks()

	// Startup housekeeping: rotate activity log.
	if settings, err := db.GetSettings(database); err == nil && settings.LogDays > 0 {
		db.RotateLog(database, settings.LogDays)
	}

	m := app.New(database)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ccsm v%s\n", model.Version)
		},
	}
}

func searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: i18n.T("search_title"),
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			database := openDB()
			defer database.Close()
			loadLang(database)

			var sessions []model.Session
			var err error
			if len(args) == 0 {
				sessions, err = db.ListSessions(database, 0)
			} else {
				sessions, err = db.SearchSessions(database, args[0])
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if len(sessions) == 0 {
				fmt.Println(i18n.T("search_no_results"))
				return
			}

			rows := make([]components.TableRow, len(sessions))
			for i, s := range sessions {
				rows[i] = components.TableRow{Num: i + 1, Session: s, Status: components.StatusOK}
			}
			fmt.Println(components.RenderTable(rows, components.TableFull, 120))
		},
	}
}

func statsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: i18n.T("stats_title"),
		Run: func(cmd *cobra.Command, args []string) {
			database := openDB()
			defer database.Close()
			loadLang(database)

			sessions, _ := db.ListSessions(database, 0)
			lifetimeIn, lifetimeOut, _ := db.GetLifetimeTokens(database)

			fmt.Printf("%s v%s — %d sessions\n\n", i18n.T("title"), model.Version, len(sessions))

			if len(sessions) == 0 {
				fmt.Println(i18n.T("stats_none"))
				return
			}

			// Active totals.
			var activeIn, activeOut int64
			for _, s := range sessions {
				activeIn += s.TotalInputTokens
				activeOut += s.TotalOutputTokens
			}

			fmt.Printf("%s:\n", i18n.T("stats_active"))
			fmt.Printf("  %s: %s\n", i18n.T("stats_input"), components.FormatTokens(activeIn))
			fmt.Printf("  %s: %s\n\n", i18n.T("stats_output"), components.FormatTokens(activeOut))

			if lifetimeIn > 0 || lifetimeOut > 0 {
				fmt.Printf("%s:\n", i18n.T("stats_lifetime"))
				fmt.Printf("  %s: %s\n", i18n.T("stats_input"), components.FormatTokens(lifetimeIn))
				fmt.Printf("  %s: %s\n\n", i18n.T("stats_output"), components.FormatTokens(lifetimeOut))
			}

			rows := make([]components.TableRow, len(sessions))
			for i, s := range sessions {
				rows[i] = components.TableRow{Num: i + 1, Session: s, Status: components.StatusOK}
			}
			fmt.Println(components.RenderTable(rows, components.TableFull, 120))
		},
	}
}

func cleanupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup",
		Short: "Remove old sessions",
		Run: func(cmd *cobra.Command, args []string) {
			database := openDB()
			defer database.Close()
			loadLang(database)

			settings, _ := db.GetSettings(database)
			if settings.CleanupDays <= 0 {
				fmt.Println("Cleanup disabled (cleanup_days = 0)")
				return
			}

			old, _ := db.GetOldSessions(database, settings.CleanupDays)
			if len(old) == 0 {
				fmt.Printf("No sessions older than %d days.\n", settings.CleanupDays)
				return
			}

			fmt.Printf("%s:\n\n", i18n.T("cleanup_title", settings.CleanupDays))
			for i, s := range old {
				fmt.Printf("  %d. %s — %s (%s)\n", i+1, s.CreatedAt.Format("2006-01-02"), s.Subject, s.CWD)
			}
			fmt.Printf("\n%d session(s) found. Use interactive mode (ccsm) to delete.\n", len(old))
		},
	}
}

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Import data from ccsm bash version",
		Run: func(cmd *cobra.Command, args []string) {
			database := openDB()
			defer database.Close()

			fmt.Println("Migrating data from ccsm bash version...")
			result, err := db.MigrateFromBash(database)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("  Sessions:  %d imported\n", result.Sessions)
			fmt.Printf("  Log:       %d entries imported\n", result.LogEntries)
			fmt.Printf("  Settings:  %v\n", result.Settings)
			if result.LifetimeIn > 0 || result.LifetimeOut > 0 {
				fmt.Printf("  Lifetime:  %s in / %s out\n",
					components.FormatTokens(result.LifetimeIn),
					components.FormatTokens(result.LifetimeOut))
			}
			if result.DismissedSIDs > 0 {
				fmt.Printf("  Dismissed: %d sessions\n", result.DismissedSIDs)
			}
			fmt.Println("\nMigration complete.")
		},
	}
}

// hookCmd is the SessionEnd hook entry point. Claude Code invokes
// `ccsm hook` with the hook payload on stdin; we parse it and spool the
// session metadata for the TUI. Hidden from help — users never call it
// directly, they register it via `ccsm install-hook`.
func hookCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "hook",
		Short:  "SessionEnd hook entry point (reads JSON from stdin)",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			// Best-effort: never surface an error that could disrupt
			// Claude Code's shutdown.
			_ = hook.HandleSessionEnd(os.Stdin)
		},
	}
}

// installHookCmd registers this ccsm binary as Claude Code's SessionEnd
// hook in ~/.claude/settings.json, idempotently and without clobbering
// other settings.
func installHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-hook",
		Short: "Register ccsm as Claude Code's SessionEnd hook",
		Run: func(cmd *cobra.Command, args []string) {
			exe, err := os.Executable()
			if err != nil {
				fmt.Fprintf(os.Stderr, "cannot determine ccsm path: %v\n", err)
				os.Exit(1)
			}
			action, path, err := hook.InstallSessionEndHook(exe)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			switch action {
			case "added":
				fmt.Printf("✓ SessionEnd hook added to %s\n", path)
			case "updated":
				fmt.Printf("✓ SessionEnd hook updated in %s\n", path)
			case "unchanged":
				fmt.Printf("✓ SessionEnd hook already up to date (%s)\n", path)
			}
			fmt.Println("  Restart Claude Code once for the change to take effect.")
		},
	}
}

// doctorCmd runs environment checks and prints a ✓/✗ report.
func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check ccsm's environment (claude, hook, database)",
		Run: func(cmd *cobra.Command, args []string) {
			allOK := true
			check := func(label string, pass bool, detail string) {
				mark := "✓"
				if !pass {
					mark = "✗"
					allOK = false
				}
				if detail != "" {
					fmt.Printf("  %s %s — %s\n", mark, label, detail)
				} else {
					fmt.Printf("  %s %s\n", mark, label)
				}
			}

			claudeOK := claude.IsInstalled()
			claudeDetail := ""
			if !claudeOK {
				claudeDetail = "install Claude Code or add it to PATH"
			}
			check("claude on PATH", claudeOK, claudeDetail)

			exe, _ := os.Executable()
			check("ccsm binary", exe != "", exe)

			installed, command, herr := hook.HookInstalled()
			switch {
			case herr != nil:
				check("SessionEnd hook", false, herr.Error())
			case installed:
				check("SessionEnd hook", true, command)
				if exe != "" && !strings.Contains(command, exe) {
					fmt.Println("    • hook points at a different command; run 'ccsm install-hook' to use this binary")
				}
			default:
				check("SessionEnd hook", false, "not registered — run 'ccsm install-hook'")
			}

			if database, derr := db.Open(); derr != nil {
				check("database", false, derr.Error())
			} else {
				n, _ := db.CountSessions(database)
				database.Close()
				check("database", true, fmt.Sprintf("%d session(s)", n))
			}

			fmt.Println()
			if allOK {
				fmt.Println("All good.")
			} else {
				fmt.Println("Some checks failed — see above.")
				os.Exit(1)
			}
		},
	}
}
