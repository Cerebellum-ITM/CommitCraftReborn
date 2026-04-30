package ai

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"commit_craft_reborn/internal/aiengine"
)

// runRegenerate re-runs the pipeline against an existing draft. By
// default it reuses the stored keypoints/tag/scope and the diff
// snapshot captured at generate time, so the AI agent can iterate on
// the message without the working tree's staged set drifting between
// runs. Any of --keypoint/--tag/--scope override the stored values
// before the pipeline re-runs.
func runRegenerate(args []string) int {
	fs := flagSet("ai regenerate")
	id := fs.Int("id", 0, "Existing commit/draft ID to regenerate (required)")
	var keypoints stringSlice
	var scopes stringSlice
	tag := fs.String("tag", "", "Override the stored commit type tag")
	fs.Var(&keypoints, "keypoint", "Override keypoints (repeatable)")
	fs.Var(&keypoints, "k", "Shorthand for --keypoint")
	fs.Var(&scopes, "scope", "Override scopes (repeatable)")
	fs.Var(&scopes, "s", "Shorthand for --scope")
	fs.StringVar(tag, "t", *tag, "Shorthand for --tag")
	noChangelog := fs.Bool(
		"no-changelog",
		false,
		"Skip the changelog refiner stage even when enabled in config",
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

	// Apply overrides only when the caller passed them. Empty slices /
	// empty strings mean "keep stored value".
	if *tag != "" {
		if !tagIsKnown(*tag, bs.finalCommitTypes) {
			printErrorJSON("invalid_input", fmt.Sprintf("unknown tag %q", *tag))
			return 2
		}
		c.Type = *tag
	}
	if len(keypoints) > 0 {
		c.KeyPoints = keypoints
	}
	if len(scopes) > 0 {
		c.Scope = strings.Join(scopes, "\n")
	}
	if c.Diff_code == "" {
		printErrorJSON("invalid_input",
			"draft has no stored diff to regenerate against; create a new draft via `ai generate`")
		return 1
	}

	in := aiengine.Input{
		KeyPoints:       c.KeyPoints,
		Type:            c.Type,
		Scope:           c.Scope,
		Diff:            c.Diff_code,
		ChangelogActive: !*noChangelog && bs.cfg.Changelog.Enabled,
	}
	out, err := aiengine.Run(aiengine.Deps{
		Cfg: bs.cfg, DB: bs.db, Log: bs.log, Pwd: bs.pwd,
	}, in)
	if err != nil {
		printErrorJSON("api_error", err.Error())
		return 1
	}

	c.IaSummary = out.Summary
	c.IaCommitRaw = out.Body
	c.IaTitle = out.Title
	c.IaChangelog = out.ChangelogEntry
	c.MessageEN = out.FinalMessage
	if err := bs.db.SaveDraft(&c); err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}
	if err := persistAICalls(bs.db, c.ID, out.Stages); err != nil {
		fmt.Fprintf(os.Stderr, "warning: ai_calls persistence failed: %v\n", err)
	}

	saved, err := bs.db.GetCommitByID(c.ID)
	if err != nil {
		saved = c
	}
	printCommitJSON(commitToJSON(saved, out.Stages))
	return 0
}
