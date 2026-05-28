# Unit 19: ai-link-commit

## Goal

Close the loop between a CommitCraft draft and the git commit it
became. Today, after `ai promote` + `git commit`, the only way to
recover the keypoints + per-stage telemetry of a past commit is to
remember the draft id. Three weeks later, that link is gone.

This unit adds:

1. A `commit_hash` column on the `commits` table (via
   `applySchemaMigrations`, default `''`).
2. `commitcraft ai link-commit --id <draft_id> --hash <git_hash>` to
   write the hash onto an existing row.
3. `commitcraft ai show --commit <hash>` to look up by git hash.
4. A new skill step that runs `ai link-commit` automatically right
   after `git commit`, using `git rev-parse HEAD`. The user never has
   to think about it; `git log` becomes the only id they need to
   retrieve any past draft's keypoints.

## Design

- **Hash resolution**: `--hash` accepts short or full git hashes. We
  resolve to the full 40-char hash via `git rev-parse <hash>` before
  storing, so short-hash lookups later always work regardless of the
  abbreviation the user originally passed.
- **Idempotent**: re-linking the same draft to the same hash is a
  no-op; re-linking to a *different* hash overwrites with a warning
  on stderr.
- **Status independence**: `link-commit` works regardless of the
  draft's status (draft / completed). The link is metadata, not a
  state transition.
- **`ai show --commit <hash>` semantics**: matches by hash prefix
  when the input is shorter than 40 chars, requiring a unique match.
  Ambiguous prefix → exit 1 with `ambiguous_hash` and the list of
  candidate IDs in the error JSON.
- **Backwards compatibility**: every existing draft row has
  `commit_hash = ''`. The new `show --commit` simply won't find
  them; the user can still use `show --id` as before.

## Implementation

### `internal/storage/database.go` — migration

Add one entry to the `alterations` slice in `applySchemaMigrations`:

```go
{
    tableName:    "commits",
    columnName:   "commit_hash",
    columnType:   "TEXT",
    defaultValue: "''",
},
```

The `duplicate column name` guard already in the loop makes this
idempotent across runs.

### `internal/storage/types.go` — struct field

Add to `Commit`:

```go
CommitHash string
```

### `internal/storage/queries.go` — read/write the column

- `GetCommits` (line 24): add `commit_hash` to SELECT + scan.
- `GetCommitByID` (line 56): same.
- New `GetCommitByHash(hash string) ([]Commit, error)` that returns
  every row whose `commit_hash` starts with the given prefix. Hash
  shorter than 4 chars is rejected (`invalid_input`).
- New `LinkCommitHash(id int, hash string) error` that runs
  `UPDATE commits SET commit_hash = ? WHERE id = ?` and returns
  `sql.ErrNoRows` (wrapped) when id doesn't exist.

INSERT / UPDATE statements (`CreateCommit`, `SaveDraft`, the two
update queries at lines 249 / 491) **don't need to change** — the
column has a default value, and the link path is the only one that
writes to it.

### `internal/cli/ai/link_commit.go` (new)

```go
func runLinkCommit(args []string) int {
    fs := flagSet("ai link-commit")
    id   := fs.Int("id", 0, "Draft id to link. Required.")
    hash := fs.String("hash", "", "Git commit hash (short or full). Required.")
    workspace := fs.String("workspace", "", "Repo path. Defaults to cwd.")
    // ... parse ...

    if *id <= 0 || strings.TrimSpace(*hash) == "" { /* usage error */ }

    boot, _ := loadBootstrap(); defer boot.db.Close()
    ws := *workspace; if ws == "" { ws = boot.pwd }

    fullHash, err := git.ResolveCommitHash(*hash) // already exists, line 278
    if err != nil { printErrorJSON("invalid_input", err.Error()); return 2 }

    // Best-effort: warn if the existing link points elsewhere.
    existing, err := boot.db.GetCommitByID(*id)
    if err != nil { printErrorJSON("not_found", err.Error()); return 1 }
    if existing.CommitHash != "" && existing.CommitHash != fullHash {
        fmt.Fprintf(os.Stderr,
            "warning: draft %d was already linked to %s; overwriting with %s\n",
            *id, existing.CommitHash[:7], fullHash[:7])
    }

    if err := boot.db.LinkCommitHash(*id, fullHash); err != nil {
        printErrorJSON("db_error", err.Error()); return 1
    }

    // Emit the canonical JSON envelope so callers can pipe it.
    saved, _ := boot.db.GetCommitByID(*id)
    cj, _ := commitToJSON(saved, loadStagesForCommit(boot.db, *id),
        boot.cfg.CommitFormat.TypeFormat)
    printCommitJSON(cj)
    return 0
}
```

