# CommitCraft v2 — Claude Guide

## What is this project

Go CLI/TUI tool that helps generate commit messages using Groq AI. The user selects a type, scope, and writes keypoints; the AI generates the final message. Also supports release notes generation.

## Stack

- **Go 1.25** — no web frameworks, stdlib + Charm dependencies only
- **Bubble Tea v2** (`charm.land/bubbletea/v2`) — Elm-style event loop (Model/Update/View)
- **Bubbles v2** (`charm.land/bubbles/v2`) — components: list, textarea, viewport, textinput
- **Lipgloss v2** (`charm.land/lipgloss/v2`) — styling and layout
- **SQLite** (`modernc.org/sqlite`) — local database, no CGO
- **Groq API** — direct HTTP in `internal/api/groq.go`
- **TOML** — global and local configuration

## Directory structure

```
cmd/cli/main.go              # Entrypoint: init config → DB → Model → tea.Run
internal/
  api/groq.go                # HTTP client for Groq Chat Completions
  config/
    loader.go                # Loads global + local config, resolves prompts
    types.go                 # Config structs
    prompts/                 # Embedded templates (go:embed)
  commit/types.go            # Default commit types (IMP, FIX, ADD, etc.)
  logger/logger.go           # Structured logging
  storage/
    database.go              # SQLite init, createTables, applySchemaMigrations
    queries.go               # CRUD: commits and releases
    types.go                 # Commit{} and Release{}
  tui/
    model.go                 # Central Model — all state lives here
    update.go                # Update() + per-state handlers
    view.go                  # View() — render per state
    keys.go                  # Keybindings per state
    utils.go                 # Git helpers (diff, branches, status)
    main_list.go             # Commit history list (HistoryCommitDelegate)
    commit_type_list.go      # Commit type selection list
    file_list.go             # File list for scope selection
    release_list.go          # Release history list
    release_main_menu_list.go
    list_popup.go            # Generic list popup
    delete_confirm_popup_model.go
    statusbar/               # StatusBar with spinner
    styles/                  # Theme + charmtone
```

## State machine (appState)

```
stateSettingAPIKey          → prompts for GROQ_API_KEY if not configured
stateChoosingCommit         → commit history list (main menu)
stateChoosingType           → select commit type (IMP, FIX, ADD…)
stateChoosingScope          → select affected file/directory
stateWritingMessage         → dual panel: keypoints input (left) + AI response (right)
stateEditMessage            → edit the AI-generated message
stateReleaseMainMenu        → release main menu
stateReleaseChoosingCommits → select commits for the release
stateReleaseBuildingText    → preview and edit the generated release
```

**Key transitions:**
- `stateChoosingCommit → stateChoosingType` (AddCommit key)
- `stateChoosingType → stateChoosingScope` (Enter)
- `stateChoosingScope → stateWritingMessage` (Enter)
- `stateWritingMessage` → AI call (Ctrl+W) → `IaCommitBuilderResultMsg` → stays in same state
- `stateWritingMessage` → Enter → `createCommit()` → `stateChoosingCommit`
- ESC on most states returns to the previous state

## Storage — schema

The `message_es` column stores **keypoints serialized with `\n`** (not a Spanish message). Serialization/deserialization happens in `queries.go` via `splitKeyPoints()`/`joinKeyPoints()`. Old single-line records are read as a single-element slice — no data loss.

```sql
commits (
  id INTEGER PRIMARY KEY,
  type TEXT,
  scope TEXT,
  message_es TEXT,   -- keypoints separated by \n
  message_en TEXT,   -- final AI-generated message
  workspace TEXT,    -- absolute repo path
  diff_code TEXT,    -- git diff summary
  status TEXT,       -- "completed" | "draft"
  created_at TEXT    -- RFC3339 UTC
)

releases (
  id INTEGER PRIMARY KEY,
  type TEXT,         -- "REL" | "MERGE"
  title TEXT,
  body TEXT,
  branch TEXT,
  commit_list TEXT,  -- comma-separated hashes
  version TEXT,
  workspace TEXT,
  created_at TEXT
)
```

**Migrations:** `applySchemaMigrations()` in `database.go` — adds columns via `ALTER TABLE` and ignores "duplicate column name" errors. New columns always go in that function, never in `createTables()`.

## Commit struct

```go
type Commit struct {
    ID        int
    Type      string
    Scope     string
    KeyPoints []string  // serialized as message_es in DB
    MessageEN string
    Workspace string
    Diff_code string
    Status    string    // "completed" | "draft"
    CreatedAt time.Time
}
```

## AI flow

```
Ctrl+W in stateWritingMessage
  → userInput = strings.Join(model.keyPoints, "\n")
  → callIaCommitBuilderCmd(userInput, model)  [async tea.Cmd]
    → GetStagedDiffSummary()  → SummaryPrompt → iaSummary
    → CommitBuilderPrompt     → iaCommitRawOutput
    → OutputFormatPrompt      → iaFormattedOutput
  → IaCommitBuilderResultMsg → model.commitTranslate = iaFormattedOutput
```

All prompts are user-configurable by overwriting templates in `~/.config/CommitCraft/prompts/`.

## Configuration

- **Global:** `~/.config/CommitCraft/config.toml` (auto-created)
- **Local:** `.commitcraft.toml` in the repo root (overrides global)
- **API key:** `~/.config/CommitCraft/.env` → `GROQ_API_KEY=...`

Commit types support `behavior = "append" | "replace"` to combine or replace the default types.

## Code conventions

- All code, function names, and comments must be in **English**.
- State handlers follow the pattern `updateXxx(msg tea.Msg, model *Model) (tea.Model, tea.Cmd)`.
- Async operations return a `tea.Cmd`; the result arrives as a typed `Msg`.
- Always use `model.Theme` for styling — never hardcode colors.
- Schema migrations go in `applySchemaMigrations()`, never modify `createTables()`.
- `model.keyPoints []string` is the source of truth for keypoints; `model.commitMsg` is the joined string passed to the AI.

## Key bindings (stateWritingMessage)

| Key | Action |
|-----|--------|
| `Enter` (on input) | Adds keypoint to `model.keyPoints` |
| `Ctrl+W` | Calls AI with current keypoints |
| `Enter` (with AI response ready) | Finalizes and saves the commit |
| `Ctrl+S` | Saves as draft |
| `Tab` / `Shift+Tab` | Switches focus between input and AI viewport |
| `Ctrl+E` | Enters AI response edit mode |
| `ESC` | Cancels and returns to previous state |
