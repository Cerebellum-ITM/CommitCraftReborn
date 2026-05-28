# Unit 21: tui-release-source-badge

## Goal

Render the `AI` / `UI` pill badge on the TUI's Releases view, mirroring
the one that already appears on every row of the commits History tab.
Closes the visual half of the user's v0.61.0 request (storage
realignment + visible source provenance).

Pure TUI work. No Groq, no DB writes, no CLI surface change. Just
read `releases.source` (added in unit 20) and pipe it through the
existing `sourcePillStyle` helper.

## Design

The commits History tab already solved this. The styling lives in
`internal/tui/main_list.go:73-87` as `sourcePillStyle(source string)
(lipgloss.Style, string)` — returns the right colors and the literal
label (`"AI"` or `"UI"`). The same helper handles the release row.

Color contract (unchanged from commits):

- `source == "ai"`: background `#3e3268` (purple), foreground
  `#e9e0ff`, label `"AI"`.
- anything else (treat as `"tui"`): the existing fallback branch —
  label `"UI"`, mid-saturation green.

The release rows in the History/Releases view need the same pill
inserted at the same relative position (right side of the row,
between the type chip and the date). Look at
`internal/tui/main_list.go:147` for the call shape and at the
delegate that renders release rows for the insertion point.

## Implementation

### Find the release list delegate

The release list rendering lives in `internal/tui/release_list.go`
(per the `Glob` output during unit 20 planning). Open it and locate
the `Render` (or `Render(w io.Writer, m list.Model, index int, item
list.Item)`) function that produces a single row.

If `release_list.go` doesn't have it, check
`internal/tui/release_main_menu_list.go` — the release main menu is
where the user picks an existing release to inspect.

### Wire the badge

In the row renderer:

1. The release item carries the `Source` value from
   `storage.Release.Source` (verify the item struct includes it; if
   not, add it and have the list constructor populate it from the
   `[]storage.Release` slice).
2. Call `srcStyle, srcLabel := sourcePillStyle(item.Source)`.
3. Render the pill and concatenate into the row at the same relative
   position commits use.

### Defaults & legacy rows

- TUI-created releases that predate v0.61.0 have `source = ""`
  in-memory (the migration set them to `'tui'` on disk, so once
  loaded they read `"tui"`). `sourcePillStyle` already treats empty
  / unknown as `"UI"`, so legacy rows render correctly without
  special handling.
- For unit-tests / fixtures that don't set Source, the default
  branch produces `"UI"` — same as the commits side.

### No public API change

`sourcePillStyle` stays unexported. No changes to `commitJSON`, no
changes to the headless CLI. The next session ships this with a
single TUI render commit (+ paired skill update is **not** needed —
the skill never mentions the TUI badge).

### Version bump

- `cmd/cli/main.go`: bump to `v0.62.0` (minor — user-visible TUI
  change).
- `CHANGELOG.md`: short entry. Example body:

  > Render the `AI` / `UI` source pill on the Releases view, mirroring
  > what already exists on commits. Closes the visual half of v0.61.0's
  > storage realignment — TUI-created releases keep showing as `UI`,
  > drafts produced by `commitcraft ai release` / `ai merge` now show
  > the `AI` pill instead of going unmarked. Pure rendering change;
  > the underlying `releases.source` column was added in v0.61.0.

## Dependencies

- Unit 20 (`releases.source` migration) — required and already shipped.
- `internal/tui/main_list.go::sourcePillStyle` — already implemented;
  this unit only consumes it.

## Verify when done

- [ ] `go build ./...` + `go vet ./...` clean.
- [ ] Smoke test in the TUI:
  - [ ] Launch `commitcraft` against a workspace that has
        TUI-created releases (legacy) and at least one headless
        `ai release` / `ai merge` draft.
  - [ ] Releases view renders: legacy rows show the `UI` pill, new
        headless drafts show the `AI` pill.
  - [ ] Pill colors and position match the commits view exactly.
- [ ] No regressions in the commits History tab (the helper is
      shared; touching its callers shouldn't change anything else).
- [ ] Update `context/progress-tracker.md` to mark unit 21 complete.

## Out of scope

- Filtering the releases view by `source` (could be a follow-up).
- Showing the badge in the headless `ai list` text output (the JSON
  already carries `source`; no styling needed).
- Migrating the orphan release-shaped rows still living in `commits`
  (units 17-19 legacy) — those stay in their old home; the badge
  doesn't apply because they're commits, not releases.

## Notes for the next session

The branch is `feat/agent-cli-improvements`. After this unit, the
branch is complete enough to merge to `main` using `ai merge`
itself — that would be a satisfying dogfood. If you want to land
the remaining umbrella items first (3, 5, 6), do those before
merging; otherwise this is a good cut point.
