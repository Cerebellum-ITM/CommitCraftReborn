package storage

import (
	"time"

	"github.com/pkg/errors"
)

// GetCommits retrieves all commits from the database.
func (db *DB) GetCommits(pwd string) ([]Commit, error) {
	rows, err := db.Query(
		"SELECT id, type, scope, message_es, message_en, workspace, diff_code, created_at FROM commits WHERE workspace = ? ORDER BY created_at DESC",
		pwd,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query commits")
	}
	defer rows.Close()

	var commits []Commit
	for rows.Next() {
		var c Commit
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Type, &c.Scope, &c.MessageES, &c.MessageEN, &c.Workspace, &c.Diff_code, &createdAt); err != nil {
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
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(
		"INSERT INTO commits (type, scope, message_es, message_en, workspace, diff_code, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		c.Type,
		c.Scope,
		c.MessageES,
		c.MessageEN,
		c.Workspace,
		c.Diff_code,
		createdAt,
	)
	return errors.Wrap(err, "failed to insert commit")
}

func (db *DB) CreateRelease(c Release) error {
	createdAt := time.Now().UTC().Format(time.RFC3339)

	_, err := db.Exec(
		"INSERT INTO releases (type, title, body,  branch, commit_list, version, workspace, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		c.Type,
		c.Title,
		c.Body,
		c.Branch,
		c.CommitList,
		c.Version,
		c.Workspace,
		createdAt,
	)
	return errors.Wrap(err, "failed to insert commit")
}

func (db *DB) GetReleases(pwd string) ([]Release, error) {
	rows, err := db.Query(
		"SELECT id, type, title, body, branch, commit_list, version, workspace, created_at FROM releases WHERE workspace = ? ORDER BY created_at DESC",
		pwd,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query releases")
	}
	defer rows.Close()

	var releases []Release
	for rows.Next() {
		var r Release
		var createdAt string
		if err := rows.Scan(&r.ID, &r.Type, &r.Title, &r.Body, &r.Branch, &r.CommitList, &r.Version, &r.Workspace, &createdAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan Release row")
		}

		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse created_at: "+createdAt)
		}
		r.CreatedAt = t.Local() // Convert to local time for display
		releases = append(releases, r)
	}
	return releases, nil
}

// DeleteCommit removes a commit from the database by its ID.
func (db *DB) DeleteCommit(id int) error {
	_, err := db.Exec("DELETE FROM commits WHERE id = ?", id)
	return errors.Wrap(err, "failed to delete commit")
}

func (db *DB) DeleteRelease(id int) error {
	_, err := db.Exec("DELETE FROM releases WHERE id = ?", id)
	return errors.Wrap(err, "failed to delete release")
}
