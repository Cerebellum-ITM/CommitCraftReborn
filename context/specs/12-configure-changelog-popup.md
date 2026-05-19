# Unit 12: configure-changelog-popup

## Goal

Mirror the v0.53.0 release configuration popup for `ChangelogConfig` so the user can toggle the optional post-pipeline changelog step and tune its inputs (file path, version-bump strategy, prompt file, prompt model) without hand-editing `.commitcraft.toml`.

## Context

`ChangelogConfig` already drives an opt-in feature: when `Enabled=true`, the commit pipeline runs a fourth AI stage that produces a CHANGELOG entry styled to match the workspace's existing file and stages it together with the commit (see `internal/changelog/`, `internal/tui/local_config.go:UpdateLocalConfigVersion` adjacent code, and the loader's `loadIaPrompts` integration).

Today the only way to turn it on is to edit `.commitcraft.toml` by hand:

```toml
[changelog]
enabled = true
path = "CHANGELOG.md"
bump_strategy = "patch"
prompt_file = "changelog.md"
prompt_model = "openai/gpt-oss-120b"
```

With Unit 10's popup precedent and the user's request, this unit adds an in-TUI surface that:

- Pre-fills `Path` from `CHANGELOG.md` if it exists in the workspace root (else falls back to `CHANGELOG`, `HISTORY.md`, `RELEASE_NOTES.md`).
- Suggests `bump_strategy` based on the workspace's recent CHANGELOG entries (heuristic — see Component C).
- Offers the model picker's selected default for `prompt_model` when blank.
- Saves through a new `UpdateLocalConfigChangelog(cfg)` helper.

## Design

### Component A — `ChangelogDetect` helper

New helpers in `internal/tui/release_config_detect.go` (rename later if it grows — the file is still the right home for filesystem detection):

```go
type ChangelogDetect struct {
    PathDetected         string // first existing path among the candidate list
    LastVersion          string // most recent ## vX.Y.Z heading, if any
    SuggestedBumpStrategy string // "patch" by default; "minor" if any of last 5 entries clearly added features
    Style                string // "keep-a-changelog" | "headings" | "unknown"
}

func DetectChangelog(pwd string) ChangelogDetect
```

Implementation:

- `PathDetected`: walk `["CHANGELOG.md", "CHANGELOG", "HISTORY.md", "RELEASE_NOTES.md"]` and return the first one that exists with `os.Stat` + `!IsDir()`. Empty when none exist.
- `LastVersion`: parse the detected file for the first `## v?\d+\.\d+\.\d+` heading. Returns the captured version string or empty.
- `SuggestedBumpStrategy`: read up to the first 30 entries (or 200 lines). If any heading immediately followed by a paragraph mentioning "added"/"new" without "fixed", suggest "minor". Otherwise "patch". (Best-effort. Spec doesn't claim perfect accuracy.)
- `Style`: detect `### Added`/`### Changed`/`### Fixed` subsections (keep-a-changelog) vs. free-form paragraphs (headings).

### Component B — popup model

New file `internal/tui/changelog_config_popup.go` mirroring `release_config_popup.go`. Same layout, same key bindings (`tab` cycles, `space` toggles `enabled`, `enter` saves on last field, `ctrl+s` saves, `esc` cancels). No `ctrl+x` carve-out; Ctrl+X always quits.

```go
const (
    changelogFieldEnabled = iota
    changelogFieldPath
    changelogFieldBumpStrategy
    changelogFieldPromptFile
    changelogFieldPromptModel
    changelogFieldCount
)

type changelogConfigPopupModel struct {
    inputs   [changelogFieldCount]textinput.Model
    labels   [changelogFieldCount]string
    hints    [changelogFieldCount]string
    focus    int
    width    int
    height   int
    theme    *styles.Theme
    detected ChangelogDetect
}
```

`Path` accepts a relative-to-workspace path. `BumpStrategy` accepts only `patch`/`minor`/`major` (validate on save; surface inline error if anything else). `PromptFile` defaults blank (built-in prompt is used). `PromptModel` shows the currently-active model as placeholder; empty means inherit.

### Component C — save helper

```go
func UpdateLocalConfigChangelog(enabled bool, path, bumpStrategy, promptFile, promptModel string) error
```

In `local_config.go`. Validates `bumpStrategy ∈ {patch, minor, major, ""}`; returns a typed error on invalid input so the popup can render an inline message instead of mutating disk.

### Component D — wire popup into the popup type switch & command palette

- Add `changelogConfigPopupModel` to the `view.go` type-switch alongside `releaseConfigPopupModel`.
- New command-palette entry `cmdConfigureChangelog = "changelog.configure"` with title "Configure changelog" and description "Toggle and tune the post-pipeline CHANGELOG entry".
- New `openChangelogConfigPopup(model *Model)` helper in `commands.go` mirroring `openReleaseConfigPopup`.

### Component E — auto-open?

The release popup auto-opens before an upload when required config is missing. There's no analogous auto-trigger for the changelog feature — it's an opt-in pipeline step. We **don't** auto-open the changelog popup. Users discover it through the command palette.

### Component F — bump version + CHANGELOG

- `cmd/cli/main.go` bumps to `v0.54.0` (same version that Unit 11 ships under — they ship together on this branch).
- CHANGELOG entry under `## v0.54.0` consolidates 11 and 12 with a `### Usage` block covering both popups.

## Out of scope

- Multi-CHANGELOG style detection beyond keep-a-changelog vs. headings. The detection is advisory.
- Actually changing how the changelog AI stage runs — only the configuration surface.
- A "preview" pane that shows what the next entry would look like. Could be a future unit.

## Verify when done

- [ ] `go build ./...` passes.
- [ ] Command palette → "Configure changelog" opens the popup with 5 fields. Defaults pre-filled (path detected, bump suggested, model placeholder).
- [ ] Toggle `enabled` with `space`; Tab cycles fields; Esc closes; Enter on last field saves.
- [ ] Saving with `bump_strategy = "weird"` shows an inline error and does not write disk.
- [ ] Saving with valid values writes the `[changelog]` table into `.commitcraft.toml` exactly once (no duplicate sections after multiple saves).
- [ ] On next `commitcraft` start, the popup re-opens pre-filled with the values just saved.
- [ ] Footer hint uses `theme.AppStyles().Help`.
- [ ] Ctrl+X quits cleanly from inside the popup.
