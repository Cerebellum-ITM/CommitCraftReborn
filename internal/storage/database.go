package storage

import (
	"commit_craft_reborn/internal/config"
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // SQLite driver, imported for its side-effect.
	"github.com/pkg/errors"
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
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create database directory")
	}

	dbPath := filepath.Join(dbDir, "commits.db")
	sqlDB, err := sql.Open("sqlite3", dbPath)
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
	return err
}
