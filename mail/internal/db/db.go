// Package db handles database connections, schema creation, and lightweight
// migration helpers for both SQLite and PostgreSQL backends.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"auto-hub/mail/internal/config"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// New opens a connection based on cfg.DBDriver.  For SQLite it enables
// foreign keys and WAL mode.  For PostgreSQL it relies on the connection
// string in cfg.DatabaseURL.
func New(cfg *config.Config) (*sql.DB, error) {
	switch cfg.DBDriver {
	case "postgres":
		if cfg.DatabaseURL == "" {
			return nil, fmt.Errorf("DATABASE_URL is required for postgres driver")
		}
		db, err := sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("open postgres: %w", err)
		}
		if err := db.Ping(); err != nil {
			return nil, fmt.Errorf("ping postgres: %w", err)
		}
		return db, nil

	default:
		// SQLite (default)
		dbDir := filepath.Dir(cfg.DatabasePath)
		if dbDir != "" && dbDir != "." {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				return nil, fmt.Errorf("create db directory: %w", err)
			}
		}

		db, err := sql.Open("sqlite", cfg.DatabasePath)
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
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "duplicate column name") ||
			strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "column \"thread_id\" of relation") ||
			strings.Contains(msg, "column \"status\" of relation") {
			return nil // already migrated
		}
	}
	return err
}
