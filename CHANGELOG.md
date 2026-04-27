# Changelog

All notable changes to CommitCraft are documented here. Newest version on top.

## v0.10.7 — 2026-04-26

When loading a commit (or draft) from history with `e` / `Enter`, the changed-files list and per-file diff are now sourced from the DB-persisted `Diff_code` instead of the live `git diff --staged`, which is unrelated to the historical commit.

- New helpers in `internal/tui/pipeline_files.go`: `parseDbDiff` splits the persisted Diff_code blob (`=== <path> ===` blocks produced by `git.GetStagedDiffSummary`) into items, per-file numstats, and per-file diff bodies; `loadPipelineFilesFromDb` swaps them into `pipelineDiffList`, `pipeline.numstat`, and a new `model.dbFileDiffs` cache.
- `setDiffFromSelectedFile` now reads from `model.dbFileDiffs` when `useDbCommmit` is true (otherwise unchanged: live staged diff).
- `pipelineStartFullRun` skips `refreshPipelineNumstat` / `applyPipelineFilesDelegate` when `useDbCommmit` is true so the historical files list isn't overwritten by the working-tree state.
- The "edit historical commit" path (`update_commit.go::EditIaCommit`) and the "continue draft" path (`Enter` on a `draft`-status item) both call `loadPipelineFilesFromDb` and set `useDbCommmit = true` for consistency.

### Usage

Pick a commit or draft from the main list and press `e` (edit) or `Enter` (drafts). Open the Pipeline tab (`Ctrl+3`): the changed-files panel now lists exactly the files captured when the commit was generated, with the same `+N -M` counts and per-file diff content stored in the DB.

## v0.10.6 — 2026-04-26

Fix the trailing `…` that appeared on every row of the compose "summary" panel after running the AI flow.

- Root cause was in `internal/tui/compose_sections.go::renderComposeKeypointsArea`: the spacer between text and `×` had a `max(1, …)` floor, so any key point whose text was wider than `width − 3` columns produced a row of `width + 1` columns, which `renderTitledPanel` (`compose_panel.go:122`) then truncated with `…`. With `innerLeftW ≈ 0.45*model.width − 4`, this fired for ~28+ char key points on a standard 80-col terminal, which is why it looked like "all rows" once the user populated the panel before pressing `Ctrl+W`.
- Fix: pre-truncate each key point's text to `width − 4` with `ansi.Truncate(..., "…")` so the natural spacer stays ≥ 1 and the row never overflows the panel. Same guard added to the section header (`label … counter`) so it can never push the row past `width` either.

## v0.10.5 — 2026-04-26

Key points are now also mandatory before any AI request, alongside the scope guard introduced in v0.10.4.

- **Compose tab (`Ctrl+W`).** After flushing the current input into `model.keyPoints`, the handler in `internal/tui/update_writing.go` checks `len(model.keyPoints) == 0` and surfaces `"At least one key point is required before requesting the AI."` in `WritingStatusBar` at `LevelError`.
- **Pipeline tab (`r`, `1`/`2`/`3`).** Same guard added to `pipelineStartFullRun` and `pipelineRetryStage` in `internal/tui/pipeline_update.go`, after the scope check.

### Usage

Before pressing `Ctrl+W` (compose) or `r` / stage retries (pipeline), make sure you have at least one scope and at least one key point. Either is missing and the top status bar shows the red `ERROR` pill explaining what to add.

## v0.10.4 — 2026-04-26

Scope is now mandatory before any AI request. Triggering generation without a scope short-circuits the call and surfaces an error in the top status bar.

- **Compose tab (`Ctrl+W`).** `CreateIaCommit` handler in `internal/tui/update_writing.go` now checks `len(model.commitScopes) == 0` and writes `"Scope is required before requesting the AI. Add at least one scope."` to `WritingStatusBar` at `LevelError`, returning before the spinner / API command starts.
- **Pipeline tab (`r`, `1`/`2`/`3` retries).** Same guard added to `pipelineStartFullRun` and `pipelineRetryStage` in `internal/tui/pipeline_update.go`. The two-pill ERROR style from `internal/tui/statusbar/statusbar.go` is what the user sees.

### Usage

Add a scope (focus the scope section in compose, press `e` / `Enter` to pick one) before pressing `Ctrl+W` or starting/retrying a Pipeline stage. If you forget, the top status bar will show the red `ERROR` pill telling you to add one.

