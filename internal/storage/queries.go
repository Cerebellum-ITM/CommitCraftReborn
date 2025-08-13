package storage

import (
	"time"

	"github.com/pkg/errors"
)

// GetCommits retrieves all commits from the database.
func (db *DB) GetCommits() ([]Commit, error) {
	rows, err := db.Query(
		"SELECT id, type, scope, message_es, message_en, workspace, created_at FROM commits ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query commits")
	}
	defer rows.Close()

	var commits []Commit
	for rows.Next() {
		var c Commit
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Type, &c.Scope, &c.MessageES, &c.MessageEN, &c.Workspace, &createdAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan commit row")
		}
		
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse created_at: "+createdAt)
		}
		c.CreatedAt = t.Local() // Convert to local time for display
		commits = append(commits, c)
	}
	return commits, nil
}

// CreateCommit adds a new commit to the database.
func (db *DB) CreateCommit(c Commit) error {
	// Note: Actual translation logic would go elsewhere.
	messageEN := c.MessageES // Placeholder for translation

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
