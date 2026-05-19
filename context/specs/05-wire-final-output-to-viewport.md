# Unit 05: wire-final-output-to-viewport

## Goal

Make the **final release card** in `stateReleaseBuildingText` render the polished release note as soon as the AI cascade succeeds, regardless of which path triggered the run. Today the card can come up empty because the cascade writes `releaseFinalOutput` / `releaseText` / `commitLivePreview` from inside the `tea.Cmd` goroutine in `iaReleaseCascade` (`internal/tui/ai_pipeline.go:272,275,276`), which is racy against the View pass triggered by the very `IaReleaseBuilderResultMsg` that marks the stages done. Move every "final output" model write into the `Update` handler so the assignments and the stage-status flip happen in the same single-threaded turn.

## Context (what's already wired)

- `pipelineShowsFinalCard` returns true when `pipeline.allDone()` and `strings.TrimSpace(releaseFinalOutput) != ""` (`pipeline_view.go:484-495`).
- `pipelineFinalCardContent` returns `releaseFinalOutput` for the release preset (`pipeline_view.go:501-508`).
- `renderFinalCommitCard` renders that content through `renderCommitMessage` and is appended to the pipeline panel when `showFinal` is true (`pipeline_view.go:280-282`).
- `applyPipelineResult` flips touched stages to `statusDone` on the `IaReleaseBuilderResultMsg` turn (`pipeline_update.go:441-473`).
- Unit 08 already added the trailing assignments to `runReleaseCascade` (`ai_pipeline.go:272-276`), but they happen on the goroutine and are not reflected back through the result message — so the read in `View()` immediately after the status flip can land before the write is visible, leaving the card empty.

## Design

No visual or layout changes. The fix is purely a data-flow rewire so the final card paints its content in the same frame that `applyPipelineResult` makes it visible.

## Implementation

### 1. Carry the cascade outputs on `IaReleaseBuilderResultMsg`

File: `internal/tui/update.go`

Extend the message struct so the cascade returns its outputs as data instead of mutating the model:

```go
type IaReleaseBuilderResultMsg struct {
    Err   error
    From  stageID
    Body  string // populated when From <= stageSummary
    Title string // populated when From <= stageBody
    Final string // always populated on success
}
```

`Body` / `Title` are filled only when the corresponding stage actually ran (matches the existing `pushStageHistory` gating). `Final` is always set on success because stage 3 always runs.

### 2. Have the cascade return data, not mutate

File: `internal/tui/ai_pipeline.go`

Refactor `iaReleaseCascade` to return `(body, title, final string, err error)` instead of mutating `model.releaseBodyOutput` / `model.releaseTitleOutput` / `model.releaseFinalOutput` / `model.commitLivePreview` / `model.releaseText` directly. Per-stage `recordStageStats` calls stay (they update the `pipeline.stages` telemetry slice — the cascade still owns that and it's part of the agreed pattern; only the output strings move out).

Remove the trailing `model.commitLivePreview = final` and `model.releaseText = final` assignments — those move to the Update handler.

Keep `iaReleaseBuilder(model)` as the public entry, but change its signature accordingly.

### 3. Have the cmd wrappers carry outputs back

File: `internal/tui/commands.go`

Update `callIaReleaseBuilderCmd` and `callIaReleaseCascadeCmd` so the closure captures the returned strings and forwards them via the message:

```go
func callIaReleaseBuilderCmd(model *Model) tea.Cmd {
    return func() tea.Msg {
        body, title, final, err := iaReleaseBuilder(model)
        return IaReleaseBuilderResultMsg{
            Err: err, From: stageSummary,
            Body: body, Title: title, Final: final,
        }
    }
}
```

Same shape for `callIaReleaseCascadeCmd` with `from` threaded through.

### 4. Apply outputs in the result handler

File: `internal/tui/update.go` (the `case IaReleaseBuilderResultMsg:` block around line 801)

Before the existing `pushStageHistory` calls, assign the outputs into the model from the message payload, gated on success and on the right `From` slice:

```go
if msg.Err == nil {
    if msg.From <= stageSummary {
        model.releaseBodyOutput = msg.Body
    }
    if msg.From <= stageBody {
        model.releaseTitleOutput = msg.Title
    }
    model.releaseFinalOutput = msg.Final
    model.releaseText = msg.Final
    model.commitLivePreview = msg.Final
}
```

This guarantees:

- The writes happen on the Bubble Tea main goroutine, in the same `Update` turn that flips stages to `statusDone`.
- `pipelineShowsFinalCard` and `pipelineFinalCardContent` read consistent state on the very first render after success.
- The picker-path wiring (`update_release.go:604`) that pre-seeds `commitLivePreview = releaseText` keeps working unchanged.

### 5. Documentation

- Bump `cmd/cli/main.go` version from `v0.51.2` to `v0.51.3` (patch — bug fix, no user-visible API change).
- Add a `## v0.51.3 — 2026-05-13` entry to `CHANGELOG.md` describing the fix. Skip the `### Usage` subsection (internal bug fix, no surface change).

## Out of scope

- The empty middle viewport in `stateReleaseChoosingCommit` after returning from a release run (`releaseViewport` / `commitLivePreview`). Already wired by `update_release.go:604` and now reinforced by step 4 above. If we still see it blank in practice, capture as a new unit.
- The `-r` direct-launch divergence from the `-w` rewrite-as-release flow — that's Unit 06.
- Any change to `renderFinalCommitCard` layout or to `pipelineShowsFinalCard` gating.

## Verify when done

- [ ] `go build ./...` passes.
- [ ] From the release picker, select N commits and press Enter. When the pipeline completes, the "final release ready" card displays the refined release note (not blank, not the body/title from earlier stages).
- [ ] Per-stage retry (`1` / `2` / `3` in `stateReleaseBuildingText`) still updates the final card after the cascade finishes — the message payload paths for partial cascades all set `Final`.
- [ ] No goroutine writes to `releaseBodyOutput` / `releaseTitleOutput` / `releaseFinalOutput` / `releaseText` / `commitLivePreview` remain in `ai_pipeline.go` — only the Update handler writes them.
- [ ] `CHANGELOG.md` has the new entry and `cmd/cli/main.go` has the bumped version.
