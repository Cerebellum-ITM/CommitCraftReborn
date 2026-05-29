# Unit 17: ai-merge

## Goal

Add `commitcraft ai merge --branch <source> [--into <target>]`: a
headless subcommand that generates a merge-commit draft in
CommitCraft's DB by feeding the commits between `<target>` and
`<source>` through the existing `aiengine.RunRelease` pipeline (the
same 3-stage body → title → refine flow the TUI uses for release
notes). Persists as a normal `storage.Commit` row with `Type="MERGE"`
and `Scope=<source branch>` so the existing `ai edit` / `ai show` /
`ai promote` / `ai verify` commands work on the draft unchanged.

When the user is ready to merge the branch, they run `ai promote --id <ID>`
and then execute `git merge --no-ff <source>` with the composed
`final_message`, which renders as:

```
[MERGE] <source-branch>: <AI-generated title>

<AI-generated body summarizing the branch>
```

Matching the project's existing convention seen in commits like
`3b397c3 [MERGE] feat/keymatches-migration: key.Matches as single
dispatch contract (v0.55.0)`.

## Design

No new prompts. The release pipeline (`aiengine.RunRelease`) is
already designed to summarize a list of commits — exactly the input
shape needed for a merge commit. Reusing it keeps the prompt surface
flat and means any future tuning of the release prompts also improves
merge messages.

Behaviour parity with the rest of `ai …`:

- JSON output on stdout, same `commitJSON` envelope as `ai generate` /
  `ai show`. The agent can pipe it the same way.
- Errors on stderr with `{code, error}` shape and meaningful exit
  codes (1 runtime / 2 usage).
- `Source: "ai"` so the draft is distinguishable from TUI-created rows
  in `ai list`.

Inputs:

- `--branch <name>` (required) — the source branch whose commits will
  be summarized. Verified to exist via `git rev-parse --verify`.
- `--into <name>` (default `main`) — the target branch the merge is
  going into. Used for the range expression `<into>..<branch>`.
- `--workspace <path>` (default `pwd`) — repo path. Same semantics as
  the workspace flag on other subcommands.

No `--keypoint` flag in v1. Merge commits derive their content from
the commit log; if extra steering is needed, the user can `ai edit`
the draft after generation. We may revisit this if it turns out we
consistently want extra context.

Out of scope for this unit (intentional carve-outs):

- **`ai regenerate` for merge drafts**: today `ai regenerate` calls
  `aiengine.Run` (the commit pipeline), not `RunRelease`. Re-running
  it on a merge draft would produce garbage. Documented in the
  subcommand's help text — users should `ai edit` for tweaks or
  re-run `ai merge` from scratch. A future unit can teach
  `ai regenerate` to route on draft type.
- **`gh` integration**: pushing the actual merge commit is outside
  the skill's contract for now. The user (or the skill) runs
  `git merge --no-ff` after promoting.
- **Empty range guard**: if `<into>..<branch>` is empty (branch is
  fully merged already), report `no_commits_in_range` and exit 1.

## Implementation

### `internal/git/git.go` — new helper