## v0.10.3 — 2026-04-26

Three Pipeline-tab fixes covering surface size, diff visibility, and the final commit card content.

- **Full terminal surface.** The shared `availableWidthForMainContent` / `availableHeightForMainContent` calc in `view.go` was double-subtracting horizontal padding (the `appStyle` it accounts for is never applied to mainContent) and shaving 20% off the height for unclear historical reasons. The Pipeline tab now bypasses both: it receives `model.width` directly and a height equal to `model.height − statusBar − tabBar − help − VerticalSpace`, so the right panel actually spans the full terminal width and stretches all the way to the help line.
- **Diff sub-block always renders.** Layout math in `renderPipelinePanel` was reserving stage card heights first (including focused-stage growth) and *then* trying to fit the diff with the leftover, which collapsed to 0 once the final-commit card appeared. Order reversed: stages-at-default + `DiffMinHeight` are reserved up front; only the *leftover* is spent on focused-stage growth. Diff now keeps a guaranteed floor (default 6 rows) even after the AI flow finishes.
- **Final card shows the full assembled commit.** `renderFinalCommitCard` was previously rendering only the first line of `commitTranslate` (just the title from stage 3). It now wraps the full `title\n\nbody` into a multi-line viewport sized by `computeFinalBodyRows` (3-8 rows depending on body length), with the title bolded in the fade-in colour and the body underneath in `theme.FG`.

### Usage

No new shortcuts. Just open the Pipeline tab (`Ctrl+3`), trigger a run with `r`, and the final card now displays the full commit (stage 2 body + stage 3 title combined) while the diff sub-block stays visible below it.

## v0.10.2 — 2026-04-26

Pipeline tab restored to the two-column layout from the original spec, with per-stage scrollable viewports and the diff moved into a dedicated sub-block inside the right panel.

- Restored outer 2-column layout: `changed files` panel on the left + `pipeline · 3 stages` panel on the right. Stacks vertically when `width < 90`.
- Each stage card now uses one of the existing `pipelineViewport1/2/3` instances as its body, so long AI outputs are scrollable. The focused stage grows to `tui.pipeline.stage_focused_height`; the others stay at `tui.pipeline.stage_default_height`.
- Diff lives as the last sub-block inside the right panel (`diff · <path> · +N -M`), driven by a fresh `pipeline.diffViewport`. Updates whenever the file cursor moves.
- Left files panel uses a 2-row delegate (`pipelineFilesDelegate`) showing status letter + path on row 1 and `+N -M` (or `+bin -bin` for binaries) on row 2. Footer renders the totals (`5 files +250 -17`). Numstat data comes from a new `git.GetStagedNumstat()` helper, cached on the Model and refreshed on tab open / pipeline re-run.
- Key reservations:
  - `↑` / `↓` always scroll the diff sub-block.
  - `pgup` / `pgdn` scroll the focused stage's viewport.
  - `tab` cycles focused stage (s1 → s2 → s3 → s1).
  - `j` / `k` move the file cursor (loads its diff into the sub-block).
- `applyPipelineResult` now also pushes the freshly produced output into the relevant per-stage viewport so the user can scroll through the full text immediately after the run.
- Configurable heights (defaults shown):

```toml
[tui.pipeline]
stage_default_height = 4
stage_focused_height = 8
diff_min_height      = 6
```

### Usage

Press `Ctrl+3` to enter the Pipeline tab. Use `j`/`k` to scrub through changed files and `↑`/`↓` to scroll the diff. `tab` cycles which stage is focused (the focused card grows); `pgup`/`pgdn` scroll inside that stage. `r` retries everything, `1`/`2`/`3` retry a specific stage (cascading downstream where supported), `↵` accepts when all stages are Done, `esc` cancels a run.

## v0.10.1 — 2026-04-26

Pipeline tab redesigned to a vertical stack of full-width stage cards so the panel actually uses the full content area and matches the reference mock.

