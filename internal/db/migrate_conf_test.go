package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattlab76/ccsm-go/internal/model"
)

// TestMigrateConf_PreservesCurrencyAndPlan is a regression test: migrateConf
// used to start from DefaultSettings() and SaveSettings() every key, which
// reset currency/exchange-rate/plan (Go-only settings absent from the bash
// conf) whenever a user ran `ccsm migrate` after configuring them. The conf
// only carries cleanup/log/lang, so only those may change.
func TestMigrateConf_PreservesCurrencyAndPlan(t *testing.T) {
	database, err := OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	defer database.Close()

	// User already configured currency + plan in the Go version.
	pre := model.DefaultSettings()
	pre.Currency = "EUR"
	pre.ExchangeRate = 0.92
	pre.PlanName = "Max 20x"
	pre.PlanMonthlyPrice = 184.0
	if err := SaveSettings(database, &pre); err != nil {
		t.Fatal(err)
	}

	// Bash conf carries only cleanup/log/lang.
	confPath := filepath.Join(t.TempDir(), "ccsm.conf")
	if err := os.WriteFile(confPath,
		[]byte("CLEANUP_DAYS=180\nLOG_DAYS=45\nCCSM_LANG=de\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if !migrateConf(database, confPath) {
		t.Fatal("migrateConf returned false")
	}

	got, err := GetSettings(database)
	if err != nil {
		t.Fatal(err)
	}
	// Conf-backed keys imported from the bash file.
	if got.CleanupDays != 180 || got.LogDays != 45 || got.Lang != "de" {
		t.Errorf("conf keys not imported: cleanup=%d log=%d lang=%q",
			got.CleanupDays, got.LogDays, got.Lang)
	}
	// Go-only keys must be preserved, not reset to defaults.
	if got.Currency != "EUR" || got.ExchangeRate != 0.92 ||
		got.PlanName != "Max 20x" || got.PlanMonthlyPrice != 184.0 {
		t.Errorf("currency/plan clobbered: currency=%q rate=%v plan=%q price=%v",
			got.Currency, got.ExchangeRate, got.PlanName, got.PlanMonthlyPrice)
	}
}
