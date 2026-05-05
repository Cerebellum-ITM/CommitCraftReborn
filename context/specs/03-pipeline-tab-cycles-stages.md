# Unit 03: pipeline-tab-cycles-stages

## Goal

Inside `stateReleaseBuildingText`, `Tab` / `Shift+Tab` must cycle the
focus across the release pipeline's elements — stage 1 (body), stage 2
(title), stage 3 (refine), and the final viewport — rather than
immediately bouncing the user back to the commit picker. "Back to
picker" moves to `Esc` (`Backspace` as fallback). The infrastructure
already exists (`pipelineModel.cycleFocus`, `focusedStage`,
`focusedFinal`); this unit re-wires the release handler to use it and
extends `cycleFocus` to recognise the release final-output card.

## Design

No new visual chrome. The focused card grows / accent border per the
existing `pipeline_view.go::renderPipelinePanel` rendering — which
already keys off `pipeline.focusedStage` / `pipeline.focusedFinal`,
identical to how the commit pipeline tab paints. The "final card" for
release reuses the same panel chrome as the commit-pipeline final card
(rounded border, `theme.Primary` when focused, `theme.Subtle`
otherwise). On-screen key hints render through
`theme.AppStyles().Help` (`ShortKey` / `ShortDesc` / `ShortSeparator`)
per the help-line invariant.

Hint text in the release pipeline status bar must include:

    `tab` cycle stage  ·  `esc` back to picker  ·  `enter` confirm

…rendered via `theme.AppStyles().Help`. No flat `Foreground(theme.Muted)`.

## Implementation

### `internal/tui/pipeline_view.go`

The current gate that decides whether the pipeline shows a final card is:

    showFinal := model.pipeline.preset == pipelinePresetCommit &&
        model.pipeline.allDone() &&
        strings.TrimSpace(model.commitTranslate) != ""

Extend it so the release preset also qualifies once stage 3 finishes
and the refined release note is non-empty:

    showFinal := model.pipeline.allDone() && (
        (model.pipeline.preset == pipelinePresetCommit &&
            strings.TrimSpace(model.commitTranslate) != "") ||
        (model.pipeline.preset == pipelinePresetRelease &&
            strings.TrimSpace(model.releaseFinalOutput) != ""))

This unit only **enables** the final-card slot for release; **wiring
its content into the actual viewport** is Unit 05's job. Until Unit 05
ships, `model.releaseFinalOutput` may be empty and `showFinal` will
stay false on release runs, which is fine — the cycle then runs
through body → title → refine → body.

A small refactor in `renderFinalCommitCard` (or its caller) is
acceptable if the function name no longer fits — rename to
`renderFinalCard` and pass the rendered text + label as parameters.
Keep the rename in scope only if it actually unblocks Unit 05; if not,
leave the symbol alone.

### `internal/tui/pipeline_state.go`

`cycleFocus(showFinal bool)` already handles the canonical cycle
(stage 1 → … → stage N → final → stage 1). No changes needed here as
long as `activeStages == 3` for the release preset (which is the
established convention — release runs body / title / refine, no
changelog stage).

Verify by reading `pipelinePresetRelease` initialization
(`internal/tui/pipeline_state.go` around `resetForPreset`); if
`activeStages` is not set to 3 there, set it explicitly.

### `internal/tui/update_release.go`

Rewrite `updateReleaseBuildingText`'s key handlers (around lines
347-389):

