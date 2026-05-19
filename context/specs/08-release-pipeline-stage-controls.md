# Unit 08: release-pipeline-stage-controls

## Goal

Wire the per-stage controls available in `updatePipeline` (commit
pipeline) into `updateReleaseBuildingText` so the release pipeline
view has parity: `r` (full retry), `1` / `2` / `3` (per-stage retry,
cascading downstream), `H` (focused stage history), `pgup` / `pgdn`
(scroll the focused stage's output). Per-stage retries must be **real
per-stage retries** at the engine level, not aliases for full
re-runs.

## Design

No new visual chrome — same pipeline cards, same focused-card grow
behaviour from Unit 03, same status-bar level usage as the commit
pipeline (`statusbar.LevelAI` for "pipeline retry · stage N",
`LevelError` when prerequisites are missing).

Hint text in the release pipeline status bar:

    `tab/⇧tab` cycle stage  ·  `r` retry all  ·  `1/2/3` retry stage  ·
    `pgup/pgdn` scroll stage  ·  `↵` create  ·  `esc` back to picker  ·
    `^x` quit

`?` popup mirrors that with longer descriptions, all rendered through
`theme.AppStyles().Help`.

## Implementation

### `internal/aiengine/release.go`

Split `RunRelease` into three callable primitives:

- `RunReleaseBody(deps, in)` → `(body, *api.CallStats, error)` —
  selected commits → release body.
- `RunReleaseTitle(deps, body, in)` → `(title, *api.CallStats, error)`
  — existing body + commits → release title.
- `RunReleaseRefine(deps, body, title)` → `(final, *api.CallStats, error)`
  — existing body + title → polished final note.

`RunRelease` becomes a thin sequencer: `RunReleaseBody` →
`RunReleaseTitle` → `RunReleaseRefine`, recording per-stage telemetry
through the existing `recordReleaseStage` helper.

### `internal/tui/ai_pipeline.go`

Add `iaReleaseCascade(model *Model, from stageID) error`:

- `from == stageSummary`: re-run body, then title, then refine.
- `from == stageBody`: reuse cached `model.releaseBodyOutput`, run
  title + refine.
- `from == stageTitle`: reuse cached body + title, run refine only.

Refine always runs (cheapest stage, owns the user-visible final
output). The existing `iaReleaseBuilder` becomes a one-line wrapper
calling `iaReleaseCascade(model, stageSummary)`.

A small helper `wrapReleaseErr` funnels per-stage errors into the
same log + status shape `iaReleaseBuilder` used to produce, so the
Update handler doesn't special-case partial cascades.

### `internal/tui/commands.go`

Add `callIaReleaseCascadeCmd(model, from)` mirroring
`callIaReleaseBuilderCmd` but parameterised on the starting stage.
The existing `callIaReleaseBuilderCmd` stays for the initial run path
(now sets `From: stageSummary` on the result message).

### `internal/tui/update.go`

`IaReleaseBuilderResultMsg` gains a `From stageID` field. The
result handler in `update.go` only pushes history for stages from
`From` onwards (a stage 3 retry only pushes refine's history; body
and title histories from the previous run stay intact). The
`touchedR` slice is also computed from `From` so error visualisation
flashes only the stages that actually re-ran.

### `internal/tui/pipeline_update.go`

Add `(*Model) pipelineReleaseRetryStage(from stageID) tea.Cmd`,
analogous to the commit pipeline's `pipelineRetryStage`:

- Validates `from <= stageTitle` (release pipeline has 3 stages, so
  stage 4 is a no-op).
- Validates `len(model.selectedCommitList) > 0` and surfaces a
  `LevelError` status bar message if missing.
- Calls `model.pipeline.resetFrom(from, time.Now())`.
- Clears `model.releaseBodyOutput / Title / Final` from `from`
  downstream (matching the commit pipeline's clear-then-run pattern).
- Sets `WritingStatusBar` to `LevelAI` with the retry label.
- Returns `tea.Batch` of: spinner start, pipeline spinner tick,
  pulse tick, and `callIaReleaseCascadeCmd(model, from)`.

### `internal/tui/update_release.go::updateReleaseBuildingText`

Mirror the commit pipeline's keymap arms (above the existing Esc /
Enter / Tab / Shift+Tab arms from Units 03–04):

- `H`: `openStageHistoryPopup(model, model.pipeline.focusedStage)`.
- `model.keys.Toggle` (`r`): `pipelineReleaseRetryStage(stageSummary)`.
- `model.keys.RerunStage1` (`1`): `pipelineReleaseRetryStage(stageSummary)`.
- `model.keys.RerunStage2` (`2`): `pipelineReleaseRetryStage(stageBody)`.
- `model.keys.RerunStage3` (`3`): `pipelineReleaseRetryStage(stageTitle)`.
- `model.keys.PgUp` / `PgDown`: `model.stageViewportModel(focusedStage).HalfPageUp/Down()`.

Stage 4 binding (`RerunStage4`) intentionally omitted — release has no
changelog refiner.

### Status bar + `?` popup

Re-add the rows that were stripped out at the end of Unit 03 to avoid
advertising non-functional bindings. All rows render through
`theme.AppStyles().Help`.

## Dependencies

- Unit 03 (already shipped) — focus cycle infrastructure
  (`focusedStage`, `cycleFocus`/`cycleFocusBackward`).
- Unit 04 (already shipped) — Enter guard sits in the same
  key-handler switch.

## Verify when done

- [ ] After the initial release run finishes:
  - [ ] Pressing `r` re-runs all 3 stages.
  - [ ] Pressing `1` re-runs body → title → refine.
  - [ ] Pressing `2` re-runs only title and refine; the body card
        keeps its previous output.
  - [ ] Pressing `3` re-runs only refine; body and title cards keep
        their previous outputs.
- [ ] After a retry, the focused stage's history popup (`H`) shows
      both the prior and new outputs.
- [ ] `pgup` / `pgdn` scroll the focused stage's viewport.
- [ ] Hitting `1` / `2` / `3` with an empty `selectedCommitList`
      surfaces a `LevelError` status bar message and does not call
      Groq.
- [ ] `go build ./...` + `go vet ./...` pass.
- [ ] `cmd/cli/main.go` version bumped to `v0.51.0` (minor — new
      user-visible bindings).
- [ ] `CHANGELOG.md` has a new top entry in English with a `### Usage`
      block listing the new keys.
- [ ] Pre-commit hook (`gofumpt → goimports-reviser → golines`) passes.
