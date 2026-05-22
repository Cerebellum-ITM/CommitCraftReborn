# Unit 11: configure-release-popup-polish

## Goal

Polish the release configuration popup that landed in `v0.53.0` so it covers every user-visible field of `ReleaseConfig`, renders the token field correctly, matches the project's help-line style, and stops swallowing the global `Ctrl+X` quit binding.

## Context

`internal/tui/release_config_popup.go` ships five fields today (Repository, Branch, Version, Binary assets path, GH_TOKEN). Four issues surfaced after the user ran v0.53.0:

1. `AutoBuild` / `BuildTool` / `BuildTarget` are part of `ReleaseConfig` (`internal/config/types.go:67-69`) but not surfaced in the popup, so the user still has to hand-edit the TOML to enable Makefile-driven asset builds before upload. The release flow already runs `execReleaseBuild` when `AutoBuild=true`, so the configuration entry point is the only thing missing.
2. The GH_TOKEN field uses `textinput.EchoPassword`, but the live render in the screenshot shows the typed character verbatim instead of a `*` mask. Bubbles defaults `EchoCharacter` to `'*'`, so the cause is likely missing on the active `View()` (possibly because `EchoMode` is set on the local `ti` value but the assignment ordering or theme overrides reset it). Needs investigation, then a deterministic fix.
3. The footer hint is rendered with `base.Foreground(theme.FgMuted).Render(...)` — a flat muted line. The project rule (saved memory `feedback_help_theme_styles.md`) says every popup hint must render keys through `theme.AppStyles().Help` with `ShortKey` for keys, `ShortDesc` for descriptions, and `ShortSeparator` for the `·` divider.
4. The global `Ctrl+X` quit binding is suppressed while the release config popup is open (carve-out in `update.go` around line 1014). The intent was to let the version field's `Ctrl+X` decrement run, but the cost is that the user loses the muscle-memory "hard quit" shortcut from any field inside this popup. The user explicitly wants global quit to win.

## Design

### Component A — add the auto-build trio to the popup

Three new fields, in this order, **after** Binary assets path and **before** GH_TOKEN:

- `Auto build` — boolean toggle. Render the input as a `[x] auto` / `[ ] auto` switch styled as a single-character "yes/no". The simplest way: keep using `textinput.Model` but constrain the value to `"true"`/`"false"` (case-insensitive); a dedicated key (e.g. `space` while focused) flips it. Detected default = false.
- `Build tool` — string. Auto-detected default `"make"` when a `Makefile` exists at the workspace root, else `""`. Hint: "make is the only supported tool today; leave blank to disable build".
- `Build target` — string. Auto-detected default from `DetectReleaseBuild()` (new helper): grep the workspace `Makefile` for a `build_release:` target (or `build:` if `build_release` is absent). Empty string when no Makefile or no obvious target.

Update `ReleaseDetect` to carry `AutoBuildDetected bool`, `BuildToolDetected string`, `BuildTargetDetected string`. The detection logic lives in `release_config_detect.go`.

Update `ReleaseConfigSnapshot` and `newReleaseConfigPopup` so the existing values pass through. `UpdateLocalConfigRelease` gets three more parameters and writes them into the TOML.

`releaseFieldCount` grows from 5 to 8; the iota constants in `release_config_popup.go` get reordered:

```go
const (
    releaseFieldRepository = iota
    releaseFieldBranch
    releaseFieldVersion
    releaseFieldAssets
    releaseFieldAutoBuild
    releaseFieldBuildTool
    releaseFieldBuildTarget
    releaseFieldToken
    releaseFieldCount
)
```

### Component B — fix the GH_TOKEN mask

Investigate first: build a tiny test in a scratch dir that constructs a `textinput.Model` with `EchoMode = EchoPassword`, feeds it a single key, and prints `.View()`. If the output is the literal char, the bug is in our usage (likely missing `EchoCharacter` or `Cursor.SetMode`). If the output is `*`, the bug is in how the popup renders the field (we might be reading `.Value()` somewhere instead of `.View()`).

Likely fix candidates (apply once confirmed):

