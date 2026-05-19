# CommitCraft

## Overview

CommitCraft is a terminal UI (TUI) for Git workflows built in Go with Bubble Tea v2. It uses Groq's LLM API to turn staged diffs into well-formatted conventional commit messages, manages drafts in a local SQLite database, and supports a release-mode flow that builds binaries, drafts changelog entries with AI, and uploads GitHub releases. It also exposes a headless `commitcraft ai ...` subcommand surface so external agents can drive the same engine without the TUI.

## Goals

1. Generate conventional commits (`[TYPE](scope): subject` + body) from staged diffs through a 3-stage AI pipeline (body → title → format).
2. Let the user review, edit, save as draft, retry per stage, and switch models without leaving the TUI.
3. Provide a release flow that builds binaries, refines `CHANGELOG.md` with AI from selected commits, and uploads to a GitHub release.
4. Expose every AI capability also as headless JSON-output subcommands (`commitcraft ai generate|edit|regenerate|...`) for agent integration.
5. Stay launchable from external tools (lazygit, shell aliases) — including direct reword mode (`-w <hash>`) and direct stdout mode (`-o`).

## Core User Flow

### Commit mode (default)

1. User stages files (`git add`) and runs `commitcraft`.
2. TUI loads in `stateWritingMessage`: left panel for user input, right for AI suggestion.
3. User types a hint and presses **Ctrl+W** → AI pipeline runs (body → title → format) in `statePipeline`, hitting Groq via `internal/api`.
4. Per-stage outputs are previewed; user can retry a stage, switch models, or accept.
5. **Enter** in `stateWritingMessage` → `createCommit()` persists a draft commit row in SQLite and moves to `stateChoosingCommit`.
6. From there: pick scope (`stateChoosingScope`) and type (`stateChoosingType`), confirm (`stateConfirming`), see output (`stateOutput`), done.
7. Final commit is executed via `git commit -m`, or printed to stdout when launched with `-o`.

### Reword mode (`-w <hash>`)

1. Launches in reword flow on a target commit; on completion runs `git commit --amend` with the new message via `internal/git.RewordCommit`.
2. If the user exits without producing a final message, prints "Reword cancelled — commit X left unchanged" to stderr and exits 0 (so lazygit's status line stays clean).

### Release mode (`-r`)

1. Launches in `stateReleaseMainMenu`; user picks commits to include (`stateReleaseChoosingCommits`).
2. AI refines a `CHANGELOG.md` entry from selected commits in `stateReleaseBuildingText`.
3. Build target runs via Make; binaries land in `bin/`; release is uploaded to GitHub.

### Headless mode (`commitcraft ai ...`)

1. First positional arg `ai` short-circuits the TUI bootstrap (see `cmd/cli/main.go:31-33`).
2. Dispatched in `internal/cli/ai/ai.go` to subcommands: `generate`, `regenerate`, `edit`, `show`, `list`, `promote`, `list-tags`, `list-addable-tags`, `add-tag`, `stage-partial`.
3. All subcommands print structured JSON (success or error envelope), reusing the same `aiengine` and `storage` packages as the TUI.

## Features

### Commit generation

- 3-stage AI pipeline (body → title → format) driven from `stateWritingMessage` via Ctrl+W.
- Per-stage retry, history (last N outputs), and live model picker.
- Customizable prompts in `~/.config/CommitCraft/prompts/` (override the embedded ones in `internal/config/prompts/`).
- Customizable commit types with palettes (`bg_block`, `fg_block`, `bg_msg`, `fg_msg`) via TOML — `behavior = "append"|"replace"`.
- Mention-completion in the input panel for tags/files (`mention_popup.go`).

### Drafts & history

- Drafts auto-saved on Ctrl+S and on TUI exit (autodraft).
- `Ctrl+D` in main list toggles drafts-only view.
- Commit/draft history in SQLite with full per-stage AI call records (model, tokens, timing).

### Release flow

- Choose commits → AI changelog refinement → build (Make) → upload (GitHub).
- Per-release commit picker with dual-panel diff view.

### Headless CLI

- `commitcraft ai generate|regenerate|edit|show|list|promote|list-tags|...` for agent-driven flows.
- JSON envelopes for both success and error.

### Theming

- Multiple themes registered: `harmonized` (default), `charmtone`, `gruvbox`, `tokyonight`.
- All UI styling routed through `model.Theme` — no raw hex in render code.
- Optional Nerd Fonts toggle for icons.

## Scope

### In Scope

- Single-repo, single-user TUI workflow.
- Groq API as the only LLM backend.
- SQLite for local persistence (commits, drafts, AI calls, rate-limits, models cache).
- macOS, Linux, Windows builds (Go cross-compile).
- `git commit` and `git commit --amend` (reword) only.

### Out of Scope

- Multi-repo orchestration.
- LLM backends other than Groq.
- `git push` — CommitCraft never pushes (also enforced by `commitcraft` skill rule globally).
- Server/cloud component — everything runs locally.
- Hosted GUI; only TUI + headless CLI.

## Success Criteria

1. From staged changes, a user can produce and execute a final commit through the TUI in under 30 seconds (happy path).
2. The headless `commitcraft ai generate` subcommand produces the same commit message as the TUI for the same staged diff.
3. `-w <hash>` rewords cleanly when launched from lazygit, and exits 0 with a stderr notice when cancelled.
4. The release mode produces a `CHANGELOG.md` entry, builds binaries in `bin/`, and uploads them to a GitHub release without manual intervention beyond commit selection.
5. _TBD: confirm with user — explicit performance/quality bars beyond "it works"._