- **`Tab` / `model.keys.NextField`**: replace the
  "back-to-picker" implementation with
  `model.pipeline.cycleFocus(showFinal)`, where `showFinal` is computed
  the same way the renderer does (extract a tiny helper —
  `releasePipelineShowFinal(model)` — used by both the renderer and
  this handler so they can't drift). After the cycle, if
  `pipeline.focusedFinal` is true and the release viewport content is
  empty, leave `model.focusedElement` untouched (Unit 05 handles
  content wiring); otherwise update `model.focusedElement` to a token
  that reflects "stage card focused" — reuse `focusViewportElement`
  for now (it's the legacy "release viewport" focus and the existing
  `releaseViewport.Update` handler already gates on it). Document the
  reuse in a comment.
- **`Shift+Tab` / `model.keys.PrevField`**: add a `cycleFocus` reverse
  variant. The existing `cycleFocus` is forward-only; either add
  `cycleFocusBackward` to `pipelineModel` (mirror `cycleFocus`'s logic
  with `(current - 1 + total) % total`) or call `cycleFocus` `total - 1`
  times. Prefer adding `cycleFocusBackward` — clearer and called from
  the commit-pipeline handler later for free.
- **`Esc` / `Backspace`**: add a new branch that returns to the
  picker. Use `key.Matches(msg, model.keys.Esc)` and
  `key.Matches(msg, model.keys.Back)` (or whichever existing binding
  represents Backspace; verify in `keys.go`). Action:

      model.focusedElement = focusReleaseChooseCommitList
      model.state = stateReleaseChoosingCommits

  …i.e. the body of today's `Tab` handler. Don't add a confirmation —
  the picker preserves selection state, and re-entering the pipeline
  with `Enter` will reuse the existing pipeline output (no re-run).

  Caveat: do NOT bind `Esc` here if the pipeline is still running —
  that should remain the existing cancellation gesture, mirroring
  `updatePipeline`'s `model.keys.Esc` branch
  (`pipeline_update.go:172-178`). Guard with
  `if model.pipeline.anyRunning() { return model, model.pipelineCancel() }`
  before the picker-return case.

### `internal/tui/keys.go`

If a `Back` / `Backspace` binding does not already exist, add one:

    Back: key.NewBinding(
        key.WithKeys("backspace"),
        key.WithHelp("backspace", "back"),
    ),

If a global `Back` binding is already used elsewhere with a conflicting
meaning, just bind `Esc` (which is already in `model.keys.Esc`) and
skip Backspace — don't introduce a new binding for the sake of it.

### Status bar hint

Update the release pipeline status-bar render path
(`compose_status.go` around line 339 where `stateReleaseBuildingText`
is matched) to surface the new hint set:

    [tab] cycle stage · [shift+tab] back · [esc] picker · [enter] confirm

Render via `theme.AppStyles().Help` `ShortKey` / `ShortDesc` /
`ShortSeparator`. Drop any "tab back to picker" copy.

### `internal/tui/keybindings_popup.go`

Add the new release-pipeline rows so the `?` popup advertises the new
behavior:

    {"tab / shift+tab", "cycle stage cards"},
    {"esc",              "back to commit picker"},

Keep the existing global Ctrl+1/2/3 tab navigation rows untouched —
those are about the top-level tab bar, not stage focus, and they
already work post-`c7a9654`.

## Dependencies

- none. (Unit 01 dependency from the build plan is already shipped on
  this branch.)

## Verify when done

- [ ] `Tab` from `stateReleaseBuildingText` advances focus across stage
      cards 1 → 2 → 3 → (final, when populated) → 1, with the focused
      card visibly growing per the existing renderer.
- [ ] `Shift+Tab` walks the cycle in reverse.
- [ ] `Esc` from `stateReleaseBuildingText` returns to
      `stateReleaseChoosingCommits` with `focusReleaseChooseCommitList`,
      preserving the prior commit selection set and the pipeline
      output.
- [ ] `Esc` while a stage is still `Running` cancels the pipeline (no
      regression vs. `updatePipeline`).
- [ ] After `Esc` → picker → `Enter` to re-enter the pipeline, the
      existing `releaseBodyOutput / releaseTitleOutput /
      releaseFinalOutput` are NOT re-generated; the cards re-render
      from the cached state.
- [ ] When `model.releaseFinalOutput` is empty (Unit 05 not yet
      shipped), the cycle naturally skips the final slot — body →
      title → refine → body — without crashing or visually flashing
      an empty card.
- [ ] Status bar hint and `?` popup reflect the new bindings; both
      render through `theme.AppStyles().Help` (no flat `Muted`).
- [ ] `grep -n "stateReleaseBuildingText" internal/tui/update_release.go`
      shows the new `cycleFocus` / `cycleFocusBackward` calls in the
      `NextField` / `PrevField` arms; the legacy "back to picker"
      bodies are gone from those arms.
- [ ] `go build ./...` and `go vet ./...` pass.
- [ ] `cmd/cli/main.go` version bumped to `v0.50.0` (minor — new
      user-visible key bindings, behavioural change to existing key).
- [ ] `CHANGELOG.md` has a new top entry in English with a `### Usage`
      subsection listing `Tab` / `Shift+Tab` / `Esc` / `Backspace`
      semantics in the release pipeline view.
- [ ] Pre-commit hook (`gofumpt → goimports-reviser → golines`) passes.