`ResolveCommitHash` already exists in `internal/git/git.go:278` — we
reuse it. (It runs `git rev-parse <rev>` and trims.)

### `internal/cli/ai/show.go` — extend with `--commit`

Add a `--commit <hash>` flag to the existing `runShow`. Behavior:

- `--id` and `--commit` are mutually exclusive (one or the other).
- When `--commit` is given, resolve via `db.GetCommitByHash(prefix)`.
- Unique match → emit the same JSON envelope as the `--id` path.
- Zero matches → exit 1, `not_found`.
- Multiple matches → exit 1, `ambiguous_hash`, with the candidate
  ids listed in the error JSON.

### `internal/cli/ai/ai.go`

- Register `case "link-commit": return runLinkCommit(rest)`.
- Add to `usage`:

  > `link-commit        Associate a draft id with a git commit hash (so `ai show --commit <hash>` works later).`

### `commitJSON` envelope

Add the new field to the JSON wire shape in `ai.go`:

```go
CommitHash string `json:"commit_hash,omitempty"`
```

`commitToJSON` populates it from `c.CommitHash`. `omitempty` keeps
old rows out of the output until they're linked.

### `cmd/cli/main.go` + `CHANGELOG.md`

- Bump to `v0.60.0`.
- CHANGELOG entry covering: the schema migration, the two new
  subcommands (`link-commit` and `show --commit`), the
  `commit_hash` field in `commitJSON`, and the recommended skill
  flow change (auto-link after `git commit`).

### Skill update (`commitcraft-skill/SKILL.md`)

In a paired commit, update the workflow:

- Step 8 ("Create the git commit") gains a new sub-step **8.5**:

  ```sh
  # Capture the new commit's hash and link it to the draft.
  COMMIT_HASH=$(git rev-parse HEAD)
  commitcraft ai link-commit --id <draft_id> --hash "$COMMIT_HASH" || \
    echo "warning: link-commit failed (commit is still good)"
  ```

  Best-effort — if linking fails the commit is already done; surface
  the warning to stderr and continue.

- "Recovering the keypoints after the commit" section: rewrite the
  command list to lead with `ai show --commit <hash>` since draft id
  recovery is no longer needed.

- Step 9 ("Hand back and continue") report-back format adds the
  hash inline (it was already the first field, but spelled out
  short — now we also bake the link confirmation):

  ```
  - Commit: <short_hash> <title>
  - Resumen: <one-line summary>
  ```

  (no new line needed — the hash IS the link key going forward).

## Dependencies

- `git.ResolveCommitHash` (`internal/git/git.go:278`) — already
  shipped, reused here.

## Verify when done

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] Migration smoke test:
  - First `commitcraft` invocation after upgrade adds the column
    without errors.
  - Existing rows queryable via `ai show --id`.
  - Existing rows have `commit_hash` absent from the JSON (because
    `omitempty`).
- [ ] Link a completed draft:
  - `commitcraft ai link-commit --id <id> --hash $(git rev-parse HEAD)` → 0.
  - `commitcraft ai show --id <id>` now includes `commit_hash`.
  - `commitcraft ai show --commit <short>` returns the same row.
- [ ] Negative cases:
  - [ ] `--id 0` or `--hash ""` → exit 2, `invalid_input`.
  - [ ] `--hash deadbeef` (nonexistent) → exit 2, `invalid_input`
        with the rev-parse error.
  - [ ] `--id 999999` → exit 1, `not_found`.
  - [ ] Re-link to a different hash → exit 0, warning on stderr,
        new hash takes effect.
  - [ ] `show --commit <ambiguous prefix>` with multiple matches →
        exit 1, `ambiguous_hash`, error JSON lists candidate ids.
- [ ] Final dogfood: after this commit lands, link its own draft id
      to its own commit hash and verify `ai show --commit <hash>`
      returns it.
