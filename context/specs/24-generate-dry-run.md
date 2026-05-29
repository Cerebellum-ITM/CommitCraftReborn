# Unit 24: generate-dry-run

## Goal

Add `--dry-run` flag to `commitcraft ai generate`. Runs the full 3-stage AI
pipeline and returns the same JSON output as a normal generate, but skips
every DB write — no `SaveDraft`, no `persistAICalls`. Lets an agent iterate
on keypoint phrasings without polluting the drafts list.

## Design

The flag is a single boolean. When set:

- The AI pipeline (`aiengine.Run`) executes normally — all 3 stages, real
  Groq calls.
- `bs.db.SaveDraft` and `persistAICalls` are skipped.
- The JSON output has `"id": 0` and `"status": "dry_run"` instead of the
  persisted row's values.
- All other fields (`final_message`, `body`, `title`, `summary`, `stages`,
  `kind`, `source`, etc.) are populated from the pipeline output exactly as
  in a normal run.

`ai verify --id 0` won't find a row — that's by design. For dry-run output
the agent reads `final_message` directly from the JSON. If they like it,
they re-run without `--dry-run` to persist.

## Implementation

File: `internal/cli/ai/generate.go`

1. Add `dryRun := fs.Bool("dry-run", false, "Run the pipeline without persisting a draft row.")`.
2. After `aiengine.Run` succeeds and `*dryRun` is true:
   - Skip `bs.db.SaveDraft` and `persistAICalls`.
   - Build a synthetic `storage.Commit` from the pipeline output (same
     fields as the normal path) with `ID = 0` and `Status = "dry_run"`.
   - Pass it to `commitToJSON` and print, then return 0.
3. The normal (non-dry-run) path is unchanged.

## Dependencies

- No new packages, no schema changes.
- `commitToJSON` already handles `id: 0` gracefully (it's just a number in the JSON).

## Verify when done

- [ ] `go build ./...` + `go vet ./...` clean.
- [ ] `commitcraft ai generate --dry-run -k "test" -t ADD -s ai` returns JSON
      with `"id": 0` and `"status": "dry_run"`.
- [ ] Running the same command twice with identical keypoints produces two
      separate pipeline outputs but zero new rows in the DB (confirm via
      `commitcraft ai list`).
- [ ] Normal `commitcraft ai generate` (without `--dry-run`) still persists
      as before.
- [ ] Update `context/progress-tracker.md` to mark unit 24 complete.
