package db

import (
	"time"

	"github.com/mattlab76/ccsm-go/internal/model"
)

// LogAction writes an activity log entry.
func LogAction(db *DB, action, message string) error {
	_, err := db.Exec(
		"INSERT INTO activity_log (timestamp, action, message) VALUES (?, ?, ?)",
		time.Now().Format("2006-01-02 15:04:05"), action, message)
	return err
}

// ListLog returns all log entries, newest first.
func ListLog(db *DB) ([]model.LogEntry, error) {
	rows, err := db.Query(
		`SELECT id, timestamp, action, message FROM activity_log ORDER BY timestamp DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.LogEntry
	for rows.Next() {
		var e model.LogEntry
		var ts string
		if err := rows.Scan(&e.ID, &ts, &e.Action, &e.Message); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// RotateLog removes log entries older than the given number of days.
// If days is 0, no rotation is performed.
func RotateLog(db *DB, days int) (int64, error) {
	if days <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	result, err := db.Exec("DELETE FROM activity_log WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
