package storage

import (
	"database/sql"
	"fmt"
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

type columnAlteration struct {
	tableName    string
	columnName   string
	columnType   string
	defaultValue string
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

func applySchemaMigrations(db *sql.DB) error {
	alterations := []columnAlteration{
		{
			tableName:    "commits",
			columnName:   "diff_code",
			columnType:   "TEXT",
			defaultValue: "''",
		},
		{
			tableName:    "releases",
			columnName:   "type",
			columnType:   "TEXT",
			defaultValue: "''",
		},
	}

	for _, alt := range alterations {
		query := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN %s %s NOT NULL DEFAULT %s;",
			alt.tableName,
			alt.columnName,
			alt.columnType,
			alt.defaultValue,
		)

		_, err := db.Exec(query)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			return fmt.Errorf(
				"failed to add column %s to table %s: %w",
				alt.columnName,
				alt.tableName,
				err,
			)
		}
	}

	return nil
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

	err = applySchemaMigrations(db)
	if err != nil {
		// If the error is "duplicate column name", we safely ignore it.
		// For any other error, we return it.
		if !strings.Contains(err.Error(), "duplicate column name") {
			return err
		}
	}
	return nil
}
