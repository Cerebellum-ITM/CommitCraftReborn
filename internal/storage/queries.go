package storage

import (
	"database/sql"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func joinKeyPoints(kp []string) string {
	return strings.Join(kp, "\n")
}

func splitKeyPoints(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// GetCommits retrieves commits from the database based on a status.
func (db *DB) GetCommits(pwd string, status string) ([]Commit, error) {
	rows, err := db.Query(
		"SELECT id, type, scope, message_es, message_en, workspace, diff_code, status, ia_summary, ia_commit_raw, ia_title, created_at FROM commits WHERE workspace = ? AND status = ? ORDER BY created_at DESC",
		pwd,
		status,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query commits")
	}
	defer rows.Close()

	var commits []Commit
	for rows.Next() {
		var c Commit
		var createdAt, messageES string
		if err := rows.Scan(&c.ID, &c.Type, &c.Scope, &messageES, &c.MessageEN, &c.Workspace, &c.Diff_code, &c.Status, &c.IaSummary, &c.IaCommitRaw, &c.IaTitle, &createdAt); err != nil {
			return nil, errors.Wrap(err, "failed to scan commit row")
		}
		c.KeyPoints = splitKeyPoints(messageES)

		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse created_at: "+createdAt)
		}
		c.CreatedAt = t.Local() // Convert to local time for display
		commits = append(commits, c)
	}
	return commits, nil
}

// CreateCommit adds a new commit to the database and writes the new row
// id back into c.ID so callers can persist child rows (e.g. ai_calls)
// using the freshly minted commit id.
func (db *DB) CreateCommit(c *Commit) error {
	createdAt := time.Now().UTC().Format(time.RFC3339)

	res, err := db.Exec(
		"INSERT INTO commits (type, scope, message_es, message_en, workspace, diff_code, status, ia_summary, ia_commit_raw, ia_title, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		c.Type,
		c.Scope,
		joinKeyPoints(c.KeyPoints),
		c.MessageEN,
		c.Workspace,
		c.Diff_code,
		"completed",
		c.IaSummary,
		c.IaCommitRaw,
		c.IaTitle,
		createdAt,
	)
	if err != nil {
		return errors.Wrap(err, "failed to insert commit")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return errors.Wrap(err, "failed to retrieve last insert id for commit")
	}
	c.ID = int(id)
	return nil
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

func (db *DB) GetLatestRelease(pwd string) (Release, error) {
	row := db.QueryRow(
		"SELECT id, type, title, body, branch, commit_list, version, workspace, created_at FROM releases WHERE workspace = ? ORDER BY created_at DESC LIMIT 1",
		pwd,
	)

	r := Release{}
	var createdAt string
	err := row.Scan(
		&r.ID,
		&r.Type,
		&r.Title,
		&r.Body,
		&r.Branch,
		&r.CommitList,
		&r.Version,
		&r.Workspace,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return r, nil // No releases found for this workspace
		}
		return r, errors.Wrap(err, "failed to scan Release row")
	}

	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return r, errors.Wrap(err, "failed to parse created_at: "+createdAt)
	}
	r.CreatedAt = t.Local() // Convert to local time for display

	return r, nil
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

// SaveDraft saves a commit as a draft. It will insert a new record if the ID is 0,
// otherwise it will update the existing record.
func (db *DB) SaveDraft(c *Commit) error {
	// If ID is 0, it's a new draft, so we INSERT.
	if c.ID == 0 {
		createdAt := time.Now().UTC().Format(time.RFC3339)
		c.Status = "draft"

		res, err := db.Exec(
			"INSERT INTO commits (type, scope, message_es, message_en, workspace, diff_code, status, ia_summary, ia_commit_raw, ia_title, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			c.Type,
			c.Scope,
			joinKeyPoints(c.KeyPoints),
			c.MessageEN,
			c.Workspace,
			c.Diff_code,
			c.Status,
			c.IaSummary,
			c.IaCommitRaw,
			c.IaTitle,
			createdAt,
		)
		if err != nil {
			return errors.Wrap(err, "failed to insert new draft commit")
		}
		id, err := res.LastInsertId()
		if err != nil {
			return errors.Wrap(err, "failed to retrieve last insert ID for draft")
		}
		c.ID = int(id)
		return nil
	}

	// If ID is not 0, it's an existing draft, so we UPDATE.
	_, err := db.Exec(
		"UPDATE commits SET type = ?, scope = ?, message_es = ?, message_en = ?, diff_code = ?, ia_summary = ?, ia_commit_raw = ?, ia_title = ? WHERE id = ?",
		c.Type,
		c.Scope,
		joinKeyPoints(c.KeyPoints),
		c.MessageEN,
		c.Diff_code,
		c.IaSummary,
		c.IaCommitRaw,
		c.IaTitle,
		c.ID,
	)
	return errors.Wrap(err, "failed to update draft commit")
}

// CreateAICall inserts a single per-stage telemetry record. Returns the
// new row id so callers can correlate further updates if ever needed.
func (db *DB) CreateAICall(call AICall) (int64, error) {
	createdAt := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		"INSERT INTO ai_calls (commit_id, stage, model, prompt_tokens, completion_tokens, total_tokens, queue_time_ms, prompt_time_ms, completion_time_ms, total_time_ms, request_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		call.CommitID,
		call.Stage,
		call.Model,
		call.PromptTokens,
		call.CompletionTokens,
		call.TotalTokens,
		call.QueueTimeMs,
		call.PromptTimeMs,
		call.CompletionTimeMs,
		call.TotalTimeMs,
		call.RequestID,
		createdAt,
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert ai_call")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "failed to retrieve last insert id for ai_call")
	}
	return id, nil
}

