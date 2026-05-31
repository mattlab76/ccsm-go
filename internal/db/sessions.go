package db

import (
	"sort"
	"strings"
	"time"

	"github.com/mattlab76/ccsm-go/internal/model"
)

// ListSessions returns sessions ordered by created_at DESC.
// If limit > 0, at most limit sessions are returned.
func ListSessions(db *DB, limit int) ([]model.Session, error) {
	query := `SELECT sid, cwd, subject, created_at, tags,
		total_input_tokens, total_output_tokens,
		last_input_tokens, last_output_tokens
		FROM sessions ORDER BY created_at DESC`
	if limit > 0 {
		query += " LIMIT ?"
	}

	var rows_ interface {
		Next() bool
		Scan(...any) error
		Err() error
		Close() error
	}
	var err error
	if limit > 0 {
		rows_, err = db.Query(query, limit)
	} else {
		rows_, err = db.Query(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows_.(interface{ Close() error }).Close()

	return scanSessions(rows_)
}

// GetSession returns a single session by SID.
func GetSession(db *DB, sid string) (*model.Session, error) {
	row := db.QueryRow(`SELECT sid, cwd, subject, created_at, tags,
		total_input_tokens, total_output_tokens,
		last_input_tokens, last_output_tokens
		FROM sessions WHERE sid = ?`, sid)

	var s model.Session
	var createdAt any
	err := row.Scan(&s.SID, &s.CWD, &s.Subject, &createdAt, &s.Tags,
		&s.TotalInputTokens, &s.TotalOutputTokens,
		&s.LastInputTokens, &s.LastOutputTokens)
	if err != nil {
		return nil, err
	}
	s.CreatedAt = parseDateTime(createdAt)
	return &s, nil
}

// SaveSession inserts or updates a session.
func SaveSession(db *DB, s *model.Session) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO sessions
		(sid, cwd, subject, created_at, tags,
		total_input_tokens, total_output_tokens,
		last_input_tokens, last_output_tokens)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.SID, s.CWD, s.Subject,
		s.CreatedAt.Format("2006-01-02 15:04:05"),
		s.Tags,
		s.TotalInputTokens, s.TotalOutputTokens,
		s.LastInputTokens, s.LastOutputTokens)
	return err
}

// DeleteSession removes a session by SID. Any matching dismissed-list
// entry is removed too, so it can't outlive the session and become stale.
func DeleteSession(db *DB, sid string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM sessions WHERE sid = ?", sid); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM dismissed WHERE sid = ?", sid); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteSessions removes multiple sessions by SID, and prunes any
// matching dismissed-list entries in the same transaction.
func DeleteSessions(db *DB, sids []string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, sid := range sids {
		if _, err := tx.Exec("DELETE FROM sessions WHERE sid = ?", sid); err != nil {
			return err
		}
		if _, err := tx.Exec("DELETE FROM dismissed WHERE sid = ?", sid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SearchSessions searches by subject, cwd, and tags (case-insensitive).
func SearchSessions(db *DB, query string) ([]model.Session, error) {
	like := "%" + query + "%"
	rows, err := db.Query(`SELECT sid, cwd, subject, created_at, tags,
		total_input_tokens, total_output_tokens,
		last_input_tokens, last_output_tokens
		FROM sessions
		WHERE subject LIKE ? COLLATE NOCASE
		   OR cwd LIKE ? COLLATE NOCASE
		   OR tags LIKE ? COLLATE NOCASE
		ORDER BY created_at DESC`, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// GetOldSessions returns sessions older than the given number of days.
func GetOldSessions(db *DB, days int) ([]model.Session, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	rows, err := db.Query(`SELECT sid, cwd, subject, created_at, tags,
		total_input_tokens, total_output_tokens,
		last_input_tokens, last_output_tokens
		FROM sessions WHERE created_at < ? ORDER BY created_at ASC`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// MoveToEnd updates a session's created_at to now (so it appears at top of recent list).
func MoveToEnd(db *DB, sid string) error {
	_, err := db.Exec("UPDATE sessions SET created_at = ? WHERE sid = ?",
		time.Now().Format("2006-01-02 15:04:05"), sid)
	return err
}

// GetAllTags returns all unique tags across all sessions, sorted by frequency.
func GetAllTags(db *DB) ([]string, error) {
	rows, err := db.Query("SELECT tags FROM sessions WHERE tags != '' AND tags != '-'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var tags string
		if err := rows.Scan(&tags); err != nil {
			continue
		}
		for _, tag := range strings.Fields(tags) {
			counts[tag]++
		}
	}

	// Sort by frequency (most used first).
	type tagCount struct {
		tag   string
		count int
	}
	var sorted []tagCount
	for tag, count := range counts {
		sorted = append(sorted, tagCount{tag, count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })

	result := make([]string, len(sorted))
	for i, tc := range sorted {
		result[i] = tc.tag
	}
	return result, nil
}

// CountSessions returns the total number of sessions.
func CountSessions(db *DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	return count, err
}

type scannable interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanSessions(rows scannable) ([]model.Session, error) {
	var sessions []model.Session
	for rows.Next() {
		var s model.Session
		var createdAt any
		if err := rows.Scan(&s.SID, &s.CWD, &s.Subject, &createdAt, &s.Tags,
			&s.TotalInputTokens, &s.TotalOutputTokens,
			&s.LastInputTokens, &s.LastOutputTokens); err != nil {
			return nil, err
		}
		s.CreatedAt = parseDateTime(createdAt)
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func parseDateTime(v any) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val
	case string:
		for _, layout := range []string{
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			"2006-01-02T15:04:05Z",
			time.RFC3339,
		} {
			if t, err := time.Parse(layout, val); err == nil {
				return t
			}
		}
	case int64:
		return time.Unix(val, 0)
	}
	return time.Time{}
}
