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
		"SELECT id, type, scope, message_es, message_en, workspace, diff_code, status, ia_summary, ia_commit_raw, ia_title, ia_changelog, created_at FROM commits WHERE workspace = ? AND status = ? ORDER BY created_at DESC",
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
		if err := rows.Scan(&c.ID, &c.Type, &c.Scope, &messageES, &c.MessageEN, &c.Workspace, &c.Diff_code, &c.Status, &c.IaSummary, &c.IaCommitRaw, &c.IaTitle, &c.IaChangelog, &createdAt); err != nil {
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

// GetCommitByID returns the commit row matching id, with key points
// decoded back into a slice. Returns sql.ErrNoRows wrapped when the id
// doesn't exist so callers can branch on errors.Is(err, sql.ErrNoRows).
func (db *DB) GetCommitByID(id int) (Commit, error) {
	row := db.QueryRow(
		"SELECT id, type, scope, message_es, message_en, workspace, diff_code, status, ia_summary, ia_commit_raw, ia_title, ia_changelog, created_at FROM commits WHERE id = ?",
		id,
	)
	var c Commit
	var createdAt, messageES string
	if err := row.Scan(
		&c.ID, &c.Type, &c.Scope, &messageES, &c.MessageEN, &c.Workspace,
		&c.Diff_code, &c.Status, &c.IaSummary, &c.IaCommitRaw, &c.IaTitle, &c.IaChangelog,
		&createdAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c, errors.Wrap(err, "commit not found")
		}
		return c, errors.Wrap(err, "failed to scan commit row")
	}
	c.KeyPoints = splitKeyPoints(messageES)
	t, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return c, errors.Wrap(err, "failed to parse created_at: "+createdAt)
	}
	c.CreatedAt = t.Local()
	return c, nil
}

// CreateCommit adds a new commit to the database and writes the new row
// id back into c.ID so callers can persist child rows (e.g. ai_calls)
// using the freshly minted commit id.
func (db *DB) CreateCommit(c *Commit) error {
	createdAt := time.Now().UTC().Format(time.RFC3339)

	res, err := db.Exec(
		"INSERT INTO commits (type, scope, message_es, message_en, workspace, diff_code, status, ia_summary, ia_commit_raw, ia_title, ia_changelog, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
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
		c.IaChangelog,
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
			"INSERT INTO commits (type, scope, message_es, message_en, workspace, diff_code, status, ia_summary, ia_commit_raw, ia_title, ia_changelog, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
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
			c.IaChangelog,
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
		"UPDATE commits SET type = ?, scope = ?, message_es = ?, message_en = ?, diff_code = ?, ia_summary = ?, ia_commit_raw = ?, ia_title = ?, ia_changelog = ? WHERE id = ?",
		c.Type,
		c.Scope,
		joinKeyPoints(c.KeyPoints),
		c.MessageEN,
		c.Diff_code,
		c.IaSummary,
		c.IaCommitRaw,
		c.IaTitle,
		c.IaChangelog,
		c.ID,
	)
	return errors.Wrap(err, "failed to update draft commit")
}

// CreateAICall inserts a single per-stage telemetry record. Returns the
// new row id so callers can correlate further updates if ever needed.
func (db *DB) CreateAICall(call AICall) (int64, error) {
	createdAt := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		"INSERT INTO ai_calls (commit_id, stage, model, prompt_tokens, completion_tokens, total_tokens, queue_time_ms, prompt_time_ms, completion_time_ms, total_time_ms, request_id, tpm_limit_at_call, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
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
		call.TPMLimitAtCall,
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
		"SELECT id, commit_id, stage, model, prompt_tokens, completion_tokens, total_tokens, queue_time_ms, prompt_time_ms, completion_time_ms, total_time_ms, request_id, tpm_limit_at_call, created_at FROM ai_calls WHERE commit_id = ? ORDER BY id ASC",
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
			&c.RequestID, &c.TPMLimitAtCall, &createdAt,
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

// CreateReleaseAICall inserts one per-stage telemetry record into the
// release_ai_calls table. The AICall struct is reused; its CommitID
// field carries the owning release id for release rows.
func (db *DB) CreateReleaseAICall(call AICall) (int64, error) {
	createdAt := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		"INSERT INTO release_ai_calls (release_id, stage, model, prompt_tokens, completion_tokens, total_tokens, queue_time_ms, prompt_time_ms, completion_time_ms, total_time_ms, request_id, tpm_limit_at_call, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
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
		call.TPMLimitAtCall,
		createdAt,
	)
	if err != nil {
		return 0, errors.Wrap(err, "failed to insert release_ai_call")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, errors.Wrap(err, "failed to retrieve last insert id for release_ai_call")
	}
	return id, nil
}

