package ai

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
)

// runShow prints the full JSON for a single draft/commit, including
// per-stage telemetry rebuilt from the ai_calls table.
func runShow(args []string) int {
	fs := flagSet("ai show")
	id := fs.Int("id", 0, "Commit/draft ID to display (required)")
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
	stages := loadStagesForCommit(bs.db, c.ID)
	cj, err := commitToJSON(c, stages, bs.cfg.CommitFormat.TypeFormat)
	if err != nil {
		printErrorJSON("incomplete_commit", err.Error())
		return 1
	}
	printCommitJSON(cj)
	return 0
}
