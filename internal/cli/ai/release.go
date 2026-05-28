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

// runRelease generates a [RELEASE] draft from the commits between
// <from>..<to> using the release pipeline (aiengine.RunRelease) — the
// same engine ai merge and the TUI's release mode use. Persists as a
// normal storage.Commit row with Type="RELEASE" and Scope=<version>,
// so ai edit / ai show / ai verify / ai promote all work on it.
//
// This subcommand only DRAFTS the release notes. Publishing (gh
// release create, tag push, binary upload) stays a follow-up
// (`ai release publish`) so the agent can stop at promote without
// needing GH credentials.
//
// Storage caveat: the TUI persists release runs to a separate
// `releases` table (storage.Release); the headless flow writes to
// `commits` instead. The two surfaces don't see each other's drafts
// today. A future unit can unify them.
func runRelease(args []string) int {
	fs := flagSet("ai release")
	version := fs.String(
		"version",
		"",
		"Release version (e.g. v1.2.3). Used as the draft scope and emitted in final_message. Required.",
	)
	from := fs.String(
		"from",
		"",
		"Base ref for the commit range. Defaults to the most recent tag.",
	)
	to := fs.String(
		"to",
		"HEAD",
		"Tip ref for the commit range. Defaults to HEAD.",
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

	versionStr := strings.TrimSpace(*version)
	if versionStr == "" {
		printErrorJSON("invalid_input", "--version is required")
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

	baseRef := strings.TrimSpace(*from)
	if baseRef == "" {
		last, lastErr := lastTagAt(ws)
		if lastErr != nil || last == "" {
			printErrorJSON("no_base_ref",
				"no --from given and no tags found in the repo; pass --from explicitly")
			return 2
		}
		baseRef = last
	}

	tipRef := strings.TrimSpace(*to)
	if tipRef == "" {
		tipRef = "HEAD"
	}

	if err := git.VerifyRev(ws, baseRef); err != nil {
		printErrorJSON("invalid_input",
			fmt.Sprintf("--from %q not found in %s: %v", baseRef, ws, err))
		return 2
	}
	if err := git.VerifyRev(ws, tipRef); err != nil {
		printErrorJSON("invalid_input",
			fmt.Sprintf("--to %q not found in %s: %v", tipRef, ws, err))
		return 2
	}

	commits, err := git.GetCommitsBetween(ws, baseRef, tipRef)
	if err != nil {
		printErrorJSON("git_error", err.Error())
		return 1
	}
	if len(commits) == 0 {
		printErrorJSON("no_commits_in_range",
			fmt.Sprintf("no commits in %s..%s — nothing to release", baseRef, tipRef))
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
		Type:       "RELEASE",
		Title:      out.Title,
		Body:       out.Body,
		Version:    versionStr,
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
