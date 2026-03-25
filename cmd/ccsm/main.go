package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattlab76/ccsm-go/internal/app"
	"github.com/mattlab76/ccsm-go/internal/db"
	"github.com/mattlab76/ccsm-go/internal/i18n"
	"github.com/mattlab76/ccsm-go/internal/model"
	"github.com/mattlab76/ccsm-go/internal/ui/components"
	"github.com/spf13/cobra"
)

func main() {
	// Detect and set language before any output.
	i18n.SetLang(i18n.DetectLang())

	rootCmd := &cobra.Command{
		Use:   "ccsm",
		Short: i18n.T("title"),
		Long:  fmt.Sprintf("%s v%s", i18n.T("title"), model.Version),
		Run: func(cmd *cobra.Command, args []string) {
			runTUI()
		},
	}

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(statsCmd())
	rootCmd.AddCommand(cleanupCmd())
	rootCmd.AddCommand(migrateCmd())

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

func runTUI() {
	database := openDB()
	defer database.Close()

	loadLang(database)

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
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			database := openDB()
			defer database.Close()
			loadLang(database)

			sessions, err := db.SearchSessions(database, args[0])
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
