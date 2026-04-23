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
