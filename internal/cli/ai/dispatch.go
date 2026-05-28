package ai

import (
	"database/sql"
	"errors"

	"commit_craft_reborn/internal/storage"
)

// kind values used by the JSON envelope's `kind` field and the
// internal dispatch routing.
const (
	kindCommit  = "commit"
	kindRelease = "release"
)

// dispatchResult is the union returned by dispatchByID. Exactly one of
// Commit / Release is non-nil when Kind is non-empty.
type dispatchResult struct {
	Kind    string
	Commit  *storage.Commit
	Release *storage.Release
}

// dispatchByID looks up a row by id, with an optional kind hint to
// disambiguate id collisions between the `commits` and `releases`
// tables. When kindHint is empty, the function probes `commits`
// first and falls back to `releases` on miss — fine for the common
// case where the agent operates on a freshly created row and no
// collision exists.
//
// When kindHint is "commit" or "release", the lookup is restricted
// to that table. Agents that persist the (id, kind) tuple from the
// JSON envelope (every `ai release` / `ai merge` response includes
// `"kind": "release"`) should pass the kind through to subsequent
// `ai show / edit / verify / promote / link-commit` calls — it is
// the only collision-safe path.
//
// Returns sql.ErrNoRows wrapped when no row matches.
func dispatchByID(db *storage.DB, id int, kindHint string) (dispatchResult, error) {
	if db == nil {
		return dispatchResult{}, errors.New("nil db")
	}
	if id <= 0 {
		return dispatchResult{}, errors.New("id must be > 0")
	}

	switch kindHint {
	case kindCommit:
		c, err := db.GetCommitByID(id)
		if err != nil {
			return dispatchResult{}, err
		}
		return dispatchResult{Kind: kindCommit, Commit: &c}, nil
	case kindRelease:
		r, err := db.GetReleaseByID(id)
		if err != nil {
			return dispatchResult{}, err
		}
		return dispatchResult{Kind: kindRelease, Release: &r}, nil
	case "":
		// fall through to auto-probe
	default:
		return dispatchResult{}, errors.New("invalid --kind (must be commit or release)")
	}

	c, err := db.GetCommitByID(id)
	if err == nil {
		return dispatchResult{Kind: kindCommit, Commit: &c}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return dispatchResult{}, err
	}
	r, rerr := db.GetReleaseByID(id)
	if rerr == nil {
		return dispatchResult{Kind: kindRelease, Release: &r}, nil
	}
	if errors.Is(rerr, sql.ErrNoRows) {
		return dispatchResult{}, rerr
	}
	return dispatchResult{}, rerr
}
