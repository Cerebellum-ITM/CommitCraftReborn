# CommitCraft

Go TUI for AI-generated Git commits (Groq API). Stack: Go 1.25 + Bubble Tea v2 + SQLite.

## Structure

- `cmd/cli/main.go` — entrypoint
- `internal/tui/` — Model/Update/View, state machine with 9 states
- `internal/storage/` — SQLite, migrations in `database.go`
- `internal/api/` — Groq HTTP client
- `internal/config/` — TOML config + embed prompts

## Critical Patterns

1. **Focus returns tea.Cmd** — always capture from `update.go`, never `view.go`
2. **DB migrations** — use `applySchemaMigrations()`, never modify `createTables()`
3. **AI flow** — Ctrl+W → async tea.Cmd → 3 prompts → result msg

## Critical Transitions

- `stateWritingMessage` → AI call (Ctrl+W) → `IaCommitBuilderResultMsg` → stays in state
- `stateWritingMessage` → Enter → `createCommit()` → `stateChoosingCommit`
- ESC returns to previous state

## Config

- Global: `~/.config/CommitCraft/config.toml`
- Local: `.commitcraft.toml` (overrides global)
- API key: `~/.config/CommitCraft/.env` → `GROQ_API_KEY`
- Prompts: customizable in `~/.config/CommitCraft/prompts/`

## Charm Libraries (local paths)

Module cache: `~/go/pkg/mod/`

| Library              | Import path                       | Local path                                         |
| -------------------- | --------------------------------- | -------------------------------------------------- |
| Bubble Tea           | `charm.land/bubbletea/v2`         | `~/go/pkg/mod/charm.land/bubbletea/v2@v2.0.6`      |
| Bubbles              | `charm.land/bubbles/v2`           | `~/go/pkg/mod/charm.land/bubbles/v2@v2.1.0`        |
| Lip Gloss            | `charm.land/lipgloss/v2`          | `~/go/pkg/mod/charm.land/lipgloss/v2@v2.0.3`       |
| Glamour              | `charm.land/glamour/v2`           | `~/go/pkg/mod/charm.land/glamour/v2@v2.0.0`        |
| Log                  | `charm.land/log/v2`               | `~/go/pkg/mod/charm.land/log/v2@v2.0.0`            |
| charmbracelet/x/ansi | `github.com/charmbracelet/x/ansi` | `~/go/pkg/mod/github.com/charmbracelet/x@.../ansi` |

## Code Conventions

- All code/comments in English
- State handlers: `updateXxx(msg, model) (tea.Model, tea.Cmd)`
- Never hardcode colors — use `model.Theme`
- Every popup or on-screen key-hint line must render keys/descriptions through `theme.AppStyles().Help` (`ShortKey` for keys, `ShortDesc` for descriptions, `ShortSeparator` for the `·` divider). Never style a hint with a single flat `Foreground(theme.Muted)` call.

## Versioning

- Bump `version` in `cmd/cli/main.go` on every change (semver: patch for fixes, minor for features, major for breaking changes).

## Changelog

- Maintain `CHANGELOG.md` at the repo root. Every change that bumps the version must add an entry there.
- Write all entries in **English**, regardless of the conversation language.
- Format: `## vX.Y.Z — YYYY-MM-DD` heading, then a short summary paragraph of what changed, then a `### Usage` subsection explaining how to use any newly added or modified feature (keys, flags, config knobs). Skip the usage subsection only for pure internal refactors with no user-visible surface.
- Keep entries terse: bullet points for facts, no marketing prose.
- Newest version goes at the top.
