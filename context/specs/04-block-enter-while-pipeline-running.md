# Unit 04: block-enter-while-pipeline-running

## Goal

In `stateReleaseBuildingText`, `Enter` must not open the create-release
popup until every active stage has finished. Today the user can press
`Enter` while stage 3 is still streaming and reach the
type=MERGE/RELEASE picker before the polished output exists — the
release ends up with whatever partial body the cards happened to hold.
After this unit, `Enter` is a no-op while the pipeline is in flight, and
the status bar shows a warning (`statusbar.LevelWarning`) explaining
why. Once `pipeline.allDone()` returns true, `Enter` opens the popup
exactly as it does today.

## Design

No new visual elements. The existing `WritingStatusBar` row at the
bottom of the release pipeline view already renders levels via
`theme.AppStyles().Help` (`ShortKey` / `ShortDesc` / `ShortSeparator`)
and supports `LevelWarning`. Reuse it.

Warning copy:

    pipeline still running · wait for stage 3 to finish before creating

Capitalisation matches the existing release status messages
(`"release pipeline started · 3 stages"`, `"Release creation"` etc.) —
sentence case, `·` separator. No emoji or new symbols.

## Implementation

### `internal/tui/update_release.go::updateReleaseBuildingText`

Add a guard to the `Enter` arm (currently around the lines below the
`Esc` arm I added in Unit 03):

    case key.Matches(msg, model.keys.Enter):
        if !model.pipeline.allDone() {
            model.WritingStatusBar.Content =
                "pipeline still running · wait for stage 3 to finish before creating"
            model.WritingStatusBar.Level = statusbar.LevelWarning
            return model, nil
        }
        // …existing popup-open code unchanged.

Use `pipeline.allDone()` (not `anyRunning()`): the gate is "every
active stage reported `statusDone`", which is strictly stronger than
"none are currently `statusRunning`" — a stage that errored or was
cancelled returns `false` from `anyRunning()` but is also not done, and
we don't want `Enter` to open the popup against a failed run.

If `pipeline.allDone()` returns `true` but `releaseFinalOutput` is
empty (a corner case that today only happens if `RunRelease` returned
zero text), still allow the popup — that path is Unit 05's
responsibility. Don't entangle the two.

### Status-bar restoration

The warning sticks until something else writes to `WritingStatusBar`.
That's acceptable — the existing flow already overwrites it on
state transitions (`Esc`, `Enter` once `allDone()`, etc.). Don't add a
timer or auto-clear — that would diverge from the project's existing
status-bar conventions.

### `internal/tui/keybindings_popup.go`

In `releaseBuildingTextKeybindings()` (added in Unit 03), update the
`↵` row so the help reads:

    {"↵", "open create-release menu (after stage 3 finishes)"}

Keep all other rows untouched.

### Status bar hint

`compose_status.go::helpEntriesForState` for `stateReleaseBuildingText`
keeps `{"↵", "create"}`. Don't change the hint to "create (when ready)"
or similar — the warning shown on press is the right place to surface
the gate, not a permanent footer entry. Hint copy stays terse.

## Dependencies

- Unit 03 (already shipped) — `Esc` arm and the new `Enter` arm sit in
  the same `tea.KeyMsg` switch.

## Verify when done

- [ ] Manual: launch `commitcraft -r`, pick commits, hit Enter to
      start the pipeline, immediately hit `Enter` again while stage 1
      / 2 / 3 is `Running`. The popup does NOT open. The status bar
      shows the warning at `LevelWarning`.
- [ ] Manual: wait for stage 3 to settle (`statusDone` on all active
      stages). Press `Enter`. The popup opens normally.
- [ ] Manual: cancel a running pipeline with `Esc`, then press
      `Enter`. The popup does NOT open (cancelled stages don't satisfy
      `allDone()`). Status bar warning shown.
- [ ] Manual: trigger a stage failure (e.g. invalid model temporarily,
      or kill network mid-run). `Enter` after the failure also stays
      blocked.
- [ ] `grep -n "pipeline.allDone()" internal/tui/update_release.go`
      shows the new guard inside the `Enter` arm.
- [ ] `?` popup row for `↵` in `releaseBuildingTextKeybindings`
      reflects the gate.
- [ ] `go build ./...` and `go vet ./...` pass.
- [ ] `cmd/cli/main.go` version bumped to `v0.50.1` (patch — bug fix,
      no new user-visible surface).
- [ ] `CHANGELOG.md` has a new top entry in English. No `### Usage`
      block needed (no new key, only a guard on an existing one) but
      a one-line note in the body about the warning copy is fine.
- [ ] Pre-commit hook (`gofumpt → goimports-reviser → golines`) passes.
