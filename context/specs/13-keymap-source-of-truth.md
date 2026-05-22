# Unit 13: keymap-source-of-truth

## Goal

Populate the `releaseKeys()` and `viewPortKeys()` keymap variants with the
bindings that `stateReleaseBuildingText` already dispatches via raw
`msg.String()` (the Unit 08 workaround), and write the project's "use
`key.Matches` with the active keymap as single source of truth" rule
into `context/code-standards.md`. No behavioral change in this unit —
this is the foundation Unit 14 builds on to migrate the handlers.

## Design

No visual change. The `?` popup keeps rendering whatever
`keybindings_popup.go`'s hard-coded lists return — Unit 14 will move that
to keymap introspection.

The new bindings keep the help strings consistent with `pipelineKeys()`
in `pipeline_keys.go` so when Unit 14 lands the `?` popup reads the same
copy users already see on the commit pipeline.

## Implementation

### `internal/tui/keys.go` — add `History` field to `KeyMap`

Add a `History key.Binding` field to the `KeyMap` struct, grouped near
`Help`/`Esc`. This is the binding for `H` (open the focused stage's
history popup) and didn't exist as a struct field before — it was matched
by raw string in `update_release.go:375`.

### `internal/tui/keys.go` — populate `releaseKeys()`

Add the following bindings to the existing returned `KeyMap` literal,
using the same help text style as `pipelineKeys()`:

- `Toggle` → `key.WithKeys("r")`, help `"r" / "retry pipeline"`
- `RerunStage1` → `key.WithKeys("1")`, help `"1" / "retry stage 1 (cascades)"`
- `RerunStage2` → `key.WithKeys("2")`, help `"2" / "retry stage 2 (cascades)"`
- `RerunStage3` → `key.WithKeys("3")`, help `"3" / "retry stage 3"`
- `History` → `key.WithKeys("H")`, help `"H" / "stage history"`
- `PgUp` → `key.WithKeys("pgup")`, help `"pgup" / "stage scroll up"`
- `PgDown` → `key.WithKeys("pgdown")`, help `"pgdown" / "stage scroll down"`

Leave the existing fields (Up/Down/Enter/Quit/GlobalQuit/Help/Filter/
Esc/AddCommit/NextField/PrevField/NextViewPort/SwapMode) untouched.

### `internal/tui/keys.go` — populate `viewPortKeys()`

`viewPortKeys()` becomes the active keymap when the user cycles focus to
the release-pipeline viewport (`transitions.go:271`). It must carry the
same stage-control bindings so they keep working in that focus state.
Add the same seven bindings (`Toggle`, `RerunStage1/2/3`, `History`,
`PgUp`, `PgDown`) — note that `PgUp` and `PgDown` already exist in
`viewPortKeys()`; leave them as-is.

### `internal/tui/keys.go` — extend `ShortHelp`/`FullHelp`

Add `if k.History.Enabled() { … }` blocks in both `ShortHelp()` (after
`k.Help`) and `FullHelp()` (next to the other pipeline bindings) so the
new field flows through the existing introspection helpers. The other
fields (`Toggle`, `RerunStage1/2/3`, `PgUp`, `PgDown`) are already
covered.

### `context/code-standards.md` — new "Keyboard dispatch" section

Insert after the existing "Bubble Tea (TUI)" section:

```markdown
## Keyboard Dispatch

- All shortcut dispatch goes through `key.Matches(msg, keys.X)` where
  `keys` is the state's active keymap (e.g. `releaseKeys()`,
  `mainListKeys()`, `pipelineKeys()`).
- The active keymap is the single source of truth. It MUST populate
  every field its update handler references — `key.Matches` against a
  zero-value `key.Binding{}` silently returns `false`, so a forgotten
  field looks identical to "user didn't press it" at runtime.
- `msg.String()` matching is reserved for transient, non-advertised
  input: closing a confirm popup with `q`/`esc`, extracting a literal
  character for typed input (`@` for mentions), or navigation inside an
  already-focused bubbles component that is not shown in the `?` popup.
  If a key appears in `?`, it goes through `key.Matches`.
- Global guards in `update.go` (`ctrl+x` / `ctrl+l` / `ctrl+k` /
  `ctrl+1-3`) are an exception — they run before the state's keymap and
  intentionally stay as raw string matches.
```

### `cmd/cli/main.go` + `CHANGELOG.md`

- Bump `version` to `v0.54.1` (patch — internal scaffolding, no behavior).
- Add CHANGELOG entry under `## v0.54.1 — YYYY-MM-DD`:

  > Populate `releaseKeys()` and `viewPortKeys()` with the release
  > pipeline stage controls (`r`, `1`/`2`/`3`, `H`, `pgup`/`pgdown`) that
  > were previously matched via raw `msg.String()` since Unit 08. Adds a
  > `History` field to the `KeyMap` struct and writes the project's
  > `key.Matches`-with-keymap-as-source-of-truth rule into
  > `context/code-standards.md`. Foundation for Unit 14 (handler
  > migration). No user-visible change.

  Skip the `### Usage` subsection — pure internal refactor.

## Dependencies

- none (uses existing `bubbles/key` already in the module).

## Verify when done

- [ ] `go build ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] Release pipeline shortcuts (`r`/`1`/`2`/`3`/`H`/`pgup`/`pgdown`) still
      respond exactly as before — Unit 14 will migrate the dispatch; in
      Unit 13 they remain on the `msg.String()` switch in
      `update_release.go:374`.
- [ ] `grep -n 'msg.String()' internal/tui/update_release.go` still finds
      the original block (Unit 14's job to remove it).
- [ ] `context/code-standards.md` contains the new "Keyboard Dispatch"
      section.
- [ ] `CHANGELOG.md` has the v0.54.1 entry on top; `cmd/cli/main.go`
      version constant matches.
