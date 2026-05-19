# Unit 01: remove-merge-toggle

## Goal

Eliminate the cosmetic `release` ⇄ `merge` sub-toggle from the release flow.
Concretely: remove the `m` key handler in `stateReleaseChoosingCommits`,
delete the `model.releaseMode` field and its `"release"` default, drop the
`ReleaseInput.Mode` field from `internal/aiengine/release.go`, remove the
`m:release` pill rendered on the picker title bar, drop the
`"N commit(s) · <mode>"` mode suffix on the release pipeline left panel, and
update `context/architecture.md` to delete the sub-toggle paragraph and the
`RMode` node from the Mermaid flow diagram. After this unit, the release
pipeline runs identically; only the cosmetic toggle goes away.

**Out of scope (do NOT touch):**

- `model.releaseType` / `storage.Release.Type` — these persist the eventual
  classification (`"RELEASE"` / `"MERGE"`) chosen *after* the pipeline in the
  type popup (`update.go:547`). That selection stays.
- `release_dual_panel.go:147-149` — the `releaseLabel` derived from
  `r.Type` in the inspect view. That reads the persisted type, not the
  toggle, and remains valid.

## Design

No new visual elements. Only deletions:

- **Picker title pill** (`view_release.go:264-275`): drop the `· m:release`
  suffix appended after the `body / response` mode pill. The function keeps
  rendering the two-stage pill, ending at the first `render(right, …)` call.
- **Pipeline left panel footer** (`pipeline_view.go:67-79`): change footer
  from `"%d commit(s) · %s"` to `"%d commit(s)"`. Keep the `theme.Muted`
  styling — it already routes through `model.Theme`, no token changes needed.
- All key-hint lines remain rendered through `theme.AppStyles().Help`
  (`ShortKey` / `ShortDesc` / `ShortSeparator`) per the help-line invariant in
  `ui-context.md`. This unit removes one hint pair (`m` / `release|merge`),
  it does not replace it.

## Implementation

### `internal/tui/model.go`

- Delete the `releaseMode` field (line 197) and its leading 2-line comment
  (lines 195-196).
- Delete the `releaseMode: "release"` initializer in the `*Model`
  constructor (line 593).

### `internal/tui/update_release.go`

- Delete the entire `m`-key branch (lines 525-535) including its leading
  `//` comment block. Nothing replaces it; the focus / filter handlers
  immediately below stay unchanged.

### `internal/tui/view_release.go`

- In the function ending at line 276, drop lines 268-275 (the `releaseMode`
  resolution, `modeLabel`, `modeKey`, `modePill`) and change the final
  `return pill + …` to just `return pill`.

### `internal/tui/pipeline_view.go`

- Lines 72-79: remove the `mode := model.releaseMode` block and the
  empty-string fallback. Replace the footer format string
  `"%d commit(s) · %s"` with `"%d commit(s)"` and drop `mode` from the
  `fmt.Sprintf` args. Keep `count` and the `theme.Muted` foreground.

### `internal/tui/ai_pipeline.go`

- Lines 223-227: delete the `mode := model.releaseMode` block and the
  empty-string fallback. Construct `aiengine.ReleaseInput{Commits: commits}`
  with no `Mode` field.

### `internal/aiengine/release.go`

- Delete the `Mode` field from `ReleaseInput` (lines 38-41) and its
  doc comment about `Mode is informational`.
- In `RunRelease` line 104, drop the `"mode", in.Mode` key/value pair from
  the `deps.Log.Debug` call. Keep the rest of the log line.

### `context/architecture.md`

- Remove the **"Sub-toggle inside ReleaseMode"** paragraph (lines 28-30 plus
  the bridging sentence `So conceptually there are two pipelines …`) and
  collapse the operating-modes section so the table is followed directly by
  the "System Boundaries" header.
- In the **State Machine → Critical transitions** list, delete the bullet
  `_ReleaseMode:_ stateReleaseMainMenu → toggle releaseMode …`.
- In the **AI Pipelines → Release pipeline** subsection, remove the
  paragraph that begins `ReleaseInput.Mode is "release" or "merge"` (around
  line 128).
- In the Mermaid diagram, delete the line
  `RMM -- toggle --> RMode((release ⇄ merge<br/>informativo))` and the
  matching legend bullet `(((…)))` referring to the sub-toggle. The rest of
  the diagram stays.

### `CHANGELOG.md`

- Add a new top entry `## v0.50.0 — 2026-05-04` (or current date at impl
  time). Summary: removed the cosmetic release/merge toggle from the
  release picker; release pipeline behavior unchanged. No `### Usage`
  subsection — the change has no new key or flag to document, only a
  removed one. Mention the removed `m` key explicitly in the bullet so
  users who relied on it understand it's gone.

### `cmd/cli/main.go`

- Bump `version` from current to `v0.50.0` (minor bump — user-visible
  removal of a key binding).

## Dependencies

- none.

## Verify when done

- [ ] Pressing `m` in `stateReleaseChoosingCommits` does nothing (no toggle,
      no status-bar message, no panic).
- [ ] Picker title bar shows only the body/response pill — no `m:release`
      suffix.
- [ ] Release pipeline left-panel footer reads `N commit(s)` with no mode
      suffix.
- [ ] `grep -rn "releaseMode\|ReleaseInput.*Mode" internal/ cmd/` returns
      zero matches (excluding `releaseType` and `r.Type`, which are
      different concepts).
- [ ] `go build ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] Existing release run end-to-end (commit picker → pipeline → preview)
      produces the same final note as before — pipeline output is
      byte-identical for the same inputs and prompts.
- [ ] `context/architecture.md` no longer references `releaseMode`,
      `ReleaseInput.Mode`, or the `RMode` Mermaid node.
- [ ] `cmd/cli/main.go` version bumped; `CHANGELOG.md` has a new entry at
      the top in English.
- [ ] Pre-commit hook (`gofumpt → goimports-reviser → golines`) passes on
      every touched Go file.