- Explicit `ti.EchoCharacter = '*'` right after `ti.EchoMode = textinput.EchoPassword`. Even though it's the default, an explicit set rules out a struct-copy reordering.
- Make sure the View() call in the popup uses `m.inputs[releaseFieldToken].View()` and never `m.inputs[releaseFieldToken].Value()` for display.

After the fix, when the user types `ghp_xyz` the field shows `*******`.

### Component C — help line through `theme.AppStyles().Help`

Replace the current footer:

```go
footer := muted.Render("tab/↓ next · shift+tab/↑ prev · enter save (last field) · ctrl+s save · esc cancel")
```

with the project-standard help style. Pull `help := theme.AppStyles().Help` and render each segment via `help.ShortKey.Render("tab/↓")`, `help.ShortDesc.Render("next")`, `help.ShortSeparator.Render(" · ")`. Mirror the pattern already used in `model.renderStateHelpLine`. The keys to advertise:

| Key            | Description    |
|----------------|----------------|
| `tab` / `↓`    | next field     |
| `shift+tab`    | prev field     |
| `space`        | toggle auto build (when focused on that field) |
| `enter`        | save (on last field) |
| `ctrl+s`       | save           |
| `esc`          | cancel         |
| `ctrl+x`       | quit           |

### Component D — drop the Ctrl+X carve-out

In `update.go` around line 1014, the current logic is:

```go
if msg.String() == "ctrl+x" {
    switch model.popup.(type) {
    case versionPopupModel, releaseConfigPopupModel:
        // fall through to popup routing below
    default:
        return quitWithAutodraft(model)
    }
}
```

Remove `releaseConfigPopupModel` from the carve-out list:

```go
case versionPopupModel:
    // fall through — ctrl+x is the version-decrement shortcut here.
default:
    return quitWithAutodraft(model)
```

Inside `releaseConfigPopupModel.Update`, drop the `case "ctrl+x"` decrement handler (it never gets reached after the carve-out is gone) and rely on `ctrl+a` for increment plus manual edits. Document the trade-off in the popup's footer hint (`ctrl+x` is now advertised as quit, not decrement).

The version popup (a different state, dedicated to one field) keeps the ctrl+x decrement because it's the primary editing tool there.

### Component E — auto-detect helpers

Extend `internal/tui/release_config_detect.go`:

```go
func detectMakefileTarget(pwd string) (tool string, target string)
```

- Returns `"make", ""` when a `Makefile` exists but no `build_release:`/`build:` line was found.
- Returns `"make", "build_release"` (or `"make", "build"`) when a target name is detected.
- Returns `"", ""` when no Makefile.

Implementation: read up to 200 lines of the Makefile, regex-match `^([A-Za-z0-9_]+):` against the ordered preference list (`build_release` first, then `build`, then `release`, then any line that contains `go build` in its body — last is best-effort).

## Implementation order

1. Investigate the mask bug, write the fix (component B). Quick win, low risk.
2. Add the auto-build trio: types, ReleaseDetect, popup constants, `newReleaseConfigPopup`, `Update`, `View`, `save`, and `UpdateLocalConfigRelease` (component A + E).
3. Switch footer to `theme.AppStyles().Help` (component C).
4. Drop the Ctrl+X carve-out (component D).
5. Manual smoke test against `/tmp/cc-test/repo`.

## Documentation

- Version bumps to `v0.53.1` (patch — bug fix to popup behavior + UX polish, no breaking change to the configuration surface) or `v0.54.0` if we count the new build_tool/build_target fields as new user-facing surface. **Choose `v0.54.0`** because users will see new fields after upgrading.
- CHANGELOG entry under `## v0.54.0 — <date>` combining 11 and 12 if they ship together, else stand-alone for 11. Include `### Usage` covering the new build_tool/target fields and the restored Ctrl+X behavior.

## Out of scope

- Validating that the configured build target actually exists in the Makefile before saving (out-of-band — `make` errors loudly if wrong).
- Replacing `make` with other build systems. Spec keeps the existing single-tool guard rail.
- Encrypting the .env or moving to OS keychain — still deferred.

## Verify when done

