# Unit 14: migrate-msg-string-to-key-matches

## Goal

Replace every main-matcher `msg.String()` dispatch in `internal/tui/update_*.go`
with `key.Matches` against the active keymap, and rewrite the `?` popup so
its per-state functions read the displayed shortcuts from the active
`KeyMap` instead of duplicating literal key strings. Leaves transient /
data-extraction `msg.String()` uses (mention `@`, popup `q`/`esc` close,
scroll inside a focused bubbles component) and the global guards in
`update.go:1049-1115` intentionally untouched.

## Design

No user-visible change. The `?` popup keeps the same group structure
(Navigate / Inspect panel / Commits / App) and the same descriptions —
only the **key column** is now derived from the active `KeyMap`'s
bindings via `binding.Help().Key`. A binding rename (e.g. `r` → `R`)
propagates to the popup automatically.

## Implementation

### `internal/tui/keys.go` — new fields

Add to the `KeyMap` struct:

- `CycleFilterMode key.Binding` — `ctrl+f` (cycle filter mode pill on
  workspace / release history / release commit picker views).
- `ClearField key.Binding` — `x`/`backspace`/`delete` (destructive
  field action: clear scope, remove highlighted keypoint, etc.).
- `Left key.Binding` / `Right key.Binding` — these already exist; just
  populate them where needed.

### `internal/tui/keys.go` — populate the variants

- `mainListKeys()`: add `CycleFilterMode("ctrl+f")`, `PgUp("pgup",
  "ctrl+up")`, `PgDown("pgdown", "ctrl+down")`, `Esc("esc")`.
- `releaseMainListKeys()`: same four additions.
- `releaseKeys()`: add `CycleFilterMode("ctrl+f")` (used in the release
  commit picker dispatch).
- `writingMessageKeys()`: add `Left("left", "h")`, `Right("right", "l")`,
  `ClearField("x", "backspace", "delete")`. Already has `Up("up","k")` /
  `Down("down","j")`, no change there.

### `internal/tui/update_release.go` — migrate dispatchers

- L46 `msg.String() == "ctrl+f"` → `key.Matches(msg, model.keys.CycleFilterMode)`.
- L64-74 filter-focused inner `switch msg.String()` → `switch { case key.Matches(...) }` against `Esc` / `Enter`.
- L88-92 panel-scroll switch → match `model.keys.PgUp` / `model.keys.PgDown` (the bindings now carry both `pgup`+`ctrl+up` and `pgdown`+`ctrl+down`).
- L374-395 release pipeline `switch msg.String()` → `switch { case key.Matches(...) }` against `History`, `Toggle`, `RerunStage1/2/3`, `PgUp`, `PgDown`. Drop the explanatory comment about "Match by raw `msg.String()` rather than `key.Matches`" — it no longer applies.
- L571 release-chooser `msg.String() == "ctrl+f"` → `key.Matches(msg, model.keys.CycleFilterMode)`.

### `internal/tui/update_commit.go` — same pattern

- L147, L169 (filter focused inner switch), L193 (scroll switch) →
  `key.Matches` against `CycleFilterMode`, `Esc`, `Enter`, `PgUp`, `PgDown`.

### `internal/tui/update_writing.go` — per-focus handlers

- L198-209 (`handleTypeSectionKey`): replace `case "left", "h"` / `case "right", "l"` with `case key.Matches(msg, model.keys.Left)` / `key.Matches(msg, model.keys.Right)`.
- L217-235 (`handleScopeSectionKey`): replace `case "x", "backspace", "delete"` with `key.Matches(msg, model.keys.ClearField)`. **Keep `case "e", "enter"` as `msg.String()`** — `e` is a focus-contextual single-key shortcut not advertised in the `?` popup; per the standards rule we wrote, that's the legitimate `msg.String()` carve-out.
- L246-266 (`handleKeypointsSectionKey`): the nav case folds "up/k/left/h" into a single binding match (`Up` OR `Left`) and "down/j/right/l" into (`Down` OR `Right`). Destructive case migrates via `ClearField`.
- L282-299 (`handlePipelineModelsSectionKey`): same nav pattern. `case "enter"` → `key.Matches(msg, model.keys.Enter)`. `case "H"` → `key.Matches(msg, model.keys.History)`.

