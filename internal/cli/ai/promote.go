package ai

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
)

// runPromote flips a draft to status='completed' via FinalizeCommit.
// It does not execute `git commit` — the caller is expected to take
// the printed final_message and commit it themselves (or re-run the
// TUI flow). Idempotent: promoting an already-completed row is a
// no-op that still prints the latest JSON.
func runPromote(args []string) int {
	fs := flagSet("ai promote")
	id := fs.Int("id", 0, "Draft ID to promote to completed (required)")
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
	saved, err := bs.db.GetCommitByID(c.ID)
	if err != nil {
		saved = c
		saved.Status = "completed"
	}
	stages := loadStagesForCommit(bs.db, saved.ID)
	printCommitJSON(commitToJSON(saved, stages))
	return 0
}
