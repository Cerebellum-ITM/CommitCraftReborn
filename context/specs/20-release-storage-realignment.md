# Unit 20: release-storage-realignment

## Goal

Move `ai release` and `ai merge` drafts from the `commits` table to
the `releases` table — where the TUI already keeps every row produced
by the release pipeline. Keep the CLI surface identical (`ai release`,
`ai merge`, `ai show/edit/verify/promote/list`) by adding implicit
dispatch in the shared subcommands so the agent never has to know
about the split.

Outcome from the user's perspective: `[RELEASE]` and `[MERGE]` rows
disappear from the TUI's commits History tab (they were never
supposed to be there) and start appearing in the TUI's Releases view.
The agent's flow doesn't change.

## Design

### Storage

- New columns on `releases` (via `applySchemaMigrations`, idempotent):
  - `source TEXT NOT NULL DEFAULT 'tui'` — `tui` / `ai` discriminator.
    Default `'tui'` so the existing rows backfill correctly.
  - `status TEXT NOT NULL DEFAULT 'completed'` — release rows the TUI
    created are effectively completed; new headless drafts start at
    `'draft'` and flip to `'completed'` via `ai promote`.
  - `commit_hash TEXT NOT NULL DEFAULT ''` — lets `ai link-commit`
    work on MERGE rows (the merge produces a real git commit; the
    hash is recoverable like any other).
- `storage.Release` struct gains `Source`, `Status`, `CommitHash`,
  plus `IaSummary` for parity with the verifier (`Body` already
  exists). No, keep it minimal: just the three migration fields.
- Field mapping decision:
  - **MERGE row**: `Type="MERGE"`, `Branch=<source>`, `Version=""`,
    `Title=<ai title>`, `Body=<ai body>`. Composed `final_message`
    = `[MERGE] <branch>: <title>\n\n<body>`.
  - **RELEASE row**: `Type="RELEASE"`, `Branch=""`,
    `Version=<vX.Y.Z>`, otherwise identical. Composed
    = `[RELEASE] <version>: <title>\n\n<body>`.

### New storage methods (`internal/storage/queries.go`)

- `GetReleaseByID(id int) (Release, error)` — single-row lookup.
- `SaveReleaseDraft(*Release) error` — INSERT-or-UPDATE based on
  `r.ID == 0`. Sets `status="draft"` on insert.
- `UpdateReleaseFields(*Release) error` — used by `ai edit`. Updates
  title/body/branch/version/type.
- `FinalizeRelease(id int) error` — flips status to `completed`.
- `GetReleasesByStatus(workspace, status string) ([]Release, error)`
  — used by `ai list` to merge into the commit list.
- `LinkReleaseHash(id int, hash string) error` — parity with
  `LinkCommitHash`.

### CLI dispatch

Helper `dispatchByID(db, id, kindHint) (dispatchResult, err)` in
`internal/cli/ai/dispatch.go`. When `kindHint` is `"commit"` or
`"release"`, restricts the lookup to that table. When empty, probes
`commits` first and falls back to `releases` on miss.

