# Unit 18: ai-release

## Goal

Add `commitcraft ai release --version <vX.Y.Z> [--from <ref>] [--to <ref>]`:
a headless subcommand that generates a release-notes draft from the
commits in `<from>..<to>` by feeding them through the same
`aiengine.RunRelease` pipeline used by `ai merge` and the TUI's
release mode. Persists as a normal `storage.Commit` row with
`Type="RELEASE"` and `Scope=<version>`. The composed `final_message`
renders as `[RELEASE] vX.Y.Z: Title\n\nBody`, ready to be split by a
future `ai release publish` subcommand into GH-release title + body.

This unit only covers **drafting** the release notes. GitHub
publishing (`gh release create`, tag push, binary upload) stays
TUI-only for now — exposing it headless will be a separate, opt-in
subcommand (`ai release publish --id <ID>`) so the agent can stop at
promote without needing GH credentials.

## Design

Same shape as `ai merge` (unit 17). Differences:

- **Type label**: `RELEASE` instead of `MERGE`.
- **Scope**: a version string instead of a branch name. Required.
- **Default range**: `<latest tag>..HEAD` instead of `<into>..<branch>`.
- **Validation**: `--version` must be non-empty (releases are versioned
  by definition); refs validated via `git.VerifyRev`.

Storage caveat — the TUI's release flow persists to a separate
`releases` table (`storage.Release` in `internal/storage/types.go:79`).
The headless `ai release` writes to the regular `commits` table so
that `ai edit` / `ai show` / `ai verify` / `ai promote` work
unchanged. This means TUI-created and CLI-created release drafts
don't see each other; a future unification migration could bridge
them, but it's out of scope here. Documented in CHANGELOG so users
aren't surprised.

Required flags:

- `--version <vX.Y.Z>` — used as Scope and stored verbatim. Must be
  non-empty. No format validation (we don't want to be opinionated
  about SemVer vs CalVer).

Optional flags:

- `--from <ref>` — defaults to the most recent annotated tag via
  `git tag --sort=-v:refname | head -1`. If the repo has no tags,
  the flag becomes required (exit 2 with `no_base_ref` code).
- `--to <ref>` — defaults to `HEAD`.
- `--workspace <path>` — defaults to cwd.

Out of scope for this unit:

- **Commit selection**. The TUI lets the user deselect commits from
  the range before running the pipeline. CLI v1 takes the full
  range; user can `ai edit` the body afterwards to trim.
- **Binary build / upload**. The TUI's release mode runs
  `make build_release` + `gh release create` + asset upload. Those
  stay TUI-only until the `ai release publish` follow-up.

## Implementation

### `internal/cli/ai/release.go` (new)

Mirrors `internal/cli/ai/merge.go` structurally — same flow,
different defaults and storage labels:

```go
func runRelease(args []string) int {
    fs := flagSet("ai release")
    version  := fs.String("version", "", "Release version (e.g. v1.2.3). Required.")
    from     := fs.String("from", "", "Base ref. Default: most recent tag.")
    to       := fs.String("to", "HEAD", "Tip ref. Default: HEAD.")
    workspace := fs.String("workspace", "", "Repo path. Defaults to cwd.")
    // ...

    if strings.TrimSpace(*version) == "" {
        printErrorJSON("invalid_input", "--version is required")
        return 2
    }

    boot, _ := loadBootstrap() ; defer boot.db.Close()
    ws := *workspace; if ws == "" { ws = boot.pwd }

    baseRef := strings.TrimSpace(*from)
    if baseRef == "" {
        last, err := lastTagAt(ws)
        if err != nil || last == "" {
            printErrorJSON("no_base_ref",
                "no --from given and no tags found; pass --from explicitly")
            return 2
        }
        baseRef = last
    }

    if err := git.VerifyRev(ws, baseRef); err != nil { /* invalid_input */ }
    if err := git.VerifyRev(ws, *to);     err != nil { /* invalid_input */ }

    commits, err := git.GetCommitsBetween(ws, baseRef, *to)
    if err != nil || len(commits) == 0 { /* no_commits_in_range */ }

    in := aiengine.ReleaseInput{Commits: toReleaseCommits(commits)}
    out, err := aiengine.RunRelease(aiengine.Deps{...}, in)
    if err != nil { /* api_error */ }

    messageEN := aiengine.ComposeFinalMessage(out.Title, out.Body, "")
    c := storage.Commit{
        Type:        "RELEASE",
        Scope:       *version,
        Workspace:   ws,
        Diff_code:   serializeCommitRange(commits),
        IaSummary:   out.Body,
        IaCommitRaw: out.Body,
        IaTitle:     out.Title,
        MessageEN:   messageEN,
        Source:      "ai",
    }
    boot.db.SaveDraft(&c)

    // Emit canonical JSON.
    saved, _ := boot.db.GetCommitByID(c.ID)
    cj, _ := commitToJSON(saved, nil, boot.cfg.CommitFormat.TypeFormat)
    printCommitJSON(cj)
    return 0
}

// lastTagAt is the workspace-aware sibling of git.GetLastGitTag.
func lastTagAt(workspace string) (string, error) { ... }
```

`serializeCommitRange` and `toReleaseCommits` already exist in
`merge.go` — exported (or extracted into a small helper file
`range_helpers.go`) so `release.go` can reuse them without duplication.

### `internal/cli/ai/ai.go`

- Register `case "release": return runRelease(rest)`.
- Add to `usage`:

  > `release            Generate a [RELEASE] draft from the commits in <from>..<to> using the release pipeline.`

### `cmd/cli/main.go` + `CHANGELOG.md`

- Bump `version` to `v0.59.0`.
- New CHANGELOG entry above `v0.58.0` documenting the subcommand,
  the storage divergence from the TUI release table, and the typical
  flow including the `ai release publish` follow-up that will
  eventually land.

### `context/progress-tracker.md`

Session note for unit 18; move from "Next Up" to "In Progress".

## Dependencies

- Unit 17 (`ai merge`) — reuses the `GetCommitsBetween` helper and
  the `serializeCommitRange` / `toReleaseCommits` projections.

## Verify when done

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Smoke test against this branch:
  - `commitcraft ai release --version v0.59.0` (auto-detects last
    tag `v0.55.0`, includes the 3 + 1 unit-17/unit-18 commits) →
    JSON with `type: "RELEASE"`, `scope: "v0.59.0"`, non-empty
    title + body + final_message.
- [ ] `ai verify --id <id>` against the release draft → clean (or
      `title_too_long_soft` warning is acceptable, like merge).
- [ ] Negative cases:
  - [ ] No `--version` → exit 2, `invalid_input`.
  - [ ] Empty range (`--from HEAD --to HEAD`) → exit 1, `no_commits_in_range`.
  - [ ] Invalid `--from` → exit 2, `invalid_input`.
- [ ] Skill update (paired commit in commitcraft-skill repo):
      add a "Release notes" section under the existing "Merge
      commits" section, mirroring its structure.
