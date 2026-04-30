package ai

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/storage"
)

// runGenerate parses flags, validates inputs, runs the engine, persists
// the draft, prints JSON, and returns the exit code. It does not
// execute `git commit` — promoting a draft is a separate concern.
func runGenerate(args []string) int {
	fs := flagSet("ai generate")
	var keypoints stringSlice
	var scopes stringSlice
	tag := fs.String("tag", "", "Commit type tag (validated against the resolved type list)")
	fs.Var(&keypoints, "keypoint", "Add a keypoint (repeatable)")
	fs.Var(&keypoints, "k", "Shorthand for --keypoint")
	fs.Var(&scopes, "scope", "Add a scope (repeatable, joined with newlines for the AI)")
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

	if len(keypoints) == 0 {
		printErrorJSON("invalid_input", "at least one --keypoint is required")
		return 2
	}
	if *tag == "" {
		printErrorJSON("invalid_input", "--tag is required")
		return 2
	}
	if len(scopes) == 0 {
		printErrorJSON("invalid_input", "at least one --scope is required")
		return 2
	}

	bs, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer bs.db.Close()

	if !tagIsKnown(*tag, bs.finalCommitTypes) {
		printErrorJSON("invalid_input",
			fmt.Sprintf("unknown tag %q — run `commitcraft ai list-tags` to see valid tags", *tag))
		return 2
	}

	diff, err := validateAndStageDiff(bs.cfg.Prompts.ChangeAnalyzerMaxDiffSize)
	if err != nil {
		printErrorJSON("no_staged_diff", err.Error())
		return 1
	}

	scope := strings.Join(scopes, "\n")
	in := aiengine.Input{
		KeyPoints:       keypoints,
		Type:            *tag,
		Scope:           scope,
		Diff:            diff,
		ChangelogActive: !*noChangelog && bs.cfg.Changelog.Enabled,
	}
	out, err := aiengine.Run(aiengine.Deps{
		Cfg: bs.cfg, DB: bs.db, Log: bs.log, Pwd: bs.pwd,
	}, in)
	if err != nil {
		printErrorJSON("api_error", err.Error())
		return 1
	}

	c := storage.Commit{
		Type:        *tag,
		Scope:       scope,
		KeyPoints:   keypoints,
		Workspace:   bs.pwd,
		Diff_code:   diff,
		IaSummary:   out.Summary,
		IaCommitRaw: out.Body,
		IaTitle:     out.Title,
		IaChangelog: out.ChangelogEntry,
		MessageEN:   out.FinalMessage,
	}
	if err := bs.db.SaveDraft(&c); err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}
	if err := persistAICalls(bs.db, c.ID, out.Stages); err != nil {
		// Telemetry persistence is best-effort; surface as a warning on
		// stderr but don't fail the run — the draft is already saved.
		fmt.Fprintf(os.Stderr, "warning: ai_calls persistence failed: %v\n", err)
	}

	// Reload to get the canonical CreatedAt + Status set by SaveDraft.
	saved, err := bs.db.GetCommitByID(c.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: post-save reload failed: %v\n", err)
		saved = c
		saved.Status = "draft"
	}
	cj, err := commitToJSON(saved, out.Stages, bs.cfg.CommitFormat.TypeFormat)
	if err != nil {
		printErrorJSON("incomplete_commit", err.Error())
		return 1
	}
	printCommitJSON(cj)
	return 0
}