### `internal/tui/keybindings_popup.go` — keymap-driven popup

Change the four per-state functions
(`workspaceKeybindings`, `releaseKeybindings`,
`releaseChooseCommitsKeybindings`, `releaseBuildingTextKeybindings`)
to **take the active `KeyMap` as a parameter** and build each
`helpEntry.key` from `binding.Help().Key`. Descriptions stay as
written (they're context-rich and shouldn't be derived from the
binding's bare help string).

Update `keybindingsForState(s appState, k KeyMap)` signature and the
caller in `update.go:1143` (`keybindingsForState(model.state, model.keys)`).

For entries that combine multiple bindings (e.g. `tab / shift+tab`),
compose them from `k.NextField.Help().Key` + `k.PrevField.Help().Key`.
For entries that mention keys NOT in `KeyMap` (e.g. `^1 / ^2 / ^3`
global tab switching), keep the literal — those are the documented
global guards.

### `cmd/cli/main.go` + `CHANGELOG.md`

- Bump `version` to `v0.55.0` (minor — `?` popup implementation changes
  even though rendered content is equivalent).
- CHANGELOG entry under `## v0.55.0 — 2026-05-22`:

  > Migrate all main-matcher shortcut dispatch in `internal/tui/update_*.go`
  > from raw `msg.String()` checks to `key.Matches` against the active
  > `KeyMap`. Rewrites the `?` popup so its per-state functions read the
  > displayed key strings from the active keymap (`binding.Help().Key`)
  > instead of duplicating literals, so a binding rename propagates to
  > the popup automatically.
  >
  > ### Usage
  >
  > No user-visible change. All previously working shortcuts continue
  > to work; the `?` popup now stays in sync with the keymap
  > definition without manual list maintenance.

### `context/progress-tracker.md`

Move Unit 14 from In Progress → Completed (visible in code). Add a
session note for the migration day.

## Dependencies

- Unit 13 (`keymap-source-of-truth`) — populates `releaseKeys()` and
  `viewPortKeys()` and writes the standards rule. Unit 14 builds on
  those bindings.

## Verify when done

- [ ] `go build ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `make build` completes; pre-commit hook runs clean.
- [ ] Smoke test:
  - [ ] `stateChoosingCommit` — `?` popup renders correctly; `ctrl+f` cycles filter modes; `pgup`/`pgdown`/`ctrl+up`/`ctrl+down` scroll the inspect panel.
  - [ ] `stateReleaseMainMenu` — same.
  - [ ] `stateReleaseChoosingCommits` — `ctrl+f` cycles filter modes; `Tab` cycles focus; `Esc` walks back.
  - [ ] `stateReleaseBuildingText` — `r`/`1`/`2`/`3` retry stages; `H` opens history; `pgup`/`pgdown` scroll focused stage; `Tab` cycles cards; `Esc` cancels/back.
  - [ ] Compose tab — `←`/`→`/`h`/`l` cycle commit type; `x`/`backspace`/`delete` clear scope and remove keypoint; `↑`/`↓` / `←`/`→` / `h`/`j`/`k`/`l` navigate keypoints and pipeline-models row; `H` opens stage history from the models row.
- [ ] Mention popup still fires on `@` (literal extraction — must stay `msg.String()`).
- [ ] `q`/`esc` still close the diff-view popup and the keybindings popup (transient inputs — also stay `msg.String()`).
- [ ] Final audit: `grep -n 'msg.String()' internal/tui/*.go` returns ONLY hits in the "NO migrar" carve-out list (mention `@`, diffview_popup close, mention_popup backspace, history_dual_panel scroll, update.go global guards, keybindings_popup close).
