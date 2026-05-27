package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/commit"
)

// runVerify runs the deterministic rule set in
// aiengine.VerifyFinalMessage against a stored draft (or completed
// commit) and prints the report as JSON on stdout. The exit code is
// the actionable signal for agents: 0 when clean (or warnings only,
// without --strict-warnings), 4 when at least one error finding is
// present (or any finding under --strict-warnings).
//
// Exit code 3 is intentionally NOT reused here — it belongs to
// `ai context --strict`, and we want the two gates to be
// distinguishable by exit code alone.
func runVerify(args []string) int {
	fs := flagSet("ai verify")
	id := fs.Int(
		"id",
		0,
		"Draft or commit id to verify (required).",
	)
	strictWarnings := fs.Bool(
		"strict-warnings",
		false,
		"Treat warnings as errors for exit-code purposes. JSON output is unchanged.",
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

	boot, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer boot.db.Close()

	c, err := boot.db.GetCommitByID(*id)
	if err != nil {
		printErrorJSON("not_found", fmt.Sprintf("draft id=%d: %s", *id, err.Error()))
		return 1
	}

	final, err := commit.FormatFinalMessage(
		boot.cfg.CommitFormat.TypeFormat,
		c.Type,
		c.Scope,
		c.MessageEN,
	)
	if err != nil {
		printErrorJSON("incomplete_draft", err.Error())
		return 1
	}

	report := aiengine.VerifyFinalMessage(final)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)

	if report.HasErrors {
		return 4
	}
	if *strictWarnings && report.HasWarnings {
		return 4
	}
	return 0
}
