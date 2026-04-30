package ai

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/git"
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
	stage := fs.String(
		"stage",
		"",
		"Re-run only one stage: body | title | changelog. "+
			"Empty (default) re-runs the full pipeline. "+
			"`body` re-runs body+title+changelog; `title` re-runs title+changelog; "+
			"`changelog` re-runs only the refiner.",
	)
	refreshDiff := fs.Bool(
		"refresh-diff",
		false,
		"Re-read `git diff --cached` from the commit's workspace and persist "+
			"the new snapshot before regenerating. Use when more files were "+
			"staged after the initial `generate`. Without this flag the stored "+
			"diff snapshot is reused so iteration stays cheap.",
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
	if *refreshDiff && *stage != "" {
		printErrorJSON("invalid_input",
			"--refresh-diff requires a full regenerate (the diff only feeds stage 1); drop --stage")
		return 2
	}
	if *refreshDiff {
		fresh, derr := git.GetStagedDiffSummaryAt(
			c.Workspace,
			bs.cfg.Prompts.ChangeAnalyzerMaxDiffSize,
		)
		if derr != nil {
			printErrorJSON("api_error", fmt.Sprintf("refresh-diff: %s", derr.Error()))
			return 1
		}
		if strings.TrimSpace(fresh) == "" {
			printErrorJSON(
				"no_staged_diff",
				fmt.Sprintf(
					"--refresh-diff: no staged changes in %s — run `git add` there first",
					c.Workspace,
				),
			)
			return 1
		}
		c.Diff_code = fresh
	}
	if c.Diff_code == "" {
		printErrorJSON("invalid_input",
			"draft has no stored diff to regenerate against; create a new draft via `ai generate`")
		return 1
	}

	// Use the commit's stored Workspace as the engine's Pwd so the
	// changelog refiner targets the repo that owns the commit, not
	// whatever cwd the caller happens to be in. Falls back to bs.pwd
	// for legacy rows without a stored workspace.
	pwd := c.Workspace
	if pwd == "" {
		pwd = bs.pwd
	}
	deps := aiengine.Deps{Cfg: bs.cfg, DB: bs.db, Log: bs.log, Pwd: pwd}
	changelogActive := !*noChangelog && bs.cfg.Changelog.Enabled

	var out aiengine.Output
	switch *stage {
	case "":
		// Full re-run. Behaves exactly like the old default path.
		in := aiengine.Input{
			KeyPoints:       c.KeyPoints,
			Type:            c.Type,
			Scope:           c.Scope,
			Diff:            c.Diff_code,
			ChangelogActive: changelogActive,
		}
		out, err = aiengine.Run(deps, in)
		if err != nil {
			printErrorJSON("api_error", err.Error())
			return 1
		}
	case "body", "title", "changelog":
		out, err = runStagePartial(deps, c, *stage, changelogActive)
		if err != nil {
			printErrorJSON("api_error", err.Error())
			return 1
		}
	default:
		printErrorJSON("invalid_input",
			fmt.Sprintf("--stage must be one of body|title|changelog (got %q)", *stage))
		return 2
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
	cj, err := commitToJSON(saved, out.Stages, bs.cfg.CommitFormat.TypeFormat)
	if err != nil {
		printErrorJSON("incomplete_commit", err.Error())
		return 1
	}
	printCommitJSON(cj)
	return 0
}
