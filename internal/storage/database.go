package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	"commit_craft_reborn/internal/config"

	"github.com/pkg/errors"
	_ "modernc.org/sqlite"
)

// DB is a wrapper for the sql.DB connection pool.
type DB struct {
	*sql.DB
}

// InitDB connects to the database and returns a DB instance.
func InitDB() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user home directory")
	}

	dbDir := filepath.Join(home, config.GlobalConfigDir)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return nil, errors.Wrap(err, "failed to create database directory")
	}

	dbPath := filepath.Join(dbDir, "commits.db")
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open database")
	}

	if err := createTables(sqlDB); err != nil {
		return nil, errors.Wrap(err, "failed to create tables")
	}

	return &DB{sqlDB}, nil
}

// createTables ensures the necessary tables exist in the database.
func createTables(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS commits (
            id INTEGER PRIMARY KEY,
            type TEXT NOT NULL,
            scope TEXT NOT NULL,
            message_es TEXT NOT NULL,
            message_en TEXT NOT NULL,
            workspace TEXT NOT NULL,
            created_at TEXT NOT NULL
        );
    `)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS releases (
            id INTEGER PRIMARY KEY,
			title TEXT NOT NULL,
			body TEXT NOT NULL,
            branch TEXT NOT NULL,
			commit_list TEXT NOT NULL,
            version TEXT NOT NULL,
            workspace TEXT NOT NULL,
            created_at TEXT NOT NULL
        );
    `)
	if err != nil {
		return err
	}

	_, err = db.Exec(`ALTER TABLE commits ADD COLUMN diff_code TEXT NOT NULL DEFAULT '';`)
	if err != nil {
		// If the error is "duplicate column name", we safely ignore it.
		// For any other error, we return it.
		if !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}
