package db

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mattlab76/ccsm-go/internal/model"
)

// MigrateResult holds the counts of imported records.
type MigrateResult struct {
	Sessions      int
	LogEntries    int
	Settings      bool
	LifetimeIn    int64
	LifetimeOut   int64
	DismissedSIDs int
}

// MigrateFromBash imports all data from the bash ccsm version.
func MigrateFromBash(database *DB) (*MigrateResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	claudeDir := filepath.Join(home, ".claude")
	result := &MigrateResult{}

	// 1. Import sessions from TSV.
	tsvPath := filepath.Join(claudeDir, "session_log.tsv")
	if n, err := migrateTSV(database, tsvPath); err == nil {
		result.Sessions = n
	}

	// 2. Import settings from ccsm.conf.
	confPath := filepath.Join(claudeDir, "ccsm.conf")
	if migrateConf(database, confPath) {
		result.Settings = true
	}

	// 3. Import activity log from ccsm.log.
	logPath := filepath.Join(claudeDir, "ccsm.log")
	if n, err := migrateLog(database, logPath); err == nil {
		result.LogEntries = n
	}

	// 4. Import lifetime tokens.
	tokensPath := filepath.Join(claudeDir, "ccsm_tokens_total")
	if in, out, err := migrateLifetimeTokens(database, tokensPath); err == nil {
		result.LifetimeIn = in
		result.LifetimeOut = out
	}

	// 5. Import dismissed sessions.
	dismissedPath := filepath.Join(claudeDir, "ccsm_dismissed")
	if n, err := migrateDismissed(database, dismissedPath); err == nil {
		result.DismissedSIDs = n
	}

	return result, nil
}

func migrateTSV(database *DB, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 6)
		if len(fields) < 4 {
			continue
		}

		sid := fields[0]
		cwd := fields[1]
		subject := fields[2]
		dateStr := fields[3]
		tags := ""
		if len(fields) > 4 {
			tags = fields[4]
		}
		if tags == "-" {
			tags = ""
		}
		tokensStr := ""
		if len(fields) > 5 {
			tokensStr = fields[5]
		}

		// Parse date: "YYYY-MM-DD HH:MM" or "YYYY-MM-DD" (legacy).
		var createdAt time.Time
		if len(dateStr) <= 10 {
			dateStr += " 00:01"
		}
		createdAt, _ = time.Parse("2006-01-02 15:04", dateStr)
		if createdAt.IsZero() {
			createdAt = time.Now()
		}

		// Parse tokens: "total_in/total_out|last_in/last_out" or "total_in/total_out".
		var totalIn, totalOut, lastIn, lastOut int64
		if tokensStr != "" && tokensStr != "0/0" {
			totalPart := tokensStr
			lastPart := tokensStr
			if idx := strings.Index(tokensStr, "|"); idx >= 0 {
				totalPart = tokensStr[:idx]
				lastPart = tokensStr[idx+1:]
			}
			totalIn, totalOut = parseTokenPair(totalPart)
			lastIn, lastOut = parseTokenPair(lastPart)
		}

		s := &model.Session{
			SID:               sid,
			CWD:               cwd,
			Subject:           subject,
			CreatedAt:         createdAt,
			Tags:              tags,
			TotalInputTokens:  totalIn,
			TotalOutputTokens: totalOut,
			LastInputTokens:   lastIn,
			LastOutputTokens:  lastOut,
		}

		// INSERT OR IGNORE to skip duplicates.
		_, err := database.Exec(`INSERT OR IGNORE INTO sessions
			(sid, cwd, subject, created_at, tags,
			total_input_tokens, total_output_tokens,
			last_input_tokens, last_output_tokens)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.SID, s.CWD, s.Subject,
			s.CreatedAt.Format("2006-01-02 15:04:05"),
			s.Tags,
			s.TotalInputTokens, s.TotalOutputTokens,
			s.LastInputTokens, s.LastOutputTokens)
		if err == nil {
			count++
		}
	}
	return count, scanner.Err()
}

func migrateConf(database *DB, path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// Start from whatever is already configured so we only overwrite the
	// three keys the bash conf actually carries (cleanup / log / lang).
	// Importing must not reset currency, exchange rate or plan settings —
	// those exist only in the Go version and aren't in the bash conf.
	s, err := GetSettings(database)
	if err != nil {
		def := model.DefaultSettings()
		s = &def
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Strip inline comments.
		if idx := strings.Index(value, "#"); idx >= 0 {
			value = strings.TrimSpace(value[:idx])
		}

		switch key {
		case "CLEANUP_DAYS":
			if n, err := strconv.Atoi(value); err == nil {
				s.CleanupDays = n
			}
		case "LOG_DAYS":
			if n, err := strconv.Atoi(value); err == nil {
				s.LogDays = n
			}
		case "CCSM_LANG":
			s.Lang = value
		}
	}
	SaveSettings(database, s)
	return true
}

func migrateLog(database *DB, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// Format: "2026-03-24 06:04 [ACTION] message"
		if len(line) < 20 {
			continue
		}
		dateStr := line[:16]
		rest := line[17:]

		ts, _ := time.Parse("2006-01-02 15:04", dateStr)
		if ts.IsZero() {
			continue
		}

		// Extract [ACTION].
		action := ""
		message := rest
		if strings.HasPrefix(rest, "[") {
			end := strings.Index(rest, "]")
			if end > 0 {
				action = rest[1:end]
				message = strings.TrimSpace(rest[end+1:])
			}
		}

		_, err := database.Exec(
			"INSERT INTO activity_log (timestamp, action, message) VALUES (?, ?, ?)",
			ts.Format("2006-01-02 15:04:05"), action, message)
		if err == nil {
			count++
		}
	}
	return count, scanner.Err()
}

func migrateLifetimeTokens(database *DB, path string) (int64, int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Fields(strings.TrimSpace(string(data)))
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format")
	}
	in, _ := strconv.ParseInt(parts[0], 10, 64)
	out, _ := strconv.ParseInt(parts[1], 10, 64)
	if in > 0 || out > 0 {
		AddLifetimeTokens(database, in, out)
	}
	return in, out, nil
}

func migrateDismissed(database *DB, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		sid := strings.TrimSpace(scanner.Text())
		if sid == "" {
			continue
		}
		if err := AddDismissed(database, sid); err == nil {
			count++
		}
	}
	return count, scanner.Err()
}

func parseTokenPair(s string) (int64, int64) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	in, _ := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	out, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	return in, out
}
