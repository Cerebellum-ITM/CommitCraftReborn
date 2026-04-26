# Changelog

All notable changes to CommitCraft are documented here. Newest version on top.

## v0.9.1 — 2026-04-26

Streamlined the new-commit flow and fixed the commit-type pills not loading.

- Tab order now reads **History · Compose · Pipeline** (`^1` opens History, `^2` Compose, `^3` Pipeline). History is the entry point, so it sits first.
- Pressing `n` from the history list jumps **directly into the compose view**. The legacy fullscreen "choose type" and "choose scope" screens are no longer launched from the main flow (they remain reachable from inside compose via popups).
- Bug fix: `model.finalCommitTypes` was never populated, so the type-pills row always rendered as `(no commit types configured)` even when `.commitcraft.toml` had types. The slice is now wired in `NewModel`, and the first configured type is preselected.
- The bottom hint bar is now state-aware in **every** screen, not only compose. Each state shows the keys relevant to it (history list, release menu, scope picker, edit, reword, api-key, pipeline).
- Esc from compose returns to the history list directly (instead of going back through the deprecated scope picker).

### Usage

- `n` (from history list) — start a new commit. Lands in compose with the first commit type from `.commitcraft.toml` already selected.
- Inside compose, switch sections with `Tab` / `Shift+Tab`. The hint bar at the bottom updates to show the keys valid for that section.
- `^T` opens the commit-type popup, `^P` opens the file-picker for scope (these replace the old fullscreen screens).
- `^1` / `^2` / `^3` switch top-level tabs.
- `Esc` from compose returns to History.

## v0.9.0 — 2026-04 (earlier)

Major TUI redesign and internal package restructure.

- Extracted `internal/git/` package: all git helpers (`GetCurrentGitBranch`, `GetGitDiffStat`, `GetStagedDiffSummary`, `GetGitDiffNameStatus`, `GetStagedFileDiff`, `GetCommitDiffSummary`, `ResolveCommitHash`, `GetLastGitTag`, `RewordCommit`, `StatusData`, `GetAllGitStatusData`, `GetCommitGitStatusData`) live there now. `internal/tui` no longer shells out to git directly.
- Split the three monolith files (`update.go`, `view.go`, `utils.go`) into ~30 cohesive files: `update_writing.go`, `update_commit.go`, `update_release.go`, `update_reword.go`, `update_apikey.go`, `view_writing.go`, `view_release.go`, `view_borders.go`, `compose_sections.go`, `compose_panel.go`, `compose_status.go`, `transitions.go`, `commands.go`, `ai_pipeline.go`, `tools.go`, `format.go`, `local_config.go`, `release_upload.go`, `file_list_helpers.go`, `popup_helpers.go`, etc.
- New compose layout: header breadcrumb, persistent tab bar (Compose / History / Pipeline), titled left/right panels (`summary` + `ai suggestion`) with rounded borders, sectioned left panel (commit type pills, scope chip, summary textarea, key points list, pipeline models), bottom info bar with char counter + progress bar, focus-aware hint line.
- Theme schema rewrite (BG / Surface / FG / Muted / Subtle / Primary / Secondary / Success / Warning / Error / Add / Del / Mod / Scope). Legacy fields auto-derived via `fillLegacy()`. Theme registry: charmtone, harmonized, tokyonight, gruvbox-dark.
- Commit type and scope are now editable in-place inside compose: `^T` opens the type popup, `^P` opens the scope file-picker. Scope is single-value; the chip is replaced when a new path is picked.
- Reword flow (`-w <hash>`): startup popup asks "Reword as commit" or "Reword as release". The "Reword as release" path enters the regular `-r` flow.
- Real-time log popup: `^L` toggles a viewport that streams the charm-log channel.
- `@` inside the summary textarea opens a file-mention popup filtered by the staged-diff name list. Selecting a file inserts its path at the cursor.
- Diff popup is syntax-highlighted via Chroma.
- Pipeline tab placeholder (still being rebuilt against the new layout).

### Usage

- `commit_craft` — normal mode (history → compose).
- `commit_craft -r` — release mode (release main menu).
- `commit_craft -o` — direct stdout output of the chosen commit message (no popup menu).
- `commit_craft -w <hash>` — reword mode. Resolves the hash, then asks whether to reword as a regular commit or as a release.
- Inside compose: `Tab` cycles sections, `^W` runs the AI pipeline, `^A` adds a key point, `@` mentions a file, `^E` edits the AI reply, `Enter` accepts, `^S` saves a draft.
- `^L` toggles logs popup, `^V` opens the version editor (writes `[release] version` in `.commitcraft.toml`).
- `^1` / `^2` / `^3` switch top-level tabs.
- Themes: set `[tui] theme = "charmtone" | "harmonized" | "tokyonight" | "gruvbox-dark"` in config.
