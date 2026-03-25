package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the ccsm database at ~/.claude/ccsm.db.
func Open() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create %s: %w", dir, err)
	}
	return OpenPath(filepath.Join(dir, "ccsm.db"))
}

// OpenPath opens (or creates) the ccsm database at the given path.
func OpenPath(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}
	// Enable WAL mode for better concurrent access.
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("cannot set WAL mode: %w", err)
	}
	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	return db, nil
}

// OpenMemory opens an in-memory database (for testing).
func OpenMemory() (*DB, error) {
	return OpenPath(":memory:")
}

const schemaVersion = 1

func (db *DB) migrate() error {
	// Create schema_version table first.
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY
	)`); err != nil {
		return err
	}

	var current int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&current)
	if err != nil {
		return err
	}

	if current >= schemaVersion {
		return nil
	}

	// Version 1: initial schema.
	if current < 1 {
		stmts := []string{
			`CREATE TABLE IF NOT EXISTS sessions (
				sid TEXT PRIMARY KEY,
				cwd TEXT NOT NULL,
				subject TEXT NOT NULL,
				created_at DATETIME NOT NULL,
				tags TEXT DEFAULT '',
				total_input_tokens INTEGER DEFAULT 0,
				total_output_tokens INTEGER DEFAULT 0,
				last_input_tokens INTEGER DEFAULT 0,
				last_output_tokens INTEGER DEFAULT 0
			)`,
			`CREATE TABLE IF NOT EXISTS activity_log (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp DATETIME NOT NULL,
				action TEXT NOT NULL,
				message TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS settings (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS dismissed (
				sid TEXT PRIMARY KEY
			)`,
			`CREATE TABLE IF NOT EXISTS lifetime_tokens (
				id INTEGER PRIMARY KEY CHECK (id = 1),
				total_input INTEGER DEFAULT 0,
				total_output INTEGER DEFAULT 0
			)`,
			`INSERT OR IGNORE INTO lifetime_tokens (id, total_input, total_output) VALUES (1, 0, 0)`,
			`INSERT OR REPLACE INTO schema_version (version) VALUES (1)`,
		}
		for _, stmt := range stmts {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("migration v1: %w\nSQL: %s", err, stmt)
			}
		}
	}

	return nil
}
