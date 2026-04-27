package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"auto-hub/web/internal/config"

	_ "modernc.org/sqlite"
)

// New opens a SQLite connection with WAL and foreign keys enabled.
func New(cfg *config.Config) (*sql.DB, error) {
	if cfg.DBPath != ":memory:" {
		dbDir := filepath.Dir(cfg.DBPath)
		if dbDir != "" && dbDir != "." {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				return nil, fmt.Errorf("create db directory: %w", err)
			}
		}
		if err := os.MkdirAll(cfg.ListingsDir, 0755); err != nil {
			return nil, fmt.Errorf("create listings directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return db, nil
}

// InitSchema creates tables and indexes if they don't exist.
func InitSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS inventory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL,
			price INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT '',
			source_url TEXT,
			status TEXT NOT NULL DEFAULT 'active',
			year INTEGER,
			make TEXT,
			model TEXT,
			mileage_km INTEGER,
			location TEXT,
			condition TEXT,
			notes TEXT,
			transmission TEXT,
			drivetrain TEXT,
			published_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS stats (
			target_type TEXT NOT NULL CHECK(target_type IN ('site', 'listing')),
			target_id INTEGER NOT NULL DEFAULT 0,
			view_count INTEGER DEFAULT 0,
			PRIMARY KEY (target_type, target_id)
		)`,
		`CREATE TABLE IF NOT EXISTS view_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			target_type TEXT NOT NULL CHECK(target_type IN ('site', 'listing')),
			target_id INTEGER NOT NULL DEFAULT 0,
			fingerprint TEXT NOT NULL,
			viewed_date TEXT NOT NULL DEFAULT (date('now')),
			viewed_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_view_log_unique
			ON view_log(target_type, target_id, fingerprint, viewed_date)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			full_name TEXT,
			is_verified INTEGER DEFAULT 0,
			verification_code TEXT,
			verification_expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}
