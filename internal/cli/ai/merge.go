package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
)

// runMerge generates a [MERGE] draft from the commits between
// <into>..<branch> using the existing release pipeline
// (aiengine.RunRelease). The draft is persisted as a `releases` row
// with Type="MERGE" and Branch=<source>, alongside TUI-created
// release rows. Shared subcommands (`ai show / edit / verify /
// promote / link-commit`) dispatch on id and operate on this row
// transparently.
//
// `ai regenerate` is NOT yet wired for merge drafts — it would route
// through the commit pipeline and produce garbage. For tweaks, use
// `ai edit`; for a clean re-run, invoke `ai merge` again.
func runMerge(args []string) int {
	fs := flagSet("ai merge")
	branch := fs.String(
		"branch",
		"",
		"Source branch whose commits will be summarized into a [MERGE] draft. Required.",
	)
	into := fs.String(
		"into",
		"main",
		"Target branch the merge is going into. Used for the range expression <into>..<branch>.",
	)
	workspace := fs.String(
		"workspace",
		"",
		"Repo path. Defaults to the current directory.",
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}

	branchName := strings.TrimSpace(*branch)
	if branchName == "" {
		printErrorJSON("invalid_input", "--branch is required")
		return 2
	}
	intoName := strings.TrimSpace(*into)
	if intoName == "" {
		printErrorJSON("invalid_input", "--into cannot be empty")
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

	if err := git.VerifyRev(ws, branchName); err != nil {
		printErrorJSON("invalid_input",
			fmt.Sprintf("branch %q not found in %s: %v", branchName, ws, err))
		return 2
	}
	if err := git.VerifyRev(ws, intoName); err != nil {
		printErrorJSON("invalid_input",
			fmt.Sprintf("into %q not found in %s: %v", intoName, ws, err))
		return 2
	}

	commits, err := git.GetCommitsBetween(ws, intoName, branchName)
	if err != nil {
		printErrorJSON("git_error", err.Error())
		return 1
	}
	if len(commits) == 0 {
		printErrorJSON("no_commits_in_range",
			fmt.Sprintf(
				"no commits in %s..%s — branch is already fully merged or empty",
				intoName, branchName,
			))
		return 1
	}

	in := aiengine.ReleaseInput{Commits: projectToReleaseCommits(commits)}
	out, err := aiengine.RunRelease(aiengine.Deps{
		Cfg: boot.cfg, DB: boot.db, Log: boot.log, Pwd: ws,
	}, in)
	if err != nil {
		printErrorJSON("api_error", err.Error())
		return 1
	}

	r := storage.Release{
		Type:       "MERGE",
		Title:      out.Title,
		Body:       out.Body,
		Branch:     branchName,
		CommitList: serializeCommitRange(commits),
		Workspace:  ws,
		Source:     "ai",
		Status:     "draft",
	}
	if err := boot.db.SaveReleaseDraft(&r); err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}

	saved, err := boot.db.GetReleaseByID(r.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: post-save reload failed: %v\n", err)
		saved = r
	}

	cj, err := releaseToJSON(saved, boot.cfg.CommitFormat.TypeFormat)
	if err != nil {
		printErrorJSON("format_error", err.Error())
		return 1
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(cj)
	return 0
}