```go
// GetCommitsBetween returns the commits reachable from `source` but
// not from `target`, in chronological (oldest-first) order. Each
// entry carries the short hash, ISO date, subject, and body — the
// fields the release/merge pipeline consumes via aiengine.ReleaseCommit.
//
// Implemented as `git -C <workspace> log --reverse --pretty=…
// <target>..<source>`. NUL separators are used between fields and
// records so we can parse without quoting concerns.
func GetCommitsBetween(workspace, target, source string) ([]CommitRange, error)
```

Returns a new struct:

```go
type CommitRange struct {
    Hash    string
    Date    string
    Subject string
    Body    string
}
```

Located alongside `LookupCommitMessages` for symmetry — both are
log-shape helpers. The format spec is `--pretty=format:%h%x00%ad%x00%s%x00%b%x1f`
(`%x1f` = record separator).

### `internal/cli/ai/merge.go` (new)

CLI wrapper:

```go
func runMerge(args []string) int {
    fs := flagSet("ai merge")
    branch := fs.String("branch", "", "Source branch whose commits will be summarized. Required.")
    into   := fs.String("into",   "main", "Target branch the merge is going into.")
    workspace := fs.String("workspace", "", "Repo path. Defaults to current directory.")
    if err := fs.Parse(args); err != nil { ... }
    if strings.TrimSpace(*branch) == "" { usage error; return 2 }

    boot, err := loadBootstrap()
    ...
    ws := *workspace
    if ws == "" { ws = boot.pwd }

    // Validate both refs exist.
    if err := git.VerifyRev(ws, *branch); err != nil {
        printErrorJSON("invalid_input", fmt.Sprintf("branch %q not found: %v", *branch, err))
        return 2
    }
    if err := git.VerifyRev(ws, *into); err != nil {
        printErrorJSON("invalid_input", fmt.Sprintf("into %q not found: %v", *into, err))
        return 2
    }

    commits, err := git.GetCommitsBetween(ws, *into, *branch)
    if err != nil { printErrorJSON("git_error", err.Error()); return 1 }
    if len(commits) == 0 {
        printErrorJSON("no_commits_in_range",
            fmt.Sprintf("no commits in %s..%s — branch is already fully merged?", *into, *branch))
        return 1
    }

    // Project to aiengine.ReleaseCommit and run the release pipeline.
    in := aiengine.ReleaseInput{Commits: toReleaseCommits(commits)}
    out, err := aiengine.RunRelease(aiengine.Deps{
        Cfg: boot.cfg, DB: boot.db, Log: boot.log, Pwd: ws,
    }, in)
    if err != nil { printErrorJSON("api_error", err.Error()); return 1 }

    // Compose the title+body for storage. ComposeFinalMessage already
    // does title + "\n\n" + body, which is what we want.
    messageEN := aiengine.ComposeFinalMessage(out.Title, out.Body, "")

    c := storage.Commit{
        Type:        "MERGE",
        Scope:       *branch,
        Workspace:   ws,
        Diff_code:   serializeCommits(commits), // for traceability + future regenerate
        IaSummary:   out.Body,                  // analyzer-equivalent output
        IaCommitRaw: out.Body,
        IaTitle:     out.Title,
        MessageEN:   messageEN,
        Source:      "ai",
    }
    if err := boot.db.SaveDraft(&c); err != nil { ... }

    // Telemetry: project the 3 release stages into the existing
    // ai_calls table. Use stage names "summary" / "title" / "refine".
    saveMergeAICalls(boot.db, c.ID, out.Stages)

    // Reload and emit canonical JSON.
    saved, _ := boot.db.GetCommitByID(c.ID)
    cj, _ := commitToJSON(saved, loadStagesForCommit(boot.db, c.ID),
        boot.cfg.CommitFormat.TypeFormat)
    printCommitJSON(cj)
    return 0
}
```

### `internal/cli/ai/ai.go`

- Register `case "merge": return runMerge(rest)`.
- Add to `usage`:

  > `merge              Generate a [MERGE] draft from the commits in <into>..<branch>. Uses the release pipeline.`

### `cmd/cli/main.go` + `CHANGELOG.md`

- Bump `version` to `v0.58.0`.
- New CHANGELOG entry above `v0.57.0` describing the subcommand,
  with a `### Usage` block showing the typical flow:

  ```sh
  commitcraft ai merge --branch feat/agent-cli-improvements
  commitcraft ai verify --id <id>     # optional: dogfood the gate
  commitcraft ai edit --id <id> --title "..."   # if needed
  commitcraft ai promote --id <id>
  git checkout main
  git merge --no-ff feat/agent-cli-improvements \
    -m "$(commitcraft ai show --id <id> --field final_message)"
  ```

  (the `--field` flag on `ai show` doesn't exist yet — the spec
  example shows the *intended* helper; for v1 the user pipes the
  JSON through `jq -r .final_message`.)

### `context/progress-tracker.md`

Add Session Notes entry for unit 17. Move from "Next Up" to "In Progress".

## Dependencies

- Unit 15 (umbrella plan) — context only.
- `aiengine.RunRelease` (already shipped) — no changes needed.
- New `git.GetCommitsBetween` + `git.VerifyRev` helpers.

## Verify when done

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Smoke test: from a worktree with a real feature branch,
      `commitcraft ai merge --branch <branch> --into main` returns a
      JSON envelope with `type: "MERGE"`, `scope: <branch>`, non-empty
      `title` + `body` + `final_message`.
- [ ] `commitcraft ai verify --id <id>` against the merge draft
      returns no errors (the title format should already be correct
      because `FormatFinalMessage` wraps it with `[MERGE] <branch>:`).
- [ ] Negative cases:
  - [ ] No `--branch` flag → exit 2, `invalid_input`.
  - [ ] `--branch xxx` (nonexistent) → exit 2, `invalid_input` with
        the rev-parse error in the message.
  - [ ] Branch fully merged (`<into>..<branch>` is empty) → exit 1,
        `no_commits_in_range`.
- [ ] Final dogfood: when we close `feat/agent-cli-improvements`,
      we use `ai merge` to draft the merge commit itself.
