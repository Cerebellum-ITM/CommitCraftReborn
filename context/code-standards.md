# Code Standards

## General

- All code, identifiers, and comments in **English** — even when conversations with the user happen in Spanish.
- Default to **no comments**. Only add a comment when the *why* is non-obvious (a constraint, a workaround, a footgun like the lazygit empty-amend case in `main.go:142-154`).
- Keep modules small and single-purpose. New TUI features go in their own file (`update_<feature>.go`, `view_<feature>.go`, `<feature>_popup.go`), not appended to `update.go`/`view.go`.
- Fix root causes; do not layer workarounds.
- All code is formatted by `gofumpt` → `goimports-reviser` → `golines` (chain installed via `make install-fmt`, enforced by the `pre-commit` hook from `make install-hooks`).

## Go

- Module path: `commit_craft_reborn`.
- Toolchain: Go 1.25.
- Use `pkg/errors` for wrapping where context matters; stdlib `errors.Is/As` for control flow.
- No CGO — keep using `modernc.org/sqlite` so cross-compilation stays trivial.
- Prefer typed structs over `map[string]any` for anything that crosses a package boundary.
- Validate external input (CLI args, JSON request bodies, file content) at the package edge before trusting it.

## Bubble Tea (TUI)

- State handlers are named `updateXxx(msg, model) (tea.Model, tea.Cmd)`. Capture the returned `tea.Cmd` and return it from `Update`; **never call focus/blur logic from `View`**.
- `View` is pure render — no DB calls, no side effects, no I/O.
- Async work returns a `tea.Cmd` that emits a typed result message (e.g., `IaCommitBuilderResultMsg`). Never block the Bubble Tea event loop.
- New popup → its own file (`<feature>_popup.go`) plus a model field on `Model`. Wire show/hide through `transitions.go` patterns.
- Keep state transitions explicit: when adding a new state, declare it in `model.go`, add an entry to `pipelinePresetTitles` if AI-related, and document the transition in `architecture.md`.

## Styling (Lip Gloss)

- **Never hardcode colors in render code.** Always read from `model.Theme` (e.g., `theme.Primary`, `theme.AppStyles().Help.ShortKey`). Hex literals are only allowed inside `internal/tui/styles/*.go` theme constructors.
- **Help-line and popup key hints** must use `theme.AppStyles().Help`: `ShortKey` for keys, `ShortDesc` for descriptions, `ShortSeparator` (`·`) between pairs. A flat `Foreground(theme.Muted)` for hint lines is a bug — fix it, don't replicate.
- **Lip Gloss `Width` footgun**: `Style.Width(W)` wraps content at `W − borderSize`. When applying a bordered frame, pass *total* width to the frame, not inner width.
- All commit-type colors flow through `internal/tui/styles/commit_type_palette.go` — populated from `[commit_types.types]` TOML entries via `RegisterCustomCommitTypePalettes`.

## Storage

- **Never modify `createTables`** — it represents the original schema. New columns/tables go through `applySchemaMigrations` in `internal/storage/database.go`, which must be idempotent.
- All queries belong in `internal/storage/queries.go` as methods on `*DB`. No raw `db.Exec` calls outside the storage package.
- Persist large generated text (commit bodies, AI outputs) in SQLite — fine here, the DB is local-only and small. Don't add filesystem blobs unless asked.

## Headless CLI (`internal/cli/ai/`)

- Every subcommand emits JSON. Use `printCommitJSON` for success and `printErrorJSON(code, msg)` for errors — never plain `fmt.Println` of mixed content.
- Subcommands must not import `internal/tui`. The CLI surface and the TUI surface share `aiengine`, `storage`, `git`, `config` — that's the boundary.
- Add new subcommands by:
  1. New file `internal/cli/ai/<name>.go` with one entry function.
  2. New `case "<name>":` in `Dispatch` (`ai.go`).
  3. Update `--help` text in the help case.

## Config & Prompts

- Defaults: embedded `internal/config/prompts/*.prompt.tmpl` via `go:embed`.
- User overrides: `~/.config/CommitCraft/prompts/*.prompt.tmpl` — loader checks user dir first, falls back to embed.
- Local repo config (`.commitcraft.toml`) **overrides** global. Resolution lives in `config.LoadConfigs` and the `Resolve*Config` helpers.
- Never log the API key or `GH_TOKEN`.

## Versioning & Changelog

- Bump `version` in `cmd/cli/main.go:24` on every change. Semver: patch for fixes, minor for features, major for breaking.
- Add a `CHANGELOG.md` entry at the top in English: `## vX.Y.Z — YYYY-MM-DD`, summary paragraph, then `### Usage` subsection if the surface changed (keys, flags, config).
- Both bump + entry happen on every iteration, not only the first one of a feature.

## File Organization

- `cmd/cli/` — entrypoint only.
- `internal/tui/` — Bubble Tea app. Split files by feature; `update.go` and `view.go` are dispatchers, not catch-alls.
- `internal/tui/styles/` — themes and the help-style ladder. Hex literals allowed here.
- `internal/aiengine/` — pure pipeline orchestration. No `tea.Cmd` types.
- `internal/storage/` — SQLite. Migrations through `applySchemaMigrations` only.
- `internal/api/` — Groq HTTP + rate-limit cache.
- `internal/git/` — shells out to `git`. Add new git operations here, not inline in TUI handlers.
- `internal/cli/ai/` — headless subcommands.
- `internal/config/` — TOML, `.env`, embed prompts, palette resolution.
- `internal/commit/` — commit-type catalog and final-message formatting helpers.
- `internal/changelog/` — `CHANGELOG.md` reading/writing.
- `internal/logger/` — logger setup.
