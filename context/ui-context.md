# UI Context

## Theme

CommitCraft is a **terminal UI**, not a browser app. All "styling" is Lip Gloss `Style` objects rendered into ANSI-painted strings. Themes live in `internal/tui/styles/` as `*Theme` structs; the active one is selected by `[tui].theme` in TOML config and exposed as `model.Theme`.

The design language is a **dark technical workspace**: near-black background, layered surfaces, violet primary accent, semantic add/del/mod colors that match diff conventions. Light mode is not supported (no theme defines `IsDark = false`).

Available themes (`internal/tui/styles/registry.go` registers them):

- `harmonized` — default. Cool indigo / sage / amber. Primary `#b79cf4`.
- `charmtone` — Charmbracelet's official palette via `github.com/charmbracelet/x/exp/charmtone`.
- `gruvbox` — warm. Primary `#fe8019`.
- `tokyonight` — cool blue. Primary `#7aa2f7`.

## Color Tokens

Every Lip Gloss `Foreground` / `Background` call in render code must read from `model.Theme`. Hex literals are only allowed inside `internal/tui/styles/<themename>.go` constructors.

Token surface defined in `internal/tui/styles/theme.go` (and populated by each theme constructor):

| Role                    | Field            | Example (`harmonized`) |
| ----------------------- | ---------------- | ---------------------- |
| Page background         | `BG`             | `#0e1016`              |
| Surface (cards, panels) | `Surface`        | `#161922`              |
| Primary text            | `FG`             | `#cfd3d8`              |
| Muted text              | `Muted`          | `#6f7480`              |
| Subtle (borders)        | `Subtle`         | `#3a3e48`              |
| Primary accent          | `Primary`        | `#b79cf4`              |
| Secondary accent        | `Secondary`      | `#7ea2d8`              |
| Success                 | `Success`        | `#86c3a7`              |
| Warning                 | `Warning`        | `#c9a265`              |
| Error                   | `Error`          | `#d07070`              |
| Diff add                | `Add`            | `#86c3a7`              |
| Diff delete             | `Del`            | `#d07070`              |
| Diff modify             | `Mod`            | `#c9a265`              |
| Scope                   | `Scope`          | `#7ea2d8`              |
| AI accent               | `AI`             | `#c5a3ff`              |
| Success dim             | `SuccessDim`     | `#689683`              |
| Accept dim              | `AcceptDim`      | `#9fa3ac`              |

Legacy aliases (`FgBase`, `FgMuted`, `FgHalfMuted`, `FgSubtle`, `BgOverlay`, `Input`, `Output`, etc.) are filled by `Theme.fillLegacy()` — keep using them where existing code does, but new code should prefer the canonical names above.

## Help-line Style Ladder (Critical)

Every popup and on-screen key-hint line **must** render through `theme.AppStyles().Help`:

- `ShortKey` — the key glyph (e.g., `^W`)
- `ShortDesc` — the description (e.g., `generate`)
- `ShortSeparator` — the `·` divider between pairs

A flat `lipgloss.NewStyle().Foreground(theme.Muted).Render(...)` for hint lines is a bug — fix it; do not replicate.

## Commit-Type Palettes

A separate four-token palette per commit type (e.g., `[FEAT]`, `[FIX]`, `[STYLE]`):

| Token     | Role                            |
| --------- | ------------------------------- |
| `BgBlock` | Background of the `[TAG]` block |
| `FgBlock` | Foreground of the tag glyph     |
| `BgMsg`   | Background of the message       |
| `FgMsg`   | Foreground of the message       |

Defaults live in `internal/tui/styles/commit_type_palette.go`; user overrides come from `[commit_types.types]` in TOML and are wired in via `RegisterCustomCommitTypePalettes` (`cmd/cli/main.go:184`).

## Typography & Sizing

A TUI inherits its fonts from the terminal. CommitCraft does *not* select a font. The only typographic concern is whether **Nerd Fonts glyphs** can render — toggled by `[tui].use_nerd_fonts` in config, branched on by `Theme.applySymbols(useNerdFont)` to swap the symbol table.

Width constraints follow **terminal columns**, not pixels. Layout helpers in `internal/tui/view_borders.go` and the various `view_*.go` files measure in cell counts.

> **Lip Gloss `Width` footgun**: `Style.Width(W)` wraps content at `W − borderSize`. When applying a bordered frame, pass *total* width to the frame, not inner width. (Saved memory: `feedback_lipgloss_width_gotcha`.)

## Border Style

- Default border: Lip Gloss `RoundedBorder()` for popups and major panels.
- Subtle border color: `theme.Subtle`.
- Active/focused border: `theme.Primary` (or theme-specific accent).

## Layout Patterns

- **Writing screen** (`view_writing.go`): two-pane horizontal split — left "Your input", right "AI suggestion". Bottom row: status bar + keybinding hints.
- **Pipeline screen** (`pipeline_view.go`): vertical stack of stage cards (body / title / format / changelog) with per-stage status, output preview, and history controls.
- **History screen** (`history_view.go`, `history_dual_panel.go`): list-left, detail-right dual panel. Filter bar on top (`history_filter_bar.go`); mode bar (drafts vs commits) on top (`history_mode_bar.go`).
- **Output screen** (`view_output.go`): scrollable preview of the final commit message with copy / commit / discard hints.
- **Release screens** (`view_release.go`, `release_dual_panel.go`, etc.): commit picker → AI changelog refinement → build/upload status.
- **Popups**: centered overlay rendered on top of the current view (`popup_helpers.go`). Common popups: scope, type, model picker, command palette, keybindings, version, logs, edit message, mention, tag picker, tag palette, list, delete confirm, diff view, config, stage history.
- **Status bar** (`internal/tui/statusbar/`): bottom strip with mode indicator + active hints.

## Symbol Tables

`Theme.applySymbols(useNerdFont)` populates `AppSymbols()`. When Nerd Fonts is off, glyphs degrade to ASCII / unicode-safe fallbacks. New components must read symbols from `theme.AppSymbols().X`, never hardcode glyph runes.

## Component Library

CommitCraft does not use a "component library" in the React sense. Reusable building blocks are bare-Bubbles primitives wrapped with project conventions:

- `bubbles/list` — wrapped per popup (e.g., `commit_type_list.go`, `release_list.go`).
- `bubbles/textarea` — input fields.
- `bubbles/viewport` — scrollable panes.
- `bubbles/spinner` — pipeline activity.

Add new shared widgets as their own file under `internal/tui/`, exposed as a small struct + render method, and reuse from feature-specific `view_*.go` files.