- [x] `go build ./...` passes, `go vet ./...` clean.
- [x] Open the popup on `/tmp/cc-test/repo`. Eight fields visible. Repository/Branch/Version/Assets pre-fill as before; `Auto build` defaults to `[ ] auto`; `Build tool` shows `make` when a Makefile exists; `Build target` shows the detected target.
- [x] Typing in the GH_TOKEN field shows `*` per character, not the actual letter. (Real root cause documented below — Component B was the wrong hypothesis.)
- [x] The footer hint uses theme help styles (keys highlighted, descriptions muted, `·` separator from the theme — visually identical to the bottom-of-screen help line).
- [x] Ctrl+X anywhere inside the popup quits the TUI cleanly (autodraft fires).
- [x] Save persists `auto_build`/`build_tool`/`build_target` into `.commitcraft.toml`.
- [x] No regression: existing pre-flight auto-open from the upload path still works.

## Post-implementation notes (2026-05-20 / 2026-05-22)

The shipping implementation grew beyond the original spec in four ways. Captured here so the spec stays usable as a reading entry point.

### Component B was a misdiagnosis — the "g" was the placeholder

The hypothesis was that `textinput.EchoPassword` was failing to mask. Isolation tests proved the mask worked correctly. The actual culprit was bubbles' `placeholderView`: when no `Width()` is set on the textinput, it only paints `Width()+1 == 1` runes of the placeholder. With `Placeholder = "ghp_..."` that single rune is `g`, rendered muted (unfocused) or in cursor-reverse (focused). The screenshot the user flagged was the static placeholder, not a typed/unmasked char.

**Implemented fix:** drop the placeholder for the GH_TOKEN row entirely. The hint label underneath already explains what to put there. `EchoCharacter='*'` stays as defensive insurance.

### Configured-state indicator on GH_TOKEN

Once the placeholder was gone, a new gap appeared: an empty input with no placeholder doesn't tell the user whether the token is configured. Added a render branch — when `detected.GhTokenSet == true` and the input value is empty (no typed replacement) the row paints `  ✓ stored — type to replace` (success color + muted italic suffix). As soon as the user types anything, the regular masked textinput takes over.

### In-popup list picker on Enter for build_tool / build_target

Spec didn't anticipate this affordance. After the user shipped v0.53.0 they asked for a list-picker on Enter so the auto-detected value isn't a take-it-or-leave-it. Implementation:

- New struct fields on `releaseConfigPopupModel`: `pickerActive`, `pickerField`, `pickerOptions`, `pickerIndex`, plus cached `buildToolChoices` / `buildTargetChoices` populated at popup construction time.
- New helper `openPicker(field, options)`. Enter on the build_tool or build_target field arms it; the picker grabs Up/Down/k/j, Enter commits the highlighted option, Esc cancels.
- Picker is rendered inline where the textinput would go via `renderFieldPicker` so spatial context stays put.
- New helpers in `release_config_detect.go`: `ListMakefileTargets(pwd)` (all non-phony targets in source order) and `ListBuildTools()` (just `["make"]` today; extension point for future).
- The help footer (`renderHelpFooter`) now branches on `pickerActive` and on focused field so the advertised keys match the current mode.

### +20 % sizing + nerd-font icons + palette spacing

Three smaller polish items captured here for completeness:

- `openReleaseConfigPopup` and `openChangelogConfigPopup` switched from `width/2` + `height*3/4` to `width*3/5` (floor 72, ceiling 108) + `height*9/10` so the new fields breathe.
- New `styles.Symbols` fields `ConfigureRelease` / `ConfigureChangelog` / `BuildTool` / `TokenIcon` / `BranchIcon` (with ASCII fallbacks). Both popup titles, every field label, and the four command-palette entries lead with the matching glyph.
- The command-palette delegate (`commandPaletteDelegate.Render`) joins the icon and the title with two spaces instead of one — nerd-font glyphs are wider than ASCII so a single space looked cramped.

### Footnote on Component D

The Ctrl+X carve-out for `releaseConfigPopupModel` was removed as designed. The version popup (`versionPopupModel`) keeps its carve-out because Ctrl+X is its dedicated version-decrement shortcut. Inside the release config popup, Ctrl+A still bumps the version segment under the cursor on the Version field; the Ctrl+X decrement was dropped along with the carve-out (use Ctrl+A repeatedly or edit manually).
