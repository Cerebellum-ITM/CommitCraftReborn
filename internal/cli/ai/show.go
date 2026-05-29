package ai

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"strings"
)

// runShow prints the full JSON for a single draft/commit, including
// per-stage telemetry rebuilt from the ai_calls table. Accepts either
// --id (the internal draft id) or --commit (a git commit hash prefix
// that was previously associated via `ai link-commit`).
func runShow(args []string) int {
	fs := flagSet("ai show")
	id := fs.Int(
		"id",
		0,
		"Commit/draft id to display. Mutually exclusive with --commit.",
	)
	hash := fs.String(
		"commit",
		"",
		"Git commit hash (short prefix or full) previously linked via `ai link-commit`. Mutually exclusive with --id.",
	)
	kind := fs.String(
		"kind",
		"",
		"Force dispatch table when --id collides across commits/releases: 'commit' | 'release'. Optional; auto-probes when empty.",
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}

	hashPrefix := strings.TrimSpace(*hash)
	switch {
	case *id <= 0 && hashPrefix == "":
		printErrorJSON("invalid_input", "--id or --commit is required")
		return 2
	case *id > 0 && hashPrefix != "":
		printErrorJSON("invalid_input", "--id and --commit are mutually exclusive")
		return 2
	}

	bs, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer bs.db.Close()

	if hashPrefix != "" {
		matches, err := bs.db.GetCommitsByHashPrefix(hashPrefix)
		if err != nil {
			printErrorJSON("invalid_input", err.Error())
			return 2
		}
		switch len(matches) {
		case 0:
			printErrorJSON("not_found",
				fmt.Sprintf("no linked commit matches hash prefix %q", hashPrefix))
			return 1
		case 1:
			c := matches[0]
			cj, err := commitToJSON(c, loadStagesForCommit(bs.db, c.ID),
				bs.cfg.CommitFormat.TypeFormat)
			if err != nil {
				printErrorJSON("incomplete_commit", err.Error())
				return 1
			}
			printCommitJSON(cj)
			return 0
		default:
			ids := make([]int, len(matches))
			hashes := make([]string, len(matches))
			for i, m := range matches {
				ids[i] = m.ID
				hashes[i] = shortHash(m.CommitHash)
			}
			printErrorJSON(
				"ambiguous_hash",
				fmt.Sprintf(
					"hash prefix %q matches %d rows (ids=%v hashes=%v) — pass a longer prefix",
					hashPrefix,
					len(matches),
					ids,
					hashes,
				),
			)
			return 1
		}
	}

	res, err := dispatchByID(bs.db, *id, *kind)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			printErrorJSON("not_found",
				fmt.Sprintf("no commit or release with id=%d", *id))
			return 1
		}
		printErrorJSON("db_error", err.Error())
		return 1
	}

	switch res.Kind {
	case kindCommit:
		c := *res.Commit
		cj, err := commitToJSON(c, loadStagesForCommit(bs.db, c.ID),
			bs.cfg.CommitFormat.TypeFormat)
		if err != nil {
			printErrorJSON("incomplete_commit", err.Error())
			return 1
		}
		printCommitJSON(cj)
	case kindRelease:
		cj, err := releaseToJSON(*res.Release, bs.cfg.CommitFormat.TypeFormat)
		if err != nil {
			printErrorJSON("incomplete_release", err.Error())
			return 1
		}
		printCommitJSON(cj)
	default:
		printErrorJSON("not_found",
			fmt.Sprintf("no commit or release with id=%d", *id))
		return 1
	}
	return 0
}
