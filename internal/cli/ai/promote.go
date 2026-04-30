package ai

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"

	"commit_craft_reborn/internal/changelog"
	"commit_craft_reborn/internal/git"
)

// runPromote flips a draft to status='completed' via FinalizeCommit.
// It does not execute `git commit` — the caller is expected to take
// the printed final_message and commit it themselves (or re-run the
// TUI flow). Idempotent: promoting an already-completed row is a
// no-op that still prints the latest JSON.
func runPromote(args []string) int {
	fs := flagSet("ai promote")
	id := fs.Int("id", 0, "Draft ID to promote to completed (required)")
	noChangelogWrite := fs.Bool(
		"no-changelog-write",
		false,
		"Skip writing/staging CHANGELOG.md even when the draft has a changelog entry",
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}
	if *id <= 0 {
		printErrorJSON("invalid_input", "--id is required")
		return 2
	}

	bs, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer bs.db.Close()

	c, err := bs.db.GetCommitByID(*id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			printErrorJSON("not_found", fmt.Sprintf("commit with id=%d not found", *id))
			return 1
		}
		printErrorJSON("db_error", err.Error())
		return 1
	}
	if c.MessageEN == "" {
		printErrorJSON(
			"invalid_input",
			fmt.Sprintf(
				"draft id=%d has no final_message yet — run `ai generate` or `ai regenerate` first",
				*id,
			),
		)
		return 1
	}
	if err := bs.db.FinalizeCommit(c); err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}

	// Mirror the TUI's write-on-accept timing: the entry text was produced
	// by stage 4 and stored on the draft, but the file is only updated when
	// the draft is promoted. Re-detect the path here (workspace + config)
	// instead of persisting it — same logic the TUI uses, idempotent on
	// re-promote, and avoids a schema migration.
	if bs.cfg.Changelog.Enabled && c.IaChangelog != "" && !*noChangelogWrite {
		info, derr := changelog.Detect(c.Workspace, bs.cfg.Changelog.Path)
		if derr != nil || info == nil || info.Path == "" {
			msg := "changelog target not found"
			if derr != nil {
				msg = derr.Error()
			}
			printErrorJSON("changelog_target_missing", msg)
			return 1
		}
		if err := changelog.Prepend(info.Path, c.IaChangelog); err != nil {
			printErrorJSON("changelog_write_error", err.Error())
			return 1
		}
		if err := git.StageFile(info.Path); err != nil {
			printErrorJSON("changelog_stage_error", err.Error())
			return 1
		}
	}

	saved, err := bs.db.GetCommitByID(c.ID)
	if err != nil {
		saved = c
		saved.Status = "completed"
	}
	stages := loadStagesForCommit(bs.db, saved.ID)
	cj, err := commitToJSON(saved, stages, bs.cfg.CommitFormat.TypeFormat)
	if err != nil {
		printErrorJSON("incomplete_commit", err.Error())
		return 1
	}
	printCommitJSON(cj)
	return 0
}
