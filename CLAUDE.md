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

## Code Conventions
- All code/comments in English
- State handlers: `updateXxx(msg, model) (tea.Model, tea.Cmd)`
- Never hardcode colors — use `model.Theme`