// GetAICallsByReleaseID returns the release_ai_calls rows linked to
// releaseID, in insertion order. Empty slice + nil error when none.
func (db *DB) GetAICallsByReleaseID(releaseID int) ([]AICall, error) {
	rows, err := db.Query(
		"SELECT id, release_id, stage, model, prompt_tokens, completion_tokens, total_tokens, queue_time_ms, prompt_time_ms, completion_time_ms, total_time_ms, request_id, tpm_limit_at_call, created_at FROM release_ai_calls WHERE release_id = ? ORDER BY id ASC",
		releaseID,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query release_ai_calls")
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
			&c.RequestID, &c.TPMLimitAtCall, &createdAt,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan release_ai_call row")
		}
		t, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse release_ai_call created_at: "+createdAt)
		}
		c.CreatedAt = t.Local()
		calls = append(calls, c)
	}
	return calls, nil
}

// DeleteAICallsByReleaseID purges every release_ai_call linked to
// releaseID. Used when re-inserting fresh stats so the table doesn't
// grow orphan rows when the user re-runs the release pipeline.
func (db *DB) DeleteAICallsByReleaseID(releaseID int) error {
	_, err := db.Exec("DELETE FROM release_ai_calls WHERE release_id = ?", releaseID)
	return errors.Wrap(err, "failed to delete release_ai_calls")
}

// SaveModelRateLimits UPSERTs the latest rate-limit snapshot for one
// model. captured_at is overwritten on every call so freshness checks
// at render time can decide when the bucket has refilled.
func (db *DB) SaveModelRateLimits(rl ModelRateLimits) error {
	if rl.ModelID == "" {
		return nil
	}
	capturedAt := rl.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now()
	}
	_, err := db.Exec(
		"INSERT INTO model_rate_limits (model_id, limit_requests, remaining_requests, reset_requests_ms, limit_tokens, remaining_tokens, reset_tokens_ms, captured_at, requests_parsed, tokens_parsed, requests_today, requests_day) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(model_id) DO UPDATE SET limit_requests=excluded.limit_requests, remaining_requests=excluded.remaining_requests, reset_requests_ms=excluded.reset_requests_ms, limit_tokens=excluded.limit_tokens, remaining_tokens=excluded.remaining_tokens, reset_tokens_ms=excluded.reset_tokens_ms, captured_at=excluded.captured_at, requests_parsed=excluded.requests_parsed, tokens_parsed=excluded.tokens_parsed, requests_today=excluded.requests_today, requests_day=excluded.requests_day",
		rl.ModelID,
		rl.LimitRequests,
		rl.RemainingRequests,
		rl.ResetRequestsMs,
		rl.LimitTokens,
		rl.RemainingTokens,
		rl.ResetTokensMs,
		capturedAt.UTC().Format(time.RFC3339),
		boolToInt(rl.RequestsParsed),
		boolToInt(rl.TokensParsed),
		rl.RequestsToday,
		rl.RequestsDay,
	)
	return errors.Wrap(err, "failed to upsert model_rate_limits")
}

// boolToInt maps a Go bool to the 0/1 SQLite integer convention used by
// the rate-limit table's parsed flags.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// LoadAllModelRateLimits returns every persisted rate-limit row so the
// in-memory cache can be hydrated at startup.
func (db *DB) LoadAllModelRateLimits() ([]ModelRateLimits, error) {
	rows, err := db.Query(
		"SELECT model_id, limit_requests, remaining_requests, reset_requests_ms, limit_tokens, remaining_tokens, reset_tokens_ms, captured_at, requests_parsed, tokens_parsed, requests_today, requests_day FROM model_rate_limits",
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query model_rate_limits")
	}
	defer rows.Close()

	var out []ModelRateLimits
	for rows.Next() {
		var r ModelRateLimits
		var capturedAt string
		var requestsParsed, tokensParsed int
		if err := rows.Scan(
			&r.ModelID,
			&r.LimitRequests, &r.RemainingRequests, &r.ResetRequestsMs,
			&r.LimitTokens, &r.RemainingTokens, &r.ResetTokensMs,
			&capturedAt,
			&requestsParsed, &tokensParsed,
			&r.RequestsToday, &r.RequestsDay,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan model_rate_limits row")
		}
		t, err := time.Parse(time.RFC3339, capturedAt)
		if err != nil {
			return nil, errors.Wrap(
				err,
				"failed to parse model_rate_limits captured_at: "+capturedAt,
			)
		}
		r.CapturedAt = t
		r.RequestsParsed = requestsParsed != 0
		r.TokensParsed = tokensParsed != 0
		out = append(out, r)
	}
	return out, nil
}

// FinalizeCommit updates a commit to set its status to 'completed' and saves final data.
func (db *DB) FinalizeCommit(c Commit) error {
	_, err := db.Exec(
		"UPDATE commits SET type = ?, scope = ?, message_es = ?, message_en = ?, diff_code = ?, ia_summary = ?, ia_commit_raw = ?, ia_title = ?, ia_changelog = ?, status = 'completed' WHERE id = ?",
		c.Type,
		c.Scope,
		joinKeyPoints(c.KeyPoints),
		c.MessageEN,
		c.Diff_code,
		c.IaSummary,
		c.IaCommitRaw,
		c.IaTitle,
		c.IaChangelog,
		c.ID,
	)
	return errors.Wrap(err, "failed to finalize commit")
}
