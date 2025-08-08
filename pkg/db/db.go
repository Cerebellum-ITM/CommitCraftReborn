package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"commit_craft_reborn/pkg/model"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/pkg/errors"
)

// DB wraps the sql.DB connection pool.
type DB struct {
	*sql.DB
}

// New connects to the database and returns a DB instance.
func New() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user home directory")
	}

	dbDir := filepath.Join(home, ".commitcraft")
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

// createTables ensures the necessary tables exist.
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

// GetCommits retrieves all commits from the database.
func (db *DB) GetCommits() ([]model.Commit, error) {
	rows, err := db.Query(
		"SELECT id, type, scope, message_es, message_en, workspace, created_at FROM commits ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query commits")
	}
	defer rows.Close()

	var commits []model.Commit
	for rows.Next() {
		var c model.Commit
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Type, &c.Scope, &c.MessageES, &c.MessageEN, &c.Workspace, &createdAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan commit row")
		}
		// Parse the UTC time from the database
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse created_at")
		}
		c.CreatedAt = t.Local() // Convert to local time for display
		commits = append(commits, c)
	}
	return commits, nil
}

// CreateCommit adds a new commit to the database.
func (db *DB) CreateCommit(c model.Commit) error {
	// Simple placeholder for translation
	messageEN := c.MessageES
	// Store time in UTC
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(
		"INSERT INTO commits (type, scope, message_es, message_en, workspace, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		c.Type, c.Scope, c.MessageES, messageEN, c.Workspace, createdAt,
	)
	return errors.Wrap(err, "failed to insert commit")
}

// DeleteCommit removes a commit from the database by its ID.
func (db *DB) DeleteCommit(id int) error {
	_, err := db.Exec("DELETE FROM commits WHERE id = ?", id)
	return errors.Wrap(err, "failed to delete commit")
}
