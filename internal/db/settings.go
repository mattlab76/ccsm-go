package db

import (
	"strconv"

	"github.com/mattlab76/ccsm-go/internal/model"
)

// settings keys used in the key/value settings table.
const (
	keyCleanupDays      = "cleanup_days"
	keyLogDays          = "log_days"
	keyLang             = "lang"
	keyCurrency         = "currency"
	keyExchangeRate     = "exchange_rate"
	keyPlanName         = "plan_name"
	keyPlanMonthlyPrice = "plan_monthly_price"
)

// GetSettings reads all settings from the database, returning defaults for missing keys.
func GetSettings(db *DB) (*model.Settings, error) {
	s := model.DefaultSettings()

	rows, err := db.Query("SELECT key, value FROM settings")
	if err != nil {
		return &s, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			continue
		}
		switch key {
		case keyCleanupDays:
			if v, err := strconv.Atoi(value); err == nil {
				s.CleanupDays = v
			}
		case keyLogDays:
			if v, err := strconv.Atoi(value); err == nil {
				s.LogDays = v
			}
		case keyLang:
			s.Lang = value
		case keyCurrency:
			s.Currency = value
		case keyExchangeRate:
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				s.ExchangeRate = v
			}
		case keyPlanName:
			s.PlanName = value
		case keyPlanMonthlyPrice:
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				s.PlanMonthlyPrice = v
			}
		}
	}
	return &s, rows.Err()
}

// SaveSettings persists all settings to the database.
func SaveSettings(db *DB, s *model.Settings) error {
	pairs := map[string]string{
		keyCleanupDays:      strconv.Itoa(s.CleanupDays),
		keyLogDays:          strconv.Itoa(s.LogDays),
		keyLang:             s.Lang,
		keyCurrency:         s.Currency,
		keyExchangeRate:     strconv.FormatFloat(s.ExchangeRate, 'f', 4, 64),
		keyPlanName:         s.PlanName,
		keyPlanMonthlyPrice: strconv.FormatFloat(s.PlanMonthlyPrice, 'f', 2, 64),
	}
	for key, value := range pairs {
		if _, err := db.Exec(
			"INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)",
			key, value,
		); err != nil {
			return err
		}
	}
	return nil
}

// GetDismissed returns all dismissed session IDs.
func GetDismissed(db *DB) ([]string, error) {
	rows, err := db.Query("SELECT sid FROM dismissed")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sids []string
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			continue
		}
		sids = append(sids, sid)
	}
	return sids, rows.Err()
}

// AddDismissed adds a session ID to the dismissed list.
func AddDismissed(db *DB, sid string) error {
	_, err := db.Exec("INSERT OR IGNORE INTO dismissed (sid) VALUES (?)", sid)
	return err
}

// RemoveDismissed removes session IDs from the dismissed list.
func RemoveDismissed(db *DB, sids []string) error {
	for _, sid := range sids {
		if _, err := db.Exec("DELETE FROM dismissed WHERE sid = ?", sid); err != nil {
			return err
		}
	}
	return nil
}