// GetAICallsByCommitID returns the ai_calls rows linked to commitID, in
// insertion order. Empty slice + nil error when the commit has no calls.
func (db *DB) GetAICallsByCommitID(commitID int) ([]AICall, error) {
	rows, err := db.Query(
		"SELECT id, commit_id, stage, model, prompt_tokens, completion_tokens, total_tokens, queue_time_ms, prompt_time_ms, completion_time_ms, total_time_ms, request_id, created_at FROM ai_calls WHERE commit_id = ? ORDER BY id ASC",
		commitID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query ai_calls")
	}
	defer rows.Close()

	var calls []AICall
	for rows.Next() {
		var c AICall
		var createdAt string
		if err := rows.Scan(
			&c.ID, &c.CommitID, &c.Stage, &c.Model,
			&c.PromptTokens, &c.CompletionTokens, &c.TotalTokens,
			&c.QueueTimeMs, &c.PromptTimeMs, &c.CompletionTimeMs, &c.TotalTimeMs,
			&c.RequestID, &createdAt,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan ai_call row")
		}
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse ai_call created_at: "+createdAt)
		}
		c.CreatedAt = t.Local()
		calls = append(calls, c)
	}
	return calls, nil
}

// DeleteAICallsByCommitID purges every ai_call linked to commitID. Used
// before re-inserting fresh stats when a draft is saved repeatedly so the
// telemetry table never grows orphan rows for an evolving draft.
func (db *DB) DeleteAICallsByCommitID(commitID int) error {
	_, err := db.Exec("DELETE FROM ai_calls WHERE commit_id = ?", commitID)
	return errors.Wrap(err, "failed to delete ai_calls")
}

// FinalizeCommit updates a commit to set its status to 'completed' and saves final data.
func (db *DB) FinalizeCommit(c Commit) error {
	_, err := db.Exec(
		"UPDATE commits SET type = ?, scope = ?, message_es = ?, message_en = ?, diff_code = ?, ia_summary = ?, ia_commit_raw = ?, ia_title = ?, status = 'completed' WHERE id = ?",
		c.Type,
		c.Scope,
		joinKeyPoints(c.KeyPoints),
		c.MessageEN,
		c.Diff_code,
		c.IaSummary,
		c.IaCommitRaw,
		c.IaTitle,
		c.ID,
	)
	return errors.Wrap(err, "failed to finalize commit")
}
