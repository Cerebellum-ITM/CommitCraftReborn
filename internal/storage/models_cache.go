package storage

import (
	"time"

	"github.com/pkg/errors"
)

// CachedModel mirrors api.GroqModel but lives in storage to avoid the
// import cycle (api → storage would loop back).
type CachedModel struct {
	ID            string
	OwnedBy       string
	ContextWindow int
}

// SaveModelsCache replaces the cached catalogue with the fresh slice.
// Wrapped in a transaction so concurrent reads always see a consistent
// snapshot.
func (db *DB) SaveModelsCache(models []CachedModel) error {
	tx, err := db.Begin()
	if err != nil {
		return errors.Wrap(err, "begin tx")
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM groq_models_cache"); err != nil {
		return errors.Wrap(err, "wipe cache")
	}

	now := time.Now().Unix()
	stmt, err := tx.Prepare(
		"INSERT INTO groq_models_cache (id, owned_by, context_window, fetched_at) VALUES (?, ?, ?, ?)",
	)
	if err != nil {
		return errors.Wrap(err, "prepare insert")
	}
	defer stmt.Close()

	for _, m := range models {
		if _, err := stmt.Exec(m.ID, m.OwnedBy, m.ContextWindow, now); err != nil {
			return errors.Wrap(err, "insert model "+m.ID)
		}
	}

	return tx.Commit()
}

// LoadModelsCache returns the cached models and the timestamp of the
// most recent fetch. fetchedAt is the zero time when the cache is empty.
func (db *DB) LoadModelsCache() ([]CachedModel, time.Time, error) {
	rows, err := db.Query(
		"SELECT id, owned_by, context_window, fetched_at FROM groq_models_cache ORDER BY id ASC",
	)
	if err != nil {
		return nil, time.Time{}, errors.Wrap(err, "query cache")
	}
	defer rows.Close()

	var (
		models  []CachedModel
		latest  int64
		hasRows bool
	)
	for rows.Next() {
		var m CachedModel
		var fetchedAt int64
		if err := rows.Scan(&m.ID, &m.OwnedBy, &m.ContextWindow, &fetchedAt); err != nil {
			return nil, time.Time{}, errors.Wrap(err, "scan cache row")
		}
		models = append(models, m)
		if fetchedAt > latest {
			latest = fetchedAt
		}
		hasRows = true
	}
	if !hasRows {
		return nil, time.Time{}, nil
	}
	return models, time.Unix(latest, 0), nil
}

// IsModelsCacheStale reports whether fetchedAt is older than ttl. A
// zero fetchedAt (no cache) is always stale.
func IsModelsCacheStale(fetchedAt time.Time, ttl time.Duration) bool {
	if fetchedAt.IsZero() {
		return true
	}
	return time.Since(fetchedAt) > ttl
}
