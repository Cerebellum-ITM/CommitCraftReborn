# Changelog

All notable changes to CommitCraft are documented here. Newest version on top.

## v0.9.4 — 2026-04-26

Status bar redesigned around a flat two-pill `TYPE  MESSAGE` scheme with a fixed dark palette per level.

- `WritingStatusBar.Render` now emits a label pill (filled, bold, `Padding(0,1)`) immediately followed by a message body in a darker shade of the same hue family. The right side keeps the version + `⌘ CommitCraft` mark in a muted grey context style; the spinner sits to its left when running.
- The level palette is hardcoded so it stays consistent across themes (the previous theme-derived backgrounds clashed with the rest of the UI). Each level has a tailored pair of `pill` + `msg` styles: INFO (blue), OK (green), WARN (amber), ERROR (red), AI (purple), RUN (teal), DEBUG (slate).
- New levels added: `LevelAI`, `LevelRun`, `LevelDebug`. Existing levels keep their constant names but render the canonical labels: `LevelSuccess` → `OK`, `LevelWarning` → `WARN`, `LevelFatal` → shares the `ERROR` rendering. No callsites needed to change.
- Two new package helpers exported for ad-hoc rendering: `statusbar.RenderStatus(level, msg)` and `statusbar.RenderStatusFull(level, msg, ctx, width)`.

### Usage

- Updating the bar is unchanged: `model.WritingStatusBar.Level = statusbar.LevelInfo` and `model.WritingStatusBar.Content = "..."`. Or call `ShowMessageForDuration("...", level, dur)` for transient messages.
- Pick the level by intent — never invent new ones:
  - `LevelInfo`    · neutral hints, ready / idle states.
  - `LevelSuccess` · successful completion.
  - `LevelWarning` · recoverable issue.
  - `LevelError`   · failure that blocks the user.
  - `LevelFatal`   · unrecoverable error.
  - `LevelAI`      · AI / model activity.
  - `LevelRun`     · long-running op in progress.
  - `LevelDebug`   · verbose-only trace.
- For ad-hoc one-shot rendering anywhere in the TUI (popups, secondary panels) use `statusbar.RenderStatus(level, "msg")`. Use `statusbar.RenderStatusFull(level, msg, ctx, width)` when you also need a right-aligned metadata column (e.g. token counts, latencies).

## v0.9.3 — 2026-04-26

Promoted the status bar to the very top of the TUI and gave info messages their own theme-aware color.

- `WritingStatusBar` is now the first row rendered in every state. It used to live inside each state's `mainContent`; it is now stacked at the top of the global view, followed by a blank line, then the tab bar, then the main content.
- A blank vertical line separates the status bar from the tab bar so the two strips read as distinct surfaces.
- Info-level messages no longer share the muted blur background with the rest of the bar. Both the `INFO` prefix block and the message body now render with `theme.Info` as background and `theme.BG` as foreground, matching the styling pattern of Success / Warning / Error / Fatal. Because `theme.Info` derives from each theme's `Secondary` color, every theme produces a different info hue (charmtone Dolly, harmonized blue, tokyonight purple, gruvbox aqua).

### Usage

- No new keys.
- Theme authors: info message styling reads from `theme.Info` (background) and `theme.BG` (foreground). Override `Secondary` (or directly set the legacy `Info`) in a theme constructor to change the info color for that theme.

## v0.9.2 — 2026-04-26

Removed the duplicated top breadcrumb header, refreshed the status-bar logo, and reworked the tab-bar selection cue.

- The standalone top header (the `commitCraft / <tab> · <pwd>` breadcrumb with the green app pill on the right) is gone. The `WritingStatusBar` (the bar with the `INFO` / `WARN` / `ERROR` level prefix that already lives at the top of every screen) is now the single status surface.
- The `CommitCraft` logo embedded inside that status bar now reads `⌘ CommitCraft` with `theme.Primary` as background and `theme.BG` as foreground (instead of the legacy `theme.Logo` background). The Mac command symbol is preserved as part of the brand mark.
- Tab bar redesign: the visual `│ History │ Compose │ Pipeline │` separators are kept, but the two `│`s flanking the active tab now render in `theme.Primary` (bold) while the rest stay in `theme.Subtle`. The active tab's label uses `theme.FG` bold; inactive labels use `theme.Muted`. This produces a clearly framed selection without relying on background fills.

### Usage

- No new keybindings.
- Theme authors: the active-tab cue reads from `theme.Primary` (active separators + bold) and `theme.Subtle` (idle separators). The status-bar logo reads from `theme.Primary` (background) and `theme.BG` (foreground).

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
