package ai

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"commit_craft_reborn/internal/git"
)

// runLinkCommit writes a git commit hash onto an existing draft/commit
// row so future `ai show --commit <hash>` lookups can recover the
// keypoints + per-stage telemetry by git hash alone. Intended to be
// called by the skill immediately after `git commit` succeeds —
// `git rev-parse HEAD` gives the hash to pass to --hash.
//
// Idempotent at the DB level. When the row already carries a hash and
// the caller passes a different one, the new hash takes effect but a
// warning goes to stderr so accidental re-links are visible.
func runLinkCommit(args []string) int {
	fs := flagSet("ai link-commit")
	id := fs.Int(
		"id",
		0,
		"Draft/commit id to link. Required.",
	)
	hash := fs.String(
		"hash",
		"",
		"Git commit hash (short or full). Required.",
	)
	workspace := fs.String(
		"workspace",
		"",
		"Repo path used to resolve the hash. Defaults to the current directory.",
	)
	kind := fs.String(
		"kind",
		"",
		"Force dispatch table when --id collides across commits/releases: 'commit' | 'release'. Optional.",
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}
	if *id <= 0 {
		printErrorJSON("invalid_input", "--id is required and must be > 0")
		return 2
	}
	if strings.TrimSpace(*hash) == "" {
		printErrorJSON("invalid_input", "--hash is required")
		return 2
	}

	boot, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer boot.db.Close()

	ws := strings.TrimSpace(*workspace)
	if ws == "" {
		ws = boot.pwd
	}

	fullHash, err := git.ResolveCommitHashAt(ws, *hash)
	if err != nil {
		printErrorJSON("invalid_input",
			fmt.Sprintf("could not resolve --hash %q in %s: %v", *hash, ws, err))
		return 2
	}

	res, err := dispatchByID(boot.db, *id, *kind)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			printErrorJSON("not_found", fmt.Sprintf("no commit or release with id=%d", *id))
			return 1
		}
		printErrorJSON("db_error", err.Error())
		return 1
	}

	var existingHash string
	switch res.Kind {
	case kindCommit:
		existingHash = res.Commit.CommitHash
	case kindRelease:
		existingHash = res.Release.CommitHash
	}
	if existingHash != "" && existingHash != fullHash {
		fmt.Fprintf(os.Stderr,
			"warning: draft %d was already linked to %s; overwriting with %s\n",
			*id, shortHash(existingHash), shortHash(fullHash))
	}

	switch res.Kind {
	case kindCommit:
		if err := boot.db.LinkCommitHash(*id, fullHash); err != nil {
			printErrorJSON("db_error", err.Error())
			return 1
		}
		saved, gerr := boot.db.GetCommitByID(*id)
		if gerr != nil {
			fmt.Fprintf(os.Stderr, "warning: post-link reload failed: %v\n", gerr)
			saved = *res.Commit
			saved.CommitHash = fullHash
		}
		cj, ferr := commitToJSON(
			saved,
			loadStagesForCommit(boot.db, saved.ID),
			boot.cfg.CommitFormat.TypeFormat,
		)
		if ferr != nil {
			printErrorJSON("format_error", ferr.Error())
			return 1
		}
		printCommitJSON(cj)
	case kindRelease:
		if err := boot.db.LinkReleaseHash(*id, fullHash); err != nil {
			printErrorJSON("db_error", err.Error())
			return 1
		}
		saved, gerr := boot.db.GetReleaseByID(*id)
		if gerr != nil {
			fmt.Fprintf(os.Stderr, "warning: post-link reload failed: %v\n", gerr)
			saved = *res.Release
			saved.CommitHash = fullHash
		}
		cj, ferr := releaseToJSON(saved, boot.cfg.CommitFormat.TypeFormat)
		if ferr != nil {
			printErrorJSON("format_error", ferr.Error())
			return 1
		}
		printCommitJSON(cj)
	}
	return 0
}

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}
