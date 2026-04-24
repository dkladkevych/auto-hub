// Package db handles SQLite database connections, schema creation, and
// lightweight migration helpers for the mail control panel.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"auto-hub/mail/internal/config"

	_ "modernc.org/sqlite"
)

// New opens a connection to the SQLite database defined in cfg, enables
// foreign-key support and WAL mode, then verifies connectivity with Ping.
func New(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

// Seed executes the SQL script at seedPath against the database.  It is
// intended for one-time development/demo data and is skipped in production.
func Seed(db *sql.DB, seedPath string) error {
	data, err := os.ReadFile(seedPath)
	if err != nil {
		return fmt.Errorf("read seed: %w", err)
	}
	_, err = db.Exec(string(data))
	return err
}

// RunMigration executes a single SQL migration file.  If the migration tries
// to add a column that already exists, the error is swallowed so the command
// can be safely re-run.
func RunMigration(db *sql.DB, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", path, err)
	}
	_, err = db.Exec(string(data))
	if err != nil && strings.Contains(err.Error(), "duplicate column name") {
		return nil // already migrated
	}
	return err
}