- Dropped the two-column layout. Each stage is now its own rounded card spanning the available width, with: top-edge dot+title (icon coloured per status) and `done`/`running`/etc. pill on the right; 2 lines of stage output as the body; a thick coloured underline at the bottom (`━` characters in `Success`/`AI`/`Error`/`Warning` per status). While running, the underline animates as a pulsing fill in `theme.AI` over a `theme.Subtle` track.
- Replaced `bubbles/v2/progress` with a hand-drawn line so the bar is always visible without threading `progress.FrameMsg` through `View()`. The `progress` import + state remain available for future smoothing.
- Final-commit card collapsed to a 4-row block ("● final commit ready · ⏎ accept & commit") that shows up only when all 3 stages are Done.
- New "selected file + diff" footer renders below the cards: header (`selected file <path> · <status> (n/m)`) plus a colour-aware diff preview pulled from `git.GetStagedFileDiff`. Arrow keys (`↑`/`↓`) cycle through the changed-files list, replacing the broken left-sidebar.
- `renderTitledPanel` extended with `iconColor` (so the status dot can be green while the title stays white) and `hintRaw` (so pre-styled pills/buttons embed in the top edge without being re-painted by the panel's hint style).
- Help line on Pipeline tab updated: `r · 1/2/3 · ↑↓ · ↵ · esc · ^1/^2/^3 · ^x`.

### Usage

Same shortcuts as v0.10.0 plus arrow keys to scrub through changed files. The currently selected file's diff is rendered live at the bottom of the tab.

## v0.10.0 — 2026-04-26

Pipeline tab promoted from placeholder to a real animated 3-stage inspector. Reuses the existing synchronous AI runner — token streaming is intentionally deferred for a follow-up.

- New theme tokens `AI`, `SuccessDim`, `AcceptDim` on `styles.Theme`, populated in `tokyonight`, `gruvbox`, `harmonized`, and `charmtone`. Defaults set in `fillLegacy` so future themes don't break.
- New per-tab state on `Model.pipeline` (`pipeline_state.go`): three `pipelineStage` records with `Status`, `Progress`, `Latency`, `Err`, `flashExpiresAt`, plus a shared spinner and three `bubbles/v2/progress` bars. Stage models hydrate from `config.Prompts.*` so the Pipeline view shows the actual Groq model id per stage.
- Two-pane view (`pipeline_view.go`): left `changed files` panel reuses the existing `pipelineDiffList`; right `pipeline` panel renders three rows (`stage N/3 · <Title>`) with icon + status pill + model id + progress bar + percent / latency. Auto-stacks vertically when `width < 90`.
- Update handler (`pipeline_update.go`) wires `r` (full retry), `1`/`2`/`3` (per-stage retry, cascading downstream), `tab` (panel switch), `enter` (accept commit when all done) and `esc` (cancel running run). Routes spinner ticks, progress frames, and the new pulse / flash / fade / shake messages.
- Animations (`pipeline_animations.go`): indeterminate progress pulse driven at 80ms, post-completion flash window (400ms), three-frame final-commit fade-in (Muted → AcceptDim → Success), and a 3-frame failure shake.
- Existing `Ia*ResultMsg` handlers in `update.go` now also call `applyPipelineResult` so a run started from Compose (Ctrl+W) or from Pipeline (`r`/`1`/`2`/`3`) updates both surfaces consistently.
- Help line on the Pipeline tab now lists `r · 1/2/3 · ↵ · tab · esc`.

### Usage

Press `Ctrl+3` to enter the Pipeline tab. If the AI ran from Compose recently, the tab opens with all three stages already marked Done. Press `r` to re-run everything against the current Compose draft, `1` / `2` / `3` to re-run a specific stage (1 cascades through 2 and 3, 2 cascades through 3, 3 retries only the title), `tab` to focus the changed-files panel, `↵` to accept the assembled commit once every stage is Done, and `esc` to cancel a running pipeline.

## v0.9.5 — 2026-04-26

Fixed `Ctrl+2` (Compose tab) routing into the deprecated step-by-step flow instead of the new multi-section compose view.

- `defaultStateForTab(TabCompose)` now returns `stateWritingMessage` (the new flow) instead of the legacy `stateChoosingType`.
- `switchToTab` now also returns a `tea.Cmd` so a fresh Compose entry can focus the summary input. A new `Model.initFreshCompose` helper resets the draft fields (mirroring the `AddCommit` shortcut on the history list) and focuses `commitsKeysInput`.
- `Ctrl+1/2/3` callsites in `update.go` propagate the returned command.

### Usage

Press `Ctrl+2` from any tab to land directly on the new Compose view, ready to type the summary. The previous draft is preserved if you had already visited Compose during the session.

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
