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
// (aiengine.RunRelease). The draft is persisted as a normal
// storage.Commit row with Type="MERGE" and Scope=<source branch>, so
// every other subcommand (ai edit / ai show / ai promote / ai verify)
// works on it unchanged.
//
// `ai regenerate` is NOT yet wired for merge drafts — it would route
// through the commit pipeline and produce garbage. Documented in the
// subcommand's `-h` text. For tweaks, use `ai edit`; for a clean
// re-run, invoke `ai merge` again.
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

	releaseCommits := make([]aiengine.ReleaseCommit, len(commits))
	for i, c := range commits {
		releaseCommits[i] = aiengine.ReleaseCommit{
			Hash:    c.Hash,
			Date:    c.Date,
			Subject: c.Subject,
			Body:    c.Body,
		}
	}

	in := aiengine.ReleaseInput{Commits: releaseCommits}
	out, err := aiengine.RunRelease(aiengine.Deps{
		Cfg: boot.cfg, DB: boot.db, Log: boot.log, Pwd: ws,
	}, in)
	if err != nil {
		printErrorJSON("api_error", err.Error())
		return 1
	}

	messageEN := aiengine.ComposeFinalMessage(out.Title, out.Body, "")

	c := storage.Commit{
		Type:        "MERGE",
		Scope:       branchName,
		Workspace:   ws,
		Diff_code:   serializeCommitRange(commits),
		IaSummary:   out.Body,
		IaCommitRaw: out.Body,
		IaTitle:     out.Title,
		MessageEN:   messageEN,
		Source:      "ai",
	}
	if err := boot.db.SaveDraft(&c); err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}

	saved, err := boot.db.GetCommitByID(c.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: post-save reload failed: %v\n", err)
		saved = c
	}

	cj, err := commitToJSON(
		saved,
		nil, // release-pipeline telemetry persistence is a follow-up
		boot.cfg.CommitFormat.TypeFormat,
	)
	if err != nil {
		printErrorJSON("format_error", err.Error())
		return 1
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(cj)
	return 0
}

// serializeCommitRange stores the input commit list on the draft's
// Diff_code field so it stays inspectable after the fact. Plain-text
// format mirroring `git log --oneline` is enough for traceability;
// regeneration on merge drafts is out of scope for this unit.
func serializeCommitRange(commits []git.CommitRange) string {
	var b strings.Builder
	for _, c := range commits {
		fmt.Fprintf(&b, "%s %s %s\n%s\n\n", c.Hash, c.Date, c.Subject, c.Body)
	}
	return strings.TrimRight(b.String(), "\n")
}