**Collision-safe hint** — Smoke testing surfaced an id collision
(`commits.id=40` AND `releases.id=40` after this unit's first
release draft landed at the fresh table's id=40). Auto-probe hit
`commits` and silently mutated the wrong row. Every shared
subcommand therefore exposes an optional `--kind commit|release`
flag that forces the table. The agent should persist the `kind`
field from the JSON envelope of `ai release` / `ai merge` and pass
it back to every subsequent call.

Subcommands that switch on kind:

- **`ai show --id`**: returns the JSON envelope from whichever side
  matched.
- **`ai edit --id`**: for commits, current behaviour. For releases,
  the supported flags shrink to `--title`, `--body`. Trying `--scope`,
  `--tag`, `--changelog` on a release row returns
  `unsupported_field_for_release` with the rule names so the agent
  knows.
- **`ai verify --id`**: composes the release's final_message
  (`[TYPE] <branch|version>: title\n\nbody`) and runs the same
  verifier. Same rule set, same exit codes.
- **`ai promote --id`**: for commits, current behaviour (FinalizeCommit
  + optional changelog write). For releases, FinalizeRelease + emit
  the JSON envelope. No changelog write step on releases.
- **`ai link-commit --id --hash`**: for commits → `LinkCommitHash`.
  For releases → `LinkReleaseHash`. RELEASE rows accepting a hash is
  unusual but not invalid (the row that gets linked is the
  `[RELEASE]` notes row, the hash would be the commit that became the
  release, e.g. a "bump version" commit).
- **`ai list --status`**: query both tables, merge, sort by
  `created_at DESC`. Each entry carries `kind` so the agent can
  branch.

### JSON envelope changes

`commitJSON` (`internal/cli/ai/ai.go`) gains:

- `Kind string` (required: `"commit"` or `"release"`).
- `Branch string` (`omitempty`) — populated for MERGE releases.
- `Version string` (`omitempty`) — populated for RELEASE releases.

`Scope` keeps backward-compatibility semantics: for releases, it's
populated with Branch (MERGE) or Version (RELEASE) so old consumers
don't break.

A small helper `releaseToCommitJSON(r Release, typeFormat string)`
projects a Release into the same envelope shape. Verifier composition
of `final_message` and the same fields layout.

### Writes — `ai release` and `ai merge`

Both refactored to:

1. Build a `storage.Release` instead of `storage.Commit`.
2. Set `Source="ai"`, `Status="draft"`.
3. `db.SaveReleaseDraft(&r)`.
4. Print `releaseToCommitJSON(r, ...)`.

Their CLI flags don't change.

### Skill repo cleanup (paired commit)

- "Release notes" section → remove the "Storage divergence from TUI"
  paragraph; it's no longer true.
- "What this skill does NOT do" → remove the parallel mention.
- No behavioural change in the documented workflow.

## Orphaned legacy rows

Drafts created by previous units that wrote to `commits` (999, 1000,
1002, 1003, 1015) stay where they are — accessible by
`ai show --id <old>` (commits table wins the dispatch lookup), but no
longer surface in the TUI's Releases view. Documented in CHANGELOG;
no retroactive migration.

## Implementation

Single commit. Concrete file list:

- `internal/storage/database.go` — three new migration entries.
- `internal/storage/types.go` — new fields on `Release`.
- `internal/storage/queries.go` — new methods + updated SELECTs/INSERTs
  for `releases`.
- `internal/cli/ai/dispatch.go` (new) — the dispatch helper.
- `internal/cli/ai/ai.go` — envelope changes, `releaseToCommitJSON`
  helper, register-on-ai-show retains `--commit` lookup over commits
  table only (hash linking semantics differ — releases use their own
  `LinkReleaseHash`).
- `internal/cli/ai/release.go` — write to `releases`.
- `internal/cli/ai/merge.go` — write to `releases`.
- `internal/cli/ai/show.go` — dispatch.
- `internal/cli/ai/edit.go` — dispatch + release-specific flag
  validation.
- `internal/cli/ai/verify.go` — dispatch + release final_message.
- `internal/cli/ai/promote.go` — dispatch.
- `internal/cli/ai/link_commit.go` — dispatch.
- `internal/cli/ai/list.go` — query both tables, merge, include kind.
- `cmd/cli/main.go` — bump to v0.61.0.
- `CHANGELOG.md` — entry with the migration note + orphan caveat.
- `context/progress-tracker.md` — session note.

Skill paired commit:

- `commitcraft-skill/SKILL.md` — drop the storage-divergence
  paragraphs.

## Dependencies

- Unit 17 (`ai merge`) — relocates its storage target.
- Unit 18 (`ai release`) — same.
- Unit 19 (`ai link-commit`) — extends to support release rows.

## Verify when done

- [ ] `go build ./...` + `go vet ./...` clean.
- [ ] Migrations apply on the existing DB without errors.
- [ ] `ai release --version v0.61.0` returns JSON with
      `kind: "release"`, `version: "v0.61.0"`, persists in
      `releases` table.
- [ ] `ai merge --branch <name>` returns JSON with `kind: "release"`
      (the TYPE inside is `MERGE`), `branch: "<name>"`, persists in
      `releases` table.
- [ ] `ai show --id <new_release_id>` returns the same envelope.
- [ ] `ai verify --id <new_release_id>` runs the rules against the
      composed `[MERGE|RELEASE] scope: title\n\nbody` string.
- [ ] `ai edit --id <new_release_id> --title "..."` updates the row.
- [ ] `ai edit --id <new_release_id> --scope foo` → exit 2,
      `unsupported_field_for_release`.
- [ ] `ai promote --id <new_release_id>` flips status to `completed`.
- [ ] `ai list -status completed` returns rows from BOTH tables,
      each tagged with the appropriate `kind`.
- [ ] `ai show --id <legacy_commit>` still works (e.g. 993).
- [ ] TUI smoke (manual): launch `commitcraft`, confirm that the
      Releases view shows the new headless drafts. Commits view
      stays clean of `[RELEASE]`/`[MERGE]` rows.
- [ ] Skill repo paired commit removes the obsolete storage notes.
