# Changelog

All notable changes to CommitCraft are documented here. Newest version on top.

## v0.20.0 — 2026-04-28

Redesigned the History list (`stateChoosingCommit`) into a four-zone layout
that surfaces commit context without leaving the screen.

- `MasterList`: dense single-line rows (`#id TYPE [TYPE] scope: title… date`)
  using a new 14-type default palette (ADD, FIX, DOC, WIP, STYLE, REFACTOR,
  TEST, PERF, CHORE, DEL, BUILD, CI, REVERT, SEC). Tags outside the spec
  fall back to a neutral theme-derived palette. The user-supplied
  `commit_type_colors` config is ignored for the History view.
- `FilterBar`: dedicated filter row with prefix `› filter`, placeholder, and
  `n / total` counter; focus-reactive border. Replaces the list's built-in
  filter UI.
- `ModeBar`: segmented switch between two inspection contexts.
- `DualPanel`: 28/flex split below the list with two modes:
  - **A — KeyPoints / Body**: keypoints list + viewport with the AI body
    (`IaCommitRaw`).
  - **B — Stages / Response**: 3 persisted IA stages (`summary`, `body`,
    `title`) + viewport with the corresponding raw output.

All previous keybindings (Enter, d, e, n/Tab, r, ctrl+d, ctrl+s, /, q, ?,
ctrl+x, ctrl+c) keep their behaviour.

### Usage

- Press `/` to focus the new FilterBar; `Esc` clears it and unfocuses.
- Press `Ctrl+M` to swap the DualPanel between *KeyPoints / Body* and
  *Stages / Response*.
- `pgup` / `pgdown` scroll the active right-side viewport.
- Drafts toggle (`Ctrl+D`) keeps the new layout — only the dataset changes.

## v0.19.2 — 2026-04-28

- Unified the rendering path for stage history entries, using the same logic as the live stage card.
- Introduced a new function to abstract the rendering method.
- Removed unnecessary imports and deleted redundant rendering logic.
- Updated the popup functionality to use the new unified rendering function.

## v0.19.1 — 2026-04-28

- Improved the stage history popup UI by refactoring its rendering and navigation logic, providing clearer separation and consistent spacing of the version list and preview pane.
- Added a new helper to display a hint for opening the popup with the **H** key.
- Standardized scrolling behavior and the apply-action flow to match other UI components.
- Enhanced readability through layout adjustments, including line rendering and element spacing.

## v0.19.0 — 2026-04-28

In-memory history of AI generations per pipeline stage. Every successful
run captures a snapshot (text + tokens + latency) so the user can swap
between alternatives mid-session before finalising the commit.

- New `History []stageHistoryEntry` field on `pipelineStage`. Push
  happens from each result-msg handler in `update.go` after the live
  stage state has been updated, so the snapshot mirrors what the card
  showed for that run.
- New popup `internal/tui/stage_history_popup.go` modelled on
  `scope_popup.go`. Lists every captured version newest-first with
  timestamp, total/in/out tokens, latency, and a one-line text preview.
  Cursor + Enter swaps the chosen entry into the live model fields and
  the per-stage telemetry; the focused stage card shows a `vN/M` badge
  whenever there is more than one version.
- History is per-session only — nothing new in SQLite. Cleared after a
  successful `CreateCommit`/`FinalizeCommit`; preserved across
  `SaveDraft` so the user can keep iterating.

### Usage

- Run the pipeline (`Ctrl+W`) or re-run a single stage (`1` / `2` / `3`
  / `4` on the Pipeline tab). Every successful generation is appended.
- Press `H` (capital) on a focused stage card (Pipeline tab) or on the
  pipeline-models row of the Compose tab to open the history popup for
  that stage. `↑↓` / `jk` navigate, `Enter` applies, `Esc` cancels.
- A `vN/M` badge appears on the stage's telemetry line when there is
  more than one version, indicating which one is currently active.
- Finalising the commit (Enter on `stateWritingMessage` after all
  stages are done) clears the history; saving as draft does not.

## v0.18.4 — 2026-04-28

Course-correct on the rate-limit work after confirming via `Ctrl+L`
that Groq's `x-ratelimit-*` headers do come back correctly — the bugs
were on our side.

- **Reverted local request counter** (`mergeWithLocalCounter`) introduced
  in v0.18.3. The `remaining-requests` header is reliable and decrements
  per call; the local counter was redundant. The `requests_today` /
  `requests_day` columns stay in `model_rate_limits` as inert
  zero-valued data (SQLite drop-column is non-trivial); no runtime path
  reads or writes them anymore.
- **Removed the auto-reset block in `EffectiveRateLimits`**. It treated
  `reset_requests` / `reset_tokens` as "time until the bucket fully
  refills", but Groq's headers actually report "time until the next slot
  becomes available" under a token-bucket refill. After 6 seconds the
  RPD bar would falsely zero itself. The function now returns the
  captured snapshot unchanged; bars only refresh on the next real call.
- **RPD / REQ bars go back to header-derived `Limit - Remaining`**,
  still gated by the `RequestsParsed` / `TokensParsed` flags from
  v0.18.2 so a missing header still falls back to "no data yet".
- **New log10 scale on quota bars** (compose RPD/TPM, picker REQ/TOK).
  `logScaleUsed` re-maps the linear `used` value onto a log curve so a
  single call against a 14k daily budget lights up at least one cell
  instead of staying invisible. Each order of magnitude advances ~2
  cells; the usage text beside the bar still shows the real numbers.
  The per-stage TPM mini-bar in the pipeline cards keeps a linear
  scale because there the actual percentage of the per-minute bucket
  matters.

## v0.18.3 — 2026-04-28

Fix: the RPD bar (compose pipeline-models section) and the REQ bar
(picker footer) now use a **locally-tracked daily counter** instead of
the unreliable `x-ratelimit-remaining-requests` header. Groq doesn't
return a refreshing daily-remaining value on the free tier, so the
previous header-based math made the bar swing between "always full"
(when the header was 0/missing) and "always empty" (after a refill or
hydration with parsed=false).

Mechanics:

- `model_rate_limits` gains `requests_today INT` + `requests_day TEXT`
  columns via the existing migration path. Counter is bumped after every
  successful AI call and persisted alongside the headers snapshot.
- `EffectiveRateLimits` zeroes the counter when its stored UTC date no
  longer matches today's, mirroring Groq's UTC midnight bucket reset
  without needing a periodic ticker.
- `LimitRequests` (header) is still used as the bar denominator so the
  ceiling matches the one Groq actually enforces.
- TPM bar (per-minute tokens) keeps using the header `remaining-tokens`
  because that bucket header *does* refresh on every call.

## v0.18.2 — 2026-04-28

Fix: rate-limit bars (compose RPD/TPM, picker REQ/TOK) sometimes rendered
fully consumed (100%) for models whose response omitted an
`x-ratelimit-remaining-*` header. The parser left the field at 0 and the
formula `used = limit - remaining` collapsed to `used = limit`.

- `api.RateLimits` gains `RequestsParsed` / `TokensParsed` flags. The
  parser only sets each flag when *both* halves of the bucket
  (limit + remaining) were present and parsed; otherwise the renderer
  falls back to the "no data yet" placeholder instead of a misleading
  full bar.
- New columns `requests_parsed` / `tokens_parsed` on `model_rate_limits`
  so the flags survive restarts via the existing migration path.
- New debug log line `"rate-limit headers"` after every Groq call. Open
  the logs popup with `Ctrl+L` to inspect what each model actually
  returns; entries with `*_parsed=false` are the ones triggering the
  fallback.

Also lands a batch of pipeline UI refinements from the same iteration:

- Progress bars across the TUI (compose char counter, quota bars,
  stage status track) now use a unified Braille-based ramp.
- Per-stage telemetry (in/out tokens, duration, TPM bar) moved from a
  body row to the centered slot of the card title, freeing the viewport
  to show one more line of AI output.
- Collapsed (non-focused) cards mirror the same telemetry in muted tones.
- Stage status bar at the card bottom now starts with a status word
  (`running` / `done` / `failed` / `cancelled` / `idle`) so its meaning
  is self-explanatory.
- Token breakdown values use accent colors: `in` rides the AI palette
  (blue), `out` rides Success (green); labels stay muted.
- Modified `renderStageBar` to use the Braille ramp and custom empty cell rune.
- Introduced `renderStageTelemetry` and `renderStageTelemetryDim` for stage telemetry rendering.
- Added `stageBarWord` to render status words alongside stage bars.
- Updated `renderBrailleRamp` to allow customization of the empty cell rune.

## v0.18.1 — 2026-04-28

Persistence + finer per-call telemetry on top of v0.18.0:

- New `model_rate_limits` SQLite table stores the latest `x-ratelimit-*`
  snapshot per model. Hydrated into the in-memory cache at every startup,
  so the RPD/TPM/REQ/TOK bars now survive `Ctrl+X` → reopen instead of
  reading "no data yet" until the next live call.
- New `EffectiveRateLimits` helper applied at render time: when the
  per-resource reset window has already passed (`captured_at + reset_*`),
  the corresponding bucket is shown as refilled. No periodic ticker —
  the value is corrected on the next repaint after the window expires.
- New `tpm_limit_at_call` column on `ai_calls` plus a per-stage TPM bar
  appended to each stage card's stats line. Reflects the % of the model's
  TPM ceiling consumed by that specific call. Survives reloads (drafts
  and completed commits, both `EditIaCommit` and the draft Enter paths).
- Quota bars now use a Braille-based smoothing ramp (`⠀⡀⣀⣄⣤⣦⣶⣷⣿`)
  for 8 sub-cell levels of fill, replacing the previous block characters.
- Internal: `applySchemaMigrations` now runs after every CREATE TABLE so
  child-table migrations (e.g. ai_calls) execute against an existing
  schema.

### Usage

Nothing to configure. After upgrading, the next AI call writes its
rate-limit snapshot to disk; subsequent restarts pre-populate the bars.
Per-stage TPM bars appear automatically next to the existing tokens/time
line on every card with telemetry.

## v0.18.0 — 2026-04-28

Per-stage AI telemetry and live model quotas. Every Groq chat completion now
carries its `usage` block (prompt / completion / total tokens, plus queue,
prompt, completion and total times) and its `x-ratelimit-*` headers back into
the TUI:

- Each stage card on the Pipeline tab now shows a compact telemetry line
  under the AI output — total tokens, in/out breakdown, and the wall-clock
  duration of the call.
- Telemetry is persisted in a new `ai_calls` SQLite table linked to the
  parent commit, so reopening a saved draft or completed commit re-displays
  the original numbers without another API call.
- The Compose tab's pipeline-models section renders two thin bars under each
  model line (request bucket and token bucket) fed by the in-memory rate-limit
  cache that the API layer hydrates on every call. Bars turn amber/red as the
  bucket nears exhaustion.
- The model picker popup adds a footer panel with the focused model's most
  recent `REQ` / `TOK` usage and the reset windows reported by Groq.

### Usage

No new keybindings — the telemetry is purely visual:

- After an AI run, check the bottom of each stage card for the `↳ ... tok ·
  ...ms` line.
- On the Compose tab, focus the *pipeline models* section and observe the two
  bars below each stage's model name. Bars stay muted and read `— no data yet`
  until the corresponding model has been called at least once in the current
  session.
- Open the model picker (`↵` over a stage in the pipeline-models section) and
  move the cursor through the table to see the per-model quota footer
  refresh in real-time.

The new `ai_calls` table is created automatically on next startup; nothing
to migrate by hand.

## v0.17.0 — 2026-04-27

Optional pre-release build step. When configured per-repo, CommitCraft now runs
a build command (currently only `make <target>`) right before kicking off the
GitHub release upload, so the binaries published with `gh release create` are
always built from the current tree. Off by default; opt-in via local config.

On build failure the release is aborted, the status bar shows
`Build failed — see logs`, and the full command output is written to the debug
log. On success the flow continues into the existing GitHub upload step.

Also: the global `Ctrl+X` quit shortcut is now suppressed while the release
version popup is open, so it can be used to decrement the version component
under the cursor (vim-style) without exiting the TUI.

### Usage

In the repo's local `.commitcraft.toml`:

```toml
[release_config]
auto_build   = true
build_tool   = "make"     # only "make" is supported for now
build_target = "build_release"
```

If `auto_build` is `true` and `build_target` is empty, auto-build is silently
disabled with a warning. Setting `build_tool` to anything other than `"make"`
also disables it with a warning.

# v0.16.2 — 2026-04-27
Adds direct exit sequence via `Ctrl+X` for the Text User Interface.
  - Checks for `ctrl+x` message and returns model and `tea.Quit` command on match
  - Conditional check enables direct exit sequence from anywhere in the TUI

# Usage

# Added

# v0.16.1 — 2026-04-27

### Changed
- Added contextual info bar below the compose panels in the bottom status bar.

- Updated `renderComposeBottomBar` to use `composeBottomBarContent`.
- Introduced `commitTypeDescription`, `composeScopeBody`, `composePipelineModelBody`, `lookupModelContext`, and `composeAISuggestionBody` helper functions.
- Implemented `RenderLabeled` in `statusbar.go` for rendering labeled pills.

- Extracted modular functions for improved code organization and readability.
- Performed code formatting adjustments for consistency.

## v0.16.0 — 2026-04-27

Interactive Groq model picker for the Compose pipeline. The list of
free-tier models is fetched from the Groq `/openai/v1/models` endpoint,
filtered against a curated free-tier allowlist and cached in SQLite for
24h. Picking a model rewrites the relevant `*_prompt_model` field in
either the global config (`~/.config/CommitCraft/config.toml`) or the
per-repo `.commitcraft.toml`, scope chosen explicitly per save.

- `internal/api/groq.go`: new `ListGroqModels(apiKey)` hitting
  `/openai/v1/models` with the existing Bearer pattern.
- `internal/config/free_models.go`: curated `FreeTierChatModels`
  allowlist (chat-capable, free-tier-listed IDs only).
- `internal/config/save_model.go`: `SaveModelForStage` /
  `ApplyModelToConfig` / `CurrentModelForStage` plus `ConfigScope` and
  `ModelStage` types. Rewrites only the targeted TOML key by parsing
  the file as a generic table so unrelated config survives.
- `internal/storage/database.go` + `models_cache.go`: new
  `groq_models_cache` table (created via `createModelsCacheTable`),
  `SaveModelsCache` / `LoadModelsCache` helpers and an
  `IsModelsCacheStale` TTL check.
- `internal/tui/model_picker_popup.go`: two-step popup — a four-column
  `bubbles/v2/table` (Model · Owner · Ctx · Status), then a `g`/`l`
  scope prompt.
- `internal/tui/model_picker_glue.go`: `openModelPickerCmd` and
  `refreshModelPickerCmd` perform cache-aware fetch/save in a tea.Cmd
  and emit `modelPickerOpenedMsg` so the parent rebuilds the popup.
- `internal/tui/update_writing.go`:
  `handlePipelineModelsSectionKey` adds `↑↓/hjkl` to move the cursor
  through the configurable stages and `enter` to open the picker.
- `internal/tui/compose_sections.go`: pipeline-models row now reads the
  current model via `config.CurrentModelForStage`, shows a `▸` cursor
  on the focused stage and a `↑↓ pick stage · enter change model`
  hint underneath.

### Usage

In the Compose tab press `Tab` until the **pipeline models** section is
focused, then `↑/↓` (or `j/k`) to pick a stage and `Enter` to open the
picker. Use `↑↓/jk/pgup/pgdn` to navigate the table, `r` to force a
fresh fetch from the Groq API, `esc` to cancel. After picking a model, press `g` to save it
in the global config or `l` to save it in `.commitcraft.toml` for the
current repo. The change is applied to the running session immediately
and reflected on the Pipeline tab.

## v0.15.14 — 2026-04-27

Rework `@`-mentions on the Compose tab so the marker survives long
enough to look like a real chip:

- The `@` is no longer stripped when the user picks a file from the
  mention popup, nor when they cancel it. The full `@<path>` token now
  stays in the textarea buffer and any saved key point keeps it too.
- Right before each AI prompt is built, every `@<path>` token is
  flattened back to a bare path via `stripMentions` (regex pass over
  the joined developer points). The AI keeps seeing clean file paths;
  only the human-facing surfaces get the marker.
- Saved key points and the AI-suggestion panel now render every
  `@<path>` mention as a green chip using the existing success-pill
  palette (`pillOK`). The chip styling is centralised in a new
  `statusbar.RenderMentionPill`.
- The live textarea (where the user types) keeps showing mentions as
  plain text — Bubbles textarea v2 has no per-token style hook and a
  custom widget would be a much larger change. Mentions become chips
  the moment the key point is saved (or once the AI panel re-renders
  the message).

### Files

- `internal/tui/statusbar/statusbar.go`: new `RenderMentionPill(token)`
  reusing `pillOK`.
- `internal/tui/update.go`: `mentionFileSelectedMsg` keeps the leading
  `@`; `closeMentionPopupMsg` no longer rewrites the value.
- `internal/tui/ai_pipeline.go`: `mentionStripRegex` + `stripMentions`,
  applied to `developerPoints` in `iaCallChangeAnalyzer`.
- `internal/tui/compose_sections.go`: new `styleMentions` helper;
  `renderComposeKeypointsArea` runs each saved key point through it
  before truncation.
- `internal/tui/view_writing.go`: `identifierRegex` now matches
  `@<token>` first; `styleIdentifiers` dispatches mention matches to
  `statusbar.RenderMentionPill` and the rest to the inline-code style.

### Usage

Type `@` in the summary as before. Pick a file with the popup (or
cancel to keep the bare `@`). Save the line with `Ctrl+A` — the chip
appears in the key-points list. When the AI runs, the `@` is dropped
internally so the prompt sees plain file paths.

## v0.15.13 — 2026-04-27

In the key-points list, the active row (the one the keypoint cursor is on
while the section has focus) now uses `theme.Warning` instead of
`theme.Primary` for its `▸` marker and `×` glyph. The amber tone pops
harder against the secondary-coloured siblings, making the deletion
target unmistakable.

- `internal/tui/compose_sections.go`: swap `theme.Primary` →
  `theme.Warning` in the `isActive` branch of the marker/remove colour
  selection.

## v0.15.12 — 2026-04-27

Polish the key-points input on the Compose tab:

- Saved items now use `theme.Secondary` for their `▸` marker (and the
  trailing `×`) when the key-points section is blurred, so they read
  louder than the surrounding muted prompts. When the section owns focus,
  the highlighted row keeps the `theme.Primary` accent and the rest fall
  back to muted so the active row stands alone.
- The `commitsKeysInput` textarea cursor is now `theme.Primary`
  (previously `theme.Secondary`). The override is local to this
  textarea, so the edit-message popup and release-text editor keep their
  current cursor colour.
- Inline navigation/removal of saved key points was already wired:
  `↑↓` / `←→` / `hjkl` to move the highlight, `x` / `backspace` /
  `delete` to remove it (see the help line on the Key points section).
  No code changes here; documenting it because the visual treatment
  above makes the cursor row obvious.

### Files

- `internal/tui/model.go`: `kpiStyles.Cursor.Color = theme.Primary`
  override for the compose textarea.
- `internal/tui/compose_sections.go:204-228`: rework `markerColor` /
  `removeColor` selection in `renderComposeKeypointsArea` to apply
  Secondary-when-blurred / Primary-when-active / Muted-otherwise.

## v0.15.11 — 2026-04-27

Compact pipeline cards now draw a decorative `─` line between the stage
title (`stage N · …`) and the right-aligned status pill, matching the
gray underline aesthetic used elsewhere in the TUI. Idle stages use the
muted gray, active stages use the subtle gray, so the active row reads
slightly louder. Total row width is preserved (1 space + N dashes + 1
space = same gap as before), and very narrow widths fall back to the
plain spacer so nothing overlaps.

- `internal/tui/pipeline_view.go`: new `renderStageCardDivider` helper;
  `renderStageCardCollapsed` now uses it instead of a plain space-padded
  gap.

## v0.15.10 — 2026-04-27

Drop the working-directory suffix from the initial WritingStatusBar
message now that the CWD pill in the tab bar is the canonical source of
truth. The status bar message becomes a clean "choose, create, or edit a
commit" / "…release" without the `::: Working directory: …` tail, so the
two surfaces no longer duplicate each other.

- `internal/tui/model.go`: simplify `statusBarInitialMessage` for both
  CommitMode and ReleaseMode entry points.

## v0.15.9 — 2026-04-27

Persistent CWD breadcrumb: the working directory is now visible as a
two-segment "CWD <path>" pill embedded in the top tab bar, horizontally
centered between the tabs (`History | Compose | Pipeline`) on the left
and the `^1/^2/^3` shortcut hints on the right. The pill uses the
existing debug palette (slate label + near-black body) so it reads as
ambient metadata, not a status alert. `$HOME` is collapsed to `~`, and
long paths are truncated from the left with a leading `…` so the trailing
segments (repo name, current subdir) stay visible on narrow terminals.
On very narrow terminals the pill is dropped and the original plain
spacer is used to keep the tab row from breaking.

- `internal/tui/statusbar/statusbar.go`: new exported
  `RenderCwdPill(path, maxWidth)` that reuses `pillDebug` / `msgDebug`
  and handles rune-safe left-truncation.
- `internal/tui/tabs.go`: `renderTabBar` now centers the CWD pill inside
  the spacer between `leftBar` and `rightBar`; new `cwdDisplayPath`
  helper collapses `$HOME` to `~`.

### Usage

The CWD pill is always on whenever the tab bar is visible; nothing to
enable. It reflects the directory the binary was launched from (the same
`pwd` already passed to the model).

## v0.15.8 — 2026-04-27

Auto-highlight code-like tokens in the non-glamour panels (Compose AI
suggestion and pipeline stages 2-4). Identifiers that previously rendered
as flat prose now get the same inline-code styling as backtick-wrapped
segments, matching the visual density of the glamour-rendered Summary
panel.

- `internal/tui/view_writing.go`: add `identifierRegex` and
  `styleIdentifiers`; `renderCommitLine` runs every non-backtick chunk
  through it. Detects camelCase/PascalCase, snake_case/CONSTANT_CASE,
  file paths, `file.ext` names, and `Func()` / `pkg.Func()` calls. Tokens
  already inside backticks are left alone (single styling pass).
- Glamour-rendered panels (release viewport, Summary stage) are not
  affected.

## v0.15.7 — 2026-04-27

Theme-tie the commit-type popup: the row cursor (`❯`) now uses
`t.Secondary`, and the hint line keys (`↑↓`, `enter`, `esc`) use
`t.Accent` — matching the help styles used elsewhere in the TUI —
while the labels stay in `t.Muted`.

- `internal/tui/commit_type_list.go`: `CommitTypeDelegate` carries a
  `Theme *styles.Theme`; the cursor glyph is rendered with
  `Theme.Secondary` (bold) when available, plain `❯` otherwise.
  `NewCommitTypeList` now takes the theme.
- `internal/tui/type_popup.go`: rebuild the hint line as a
  key/desc-styled string so the shortcuts pop in the accent color.
- `internal/tui/model.go`: pass the active theme to
  `NewCommitTypeList`.

## v0.15.6 — 2026-04-27

Improvements to the commit-type popup (`Ctrl+T`): show a hint line
with the available shortcuts and auto-fit the popup width to the
widest row instead of locking it at half the terminal width.

- `internal/tui/type_popup.go`: render a muted hint
  ("type to filter · ↑↓ nav · enter pick · esc clear/close") under
  the list. Adjusted height calc to reserve space for the hint.
  New `CommitTypePopupContentWidth` helper computes the minimum
  width needed by the longest row (tag + description).
- `internal/tui/update_writing.go`: pass `max(model.width/2,
  contentW)` clamped to `model.width-4`, so the popup grows when
  needed and never overflows the terminal.

### Usage

Open the popup with `Ctrl+T` from the writing state. It now expands
horizontally to fit the longest tag+description, and the hint line
at the bottom lists the active shortcuts.

## v0.15.5 — 2026-04-27

Fix the commit-type list filter so typing matches only against the
tag, not the description. Before, searching for a short tag like
`fix` would also surface every type whose description happened to
contain that substring.

- `internal/tui/commit_type_list.go`: `CommitTypeItem.FilterValue`
  now returns only `Tag`.

## v0.15.4 — 2026-04-27

Add a single blank line between every section pill and its content
in the compose left panel so the labels breathe instead of sitting
flush on top of their components.

- `internal/tui/compose_sections.go`: insert `""` between label and
  body in the type, scope, summary, key points and pipeline models
  renderers.

## v0.15.3 — 2026-04-27

Tweaks on top of the compose panel refresh: header and middle block
get explicit single-line breathing room, the summary area is no
longer vertically centered (it sits flush right after the header),
and the keypoint textarea prompt symbols (`>` / `:::`) now use
`t.Secondary`.

- `internal/tui/view_writing.go`: prepend a blank line to the header
  block and to the middle block; pad the gap between middle and
  footer so pipeline models stay pinned to the bottom of the panel.
- `internal/tui/styles/theme.go`: switch the focused
  `KeyPointsInput.PromptFocused` and `DotsFocused` to `t.Secondary`.

## v0.15.2 — 2026-04-27

Visual refresh of the compose tab's left panel: the section labels
become theme-aware pills, the summary + keypoints input area is
vertically centered, pipeline models sit at the panel footer with
their own divider, and the keypoint textarea prompt symbols now
follow the theme accent.

- `internal/tui/styles/theme.go`: new `Theme.SectionPill(focused)`
  helper (Surface/Muted blurred · Primary/BG focused). Switched
  `KeyPointsInput.PromptFocused` and `DotsFocused` from `t.Green`
  to `t.Primary` so the textarea prompt recolors with the theme.
- `internal/tui/compose_sections.go`: applied `SectionPill` to the
  five section labels (commit type, scope, summary, key points,
  pipeline models). The keypoints "X items" counter stays as plain
  muted text on the right.
- `internal/tui/view_writing.go`: rewrote `assembleComposeLeftBody`
  into three zones — header (type + scope + divider), centered
  middle (summary + keypoints), footer (divider + pipeline models).
  Uses `lipgloss.Place` with `lipgloss.Center` to vertically center
  the middle block in the leftover height, falling back to plain
  stacking when the panel is too short.

### Usage

No new keybindings or behavior changes. Open the compose tab as
before; the new layout shows two horizontal rules (above summary,
above pipeline models), the input zone visually anchored in the
middle, and section headers as small pills that swap to the theme
primary when focused.

## v0.15.1 — 2026-04-27

Restore `tab` / `shift+tab` as directory-navigation keys in the scope
file picker popup, mirroring `→` / `←`.

- `internal/tui/scope_popup.go`: extend the `left` / `right` cases in
  the popup's Update to also match `shift+tab` / `tab`. Hint line
  updated.

### Usage

In the scope popup (`Ctrl+P` from the writing state, or `e`/`Enter`
on the scope section): `tab` enters the highlighted directory,
`shift+tab` goes up — same effect as `→` / `←`.

## v0.15.0 — 2026-04-27

Add a persistent warning pill on the status bar when a commit is loaded
from the DB without a linked git hash (drafts and history commits
generated outside the CLI). In that mode `gitStatusData` still reflects
the live workspace, so the scope picker cannot mark the commit's
actual modified files — the pill makes that limitation visible.

- `internal/tui/model.go`: new `scopeDataStale` flag on the model.
- `internal/tui/statusbar/statusbar.go`: new `ScopeStaleIndicator` and
  `renderScopeStalePill` using `pillWarn` and the NerdFont glyph
  `U+F13D2` (󱏒), with `!` as ASCII fallback.
- `internal/tui/pipeline_update.go`: `Model.syncScopeStaleIndicator`
  pushes the flag into `WritingStatusBar` so the pill toggles in real
  time.
- `internal/tui/update_commit.go`: set the flag when loading a draft
  (Enter) or editing a saved commit (EditIaCommit).
- `internal/tui/update_reword.go`: clear the flag in both
  reword paths (`-w` startup and "Commit and reword"), since those
  replace `gitStatusData` with the target commit's real status.
- `internal/tui/transitions.go`: clear the flag when returning to the
  main list.

### Usage

No new keybinding. The pill appears automatically next to the version
indicator whenever the loaded commit lacks a hash; open with
`commitcraft -w <hash>` (or use "Commit and reword") to keep the
scope picker fully aware of the commit's modified files.

## v0.14.2 — 2026-04-27

Scope file picker now opens with the modified-only filter enabled by
default, since the scope is almost always one of the files touched by
the pending commit.

- `internal/tui/scope_popup.go`: initialize `showOnlyMod = true` and
  apply `UpdateFileListWithFilterItems` right after `NewFileList` so
  the popup's first frame already shows only changed paths.

### Usage

Open the scope picker as before (`Ctrl+P` from the writing state). It
starts filtered to modified files/folders; press `Ctrl+R` to toggle
back to the full directory listing.

## v0.14.1 — 2026-04-27

Fix the changelog refiner not running when triggered from the Compose
tab and add a persistent CHANGELOG indicator pill to the status bar.

- `internal/tui/pipeline_update.go`: extract the inline detection in
  `pipelineStartFullRun` into a reusable `Model.refreshChangelogState()`
  method. It now updates `changelogActive` (the runtime gate the
  refiner reads), `pipeline.activeStages`, and the new persistent
  indicator flags (`changelogFilePresent`, `changelogWillAutoUpdate`)
  in one shot. Returns the skip reason so callers can surface it in
  the status bar.
- `internal/tui/update_writing.go`: the Compose-tab Ctrl+W handler
  now calls `refreshChangelogState()` before dispatching
  `callIaCommitBuilderCmd`. Previously the helper only ran from the
  Pipeline tab's `r` shortcut, which left `changelogActive = false`
  and made `runChangelogRefiner` bail out unconditionally on Compose.
- `internal/tui/model.go`: `NewModel` calls
  `refreshChangelogState()` so the indicator is correct from the
  first frame. `transitions.go::createCommit` re-runs the refresh
  after a successful commit because the file just changed.
- `internal/tui/statusbar/statusbar.go`: new
  `ChangelogIndicator{Present, WillAutoUpdate, UseNerdFonts}` field on
  `StatusBar` plus a `renderChangelogPill` helper. When `Present` is
  true the right side of the bar gets a green pill (reusing
  `pillChangelog`) showing one of two NerdFont icons:
  - `󱇼` (U+F11FC) — file detected but auto-update will not run
    (feature off, dirty file, etc.).
  - `󱫓` (U+F1AD3) — auto-update will run on the next Ctrl+W.
  Without NerdFonts the icon falls back to the existing `≡` glyph.

### Usage

The CHANGELOG pill is now always visible on the right of the status
bar whenever `[changelog] enabled = true` and the configured file
exists. The icon tells you at a glance whether stage 4 will run:

- `󱫓` next to the version pill → next Ctrl+W (Compose or Pipeline
  tab) will detect, generate the entry, and stage the file.
- `󱇼` → file detected but skipped this run, usually because you
  already modified `CHANGELOG.md` yourself. The auto-write would
  clobber your edit, so the refiner stays out.

Triggering Ctrl+W from the Compose tab now follows the exact same
flow as from the Pipeline tab, including the optional 4th stage when
the file is clean.

## v0.14.0 — 2026-04-27

Refresh the Pipeline tab so the focused stage gets most of the screen,
the diff stays comfortable to read, and the rendered output across all
stage cards matches the look used by Compose's AI suggestion panel.

- `internal/tui/pipeline_keys.go`: `ctrl+↑` and `ctrl+↓` now alias
  `pgup`/`pgdown` for scrolling the focused stage's viewport, so the
  user does not have to leave home row to scroll.
- `internal/tui/pipeline_state.go`, `pipeline_view.go`,
  `pipeline_update.go`: the pipeline panel collapses every non-focused
  stage to a single line (icon + `stage N · title` + status pill) and
  the focused stage absorbs the freed space. `Tab` cycles focus through
  every active stage and the `final commit ready` card too — when the
  final card is focused the stages stay collapsed and the final card
  grows. `cycleFocus(showFinal bool)` plus a new `focusedFinal` flag
  on `pipelineModel` drive the rotation. `allDone`/`resetAll` ignore
  the optional 4th stage when CHANGELOG support is inactive.
- `internal/tui/pipeline_view.go`: stage 1 (the change analyzer
  summary) now renders through Glamour with the Tokyo Night style
  (`charm.land/glamour/v2/styles.TokyoNightStyleConfig`), since the
  summary is genuine markdown. Stages 2, 3, 4 and the
  `final commit ready` card share `renderCommitMessage` (already used
  by Compose's AI suggestion panel), so commit titles render bold,
  inline `` `code` `` segments are highlighted, and hand-written line
  breaks survive verbatim. Content is rendered fresh each frame so
  resizing or focus changes always reflow correctly.
- `internal/tui/pipeline_view.go`: the diff sub-block reserves an
  extra 20% of the right panel's inner height on top of the configured
  floor so reviewing the diff alongside an expanded focused stage
  stays comfortable on tall and short terminals.
- `internal/tui/pipeline_update.go`: the changelog refiner now gates
  on `model.changelogActive` (single source of truth populated by
  `pipelineStartFullRun`), so the "CHANGELOG already modified" skip
  reason actually prevents the AI call instead of just hiding the 4th
  card. The dirty-file check uses the new `git.HasFileChanges` helper
  (`git status --porcelain -- <path>`) which also catches unstaged and
  untracked modifications, not only staged ones.
- `internal/tui/ai_pipeline.go`,
  `internal/config/prompts/changelog_refiner.prompt.tmpl`: the
  changelog refiner emits two independent fields — `changelog_entry`
  for the file and `commit_mention_line` for the commit body — so
  stage 2's body is never rewritten by stage 4. Composition in
  `composeFinalCommitMessage` appends the mention with a blank line
  of separation. A deterministic `fallbackMentionLine` kicks in if the
  model omits the literal `CHANGELOG.md` token.

### Usage

On the Pipeline tab:

- `Tab` cycles focus: `stage 1 → … → stage N → final commit → stage 1`.
  Only the focused card expands; the rest collapse to a one-line row.
- `pgup`/`pgdown` or `ctrl+↑`/`ctrl+↓` scroll the focused stage's
  viewport. Diff scrolling and changed-file navigation are unchanged.
- The `final commit ready` card joins the cycle once the pipeline is
  done; focusing it grows the card so the assembled commit is easier
  to skim before pressing `Enter`.

When the optional CHANGELOG flow is enabled and `CHANGELOG.md` is
already dirty (staged, unstaged, or untracked), the pipeline now skips
stage 4 entirely — no extra AI call, no body mention, no auto-write.
The status bar pill shows `≡ CHANGELOG · CHANGELOG already modified ·
skipping auto-update`.

## v0.13.0 — 2026-04-26

Add an optional 4th AI step that produces a CHANGELOG entry alongside the
commit. When enabled, after stage 3 finishes the pipeline detects the
project's CHANGELOG.md format, asks the model for a matching new entry plus
a refined body that mentions the changelog update, and on commit acceptance
prepends the entry and stages the file together with the user's changes.

- `internal/changelog/changelog.go`: new package with `Detect`,
  `SuggestNextVersion`, and `Prepend`. Detection samples the title plus the
  first existing entry so the prompt can imitate the project's heading
  level, version prefix style, date format, and bullet layout. Version
  bumping defaults to patch and supports `minor`/`major` via config.
  Insertion preserves the H1 title and any intro paragraph below it.
- `internal/config/types.go`, `loader.go`, and the new
  `prompts/changelog_refiner.prompt.tmpl`: opt-in `[changelog]` config
  block (`enabled`, `path`, `bump_strategy`, `prompt_file`, `prompt_model`)
  with a default of `enabled = false`. Prompt is loaded through the same
  `createOrLoadPromptFile` flow as the other stage prompts.
- `internal/tui/ai_pipeline.go`: new `runChangelogRefiner` runs after stage
  3 in `ia_commit_builder` and the stage-2/stage-3 retry commands. It
  reads the body from a fresh `iaCommitBodyOriginal` snapshot so re-runs
  never refine on top of an already-refined paragraph. JSON parsing is
  tolerant of code fences; on parse failure a deterministic fallback entry
  is emitted and the body is left untouched.
- `internal/tui/transitions.go`: `createCommit` writes the entry via
  `changelog.Prepend` and runs `git.StageFile` (a new helper in
  `internal/git/git.go`) so the next external `git commit` picks the
  CHANGELOG up alongside the user's staged changes. Failures abort the
  commit save with a status-bar error. The write is skipped in plain
  reword flows (`-w <hash>` without "commit and reword") so the
  interactive rebase used for non-HEAD reword is never tripped by a
  newly staged file.
- `internal/tui/update.go`: status bar surfaces the suggested version
  through a brand-new `LevelChangelog` pill ("≡ CHANGELOG · AI commit +
  CHANGELOG entry vX.Y.Z ready!") whenever the refiner produced an entry.
  The pill is also raised at pipeline start ("pipeline started · 4 stages
  · CHANGELOG detected") so the user sees the extra step coming before
  it runs. Stage rerun handlers (`1`/`2`/`3`) keep the pill green when
  they cascade through the refiner.
- `internal/tui/statusbar/level.go` + `statusbar.go`: new
  `LevelChangelog` constant with a green-tinted pill/body palette and the
  `≡ CHANGELOG` label (Unicode triple-line glyph, no emojis).
- `internal/tui/pipeline_state.go`, `pipeline_view.go`, `pipeline_update.go`:
  the pipeline panel grows from 3 to 4 stage cards when CHANGELOG is
  detected at run start. The 4th card ("Changelog Refiner") shows the
  generated entry exactly like the other stages, share the same
  status/focus/retry semantics, and is hidden completely when the file is
  absent or the feature is disabled — so repos without a changelog see
  zero UI changes. A new `pipeline.activeStages` int gates rendering and
  `allDone`/`cycleFocus` skip the inactive 4th slot.
- `internal/tui/pipeline_keys.go`, `keys.go`, `commands.go`: stage 4 retry
  is bound to `4` and dispatches `callIaChangelogOnlyCmd`, which re-runs
  only the refiner against the existing stage 2/3 outputs and emits a new
  `IaChangelogResultMsg` for the per-stage status bar update — saves
  tokens compared to a full pipeline re-run.
- `internal/tui/ai_pipeline.go`: the refiner now always guarantees the
  literal string `CHANGELOG.md` ends up in `refined_body`. If the model
  drops the mention, `appendChangelogMention` patches a single trailing
  bullet (`- Updated CHANGELOG.md with vX.Y.Z entry.`) using the body's
  existing bullet style. The prompt template was tightened to require the
  mention explicitly.
- `internal/tui/pipeline_update.go`: `pipelineStartFullRun` and
  `pipelineRetryStage` now also clear the changelog snapshot fields so a
  retry starts from a clean state, and a stage 4 retry is treated as a
  refresh that does not touch stages 1–3. The refiner is also skipped
  when `git diff --cached --name-only` already lists the configured
  changelog path (new helper `git.IsFileStaged`) — protects user-authored
  changelog edits from being clobbered by the auto-prepend.

### Usage

The feature is off by default. To enable it edit
`~/.config/CommitCraft/config.toml` (or the local `.commitcraft.toml`)
and add:

```toml
[changelog]
enabled = true
path = "CHANGELOG.md"        # optional, this is the default
bump_strategy = "patch"      # patch | minor | major
prompt_file = "prompts/changelog_refiner.prompt"
prompt_model = "llama-3.1-8b-instant"
```

When enabled, `Ctrl+W` runs the regular 3-stage pipeline and, if a
CHANGELOG file exists at `path`, follows up with the refiner — visible
as a 4th stage card on the Pipeline tab. The status bar shows a
`≡ CHANGELOG` pill at run start and on completion. The final commit body
always mentions `CHANGELOG.md` (guaranteed by a deterministic safety
net even when the model drops it). Pressing `Enter` to accept the commit
prepends the new entry into CHANGELOG.md and runs `git add` on it.

Stage retries on the Pipeline tab:

- `1` → re-runs stages 1+2+3+4 (analyzer → body → title → refiner)
- `2` → re-runs stages 2+3+4
- `3` → re-runs stages 3+4
- `4` → re-runs only the refiner against the existing stage 2/3 output

Repos without a CHANGELOG, or sessions with `enabled = false`, behave
exactly as before — no extra calls, no file writes, and no 4th card in
the UI.

## v0.12.5 — 2026-04-26

Fix Nerd Font icons in the file picker and make the scope popup filter
always-on.

- `internal/tui/format.go::GetNerdFontIcon` had silently lost most of
  its glyphs (the directory branch, `.py`, `.java`, `.rs`, `.yml`,
  `.toml`, `.env`, `.md`, image extensions, `makefile`, `.gitignore`,
  and the default fallback all returned the empty string). Repopulated
  with proper Nerd Font codepoints written as `\u`/`\U` escapes so the
  glyphs survive future re-saves in editors without the font.
- Folders now render with ``; the default branch returns
  `` so any unknown file type still gets a generic file icon
  instead of nothing.
- `internal/tui/update.go`: route `list.FilterMatchesMsg` to the active
  popup. `bubbles/list` produces this message asynchronously when the
  user types into `FilterInput`; without explicit forwarding it fell
  through to the per-state handler (which ignores non-key messages),
  so `filteredItems` never got updated and the typed query did not
  filter anything visible.
- `internal/tui/type_popup.go` (Ctrl+T): same always-on filter
  treatment as the scope popup. The list opens in `Filtering` state,
  `AcceptWhileFiltering` / `CancelWhileFiltering` are cleared so `/`
  and `enter` reach the popup, `↑/↓` are routed to `CursorUp`/
  `CursorDown` directly, `enter` picks the highlighted commit type,
  and `esc` clears the filter first then closes the popup.
- `internal/tui/scope_popup.go` (Ctrl+P): the popup now opens already
  in `Filtering` state — `SetFilterText("")` followed by an explicit
  `SetFilterState(list.Filtering)` (necessary because `SetFilterText`
  by itself transitions to `FilterApplied`, which routes keys back to
  `handleBrowsing` where printables are ignored). The list's built-in
  `AcceptWhileFiltering` / `CancelWhileFiltering` bindings are cleared
  so `/` and `enter` are not consumed as filter accept/cancel — the
  popup handles them. The `h`/`l` aliases for directory navigation
  were removed; only `←`/`→` move between directories. `↑`/`↓` are
  intercepted by the popup and call `list.CursorUp` / `CursorDown`
  directly (during `Filtering` state `bubbles/list` would otherwise
  forward arrows to `FilterInput` and never move the cursor through
  items). `enter` picks the highlighted item, `esc` clears the filter
  when present and closes the popup when the filter is already empty,
  `ctrl+r` toggles modified-only. `refreshList` re-enters `Filtering`
  state on directory changes so typing keeps feeding the filter.

### Usage

Open the scope picker with `Ctrl+P` from the Compose tab. Just type to
fuzzy-search the current directory; `↑/↓` navigate items, `←/→`
go up to the parent / enter the selected directory, `Ctrl+R` toggles
"modified files only", `Enter` picks the highlighted entry, `Esc`
clears the search (or closes the popup if the search is already
empty).

## v0.12.4 — 2026-04-26

Theme-aware inline `code` styling in the AI suggestion viewport.

- `Model.renderCommitMessage` now highlights backtick-wrapped segments
  (e.g. `` `SetTheme` ``) with `Theme.Secondary` foreground on
  `Theme.Surface` background, so the styling follows the active theme.
- New `renderCommitLine` helper does per-line splitting on backticks and
  width-wraps with the line's text style so inline-code segments never
  get torn across the wrap boundary. Unmatched trailing backticks are
  rendered verbatim instead of swallowing the user's content.
- Newlines from the original message are still preserved one-for-one
  (no glamour, no Markdown semantics).

## v0.12.3 — 2026-04-26

Stop rendering the AI commit message through glamour on the Compose tab.

- Commit messages are plain text, not Markdown. Running them through a
  Markdown renderer mangled real-world bodies: lazy continuations folded
  bullets back into the previous paragraph, lines indented with 4 spaces
  became code blocks, and even the `preserveHardBreaks` workaround from
  v0.12.2 didn't survive those interactions.
- New `Model.renderCommitMessage(msg, width)` in `view_writing.go`:
  bolds the title line in `Theme.Primary`, renders the body in
  `Theme.FG`, and wraps both with `lipgloss.Style.Width` so every line
  break the user typed is preserved verbatim.
- The right-side viewport in `buildWritingMessageView` now uses this
  helper instead of `glamour.Render`. The pipeline tab previews still
  use glamour since their content is structured AI output.

## v0.12.2 — 2026-04-26

Fix line-break collapsing in the AI suggestion viewport on the Compose tab.

- The right-side viewport renders `commitTranslate` through glamour, which
  follows CommonMark and collapses single newlines inside a paragraph.
  When the user manually edited the message via the edit-message popup
  and added intra-paragraph line breaks, those breaks vanished on render.
- New `preserveHardBreaks` helper in `view_writing.go`: splits the text
  on `\n\n` and replaces remaining single `\n` with the Markdown hard
  break `"  \n"` inside each paragraph. Paragraph separators are left
  untouched so blank-line semantics still work.
- Applied to the compose tab's AI suggestion render. The pipeline tab
  previews still use the raw glamour render since they show structured
  AI output that's already paragraph-shaped.

## v0.12.1 — 2026-04-26

Fixes for the configuration popup theme flow.

- Persistence inverted: `tui.UpdateConfigTheme` now always writes the
  theme to the global `~/.config/CommitCraft/config.toml`. The local
  `.commitcraft.toml` is no longer touched by the popup, so it doesn't
  get polluted with unrelated TUI defaults (e.g. `use_nerd_fonts = false`).
- Local override now actually applied at startup: new
  `config.ResolveTUIConfig` merges `localCfg.TUI.Theme` over the global
  one in `cmd/cli/main.go` so a per-repo `.commitcraft.toml` can still
  override the user-wide theme.
- Logo now follows the active theme: added
  `statusbar.StatusBar.SetTheme` and call it from the `themePreviewMsg`
  / `themeAppliedMsg` / `closeConfigPopupMsg` handlers in `update.go`,
  so the `⌘ CommitCraft` pill picks up the new `Theme.Logo` (which
  defaults to `Theme.Primary`) instead of staying on charmtone's blue.

## v0.12.0 — 2026-04-26

New configuration popup with a theme picker. The selected theme is applied
live as you move through the list and persisted on confirm.

- New `internal/tui/config_popup.go` (`configPopupModel`): list-style popup
  built on `styles.AvailableThemes()`, emits `themePreviewMsg` on cursor
  moves, `themeAppliedMsg` on Enter, and `closeConfigPopupMsg` on Esc
  (which restores the original theme).
- `Ctrl+,` is wired in `update.go` as a global shortcut (only when no other
  popup is open).
- `TUIConfig.Theme` (new TOML field `theme` under `[tui]`) is read at
  startup via `styles.GetTheme(name, useNerdFonts)` in `model.go` and
  written by `tui.UpdateConfigTheme` to the local `.commitcraft.toml`
  when present, otherwise to the global `~/.config/CommitCraft/config.toml`.
- `Model.themeName` tracks the active theme so previews can be reverted on
  cancel.

### Usage

Press `Ctrl+,` from anywhere (no other popup open) to open the
Configuration popup. Use `↑/↓` to preview each theme live in the TUI,
`Enter` to save the selection (persists to `.commitcraft.toml` if it
exists in the cwd, otherwise to `~/.config/CommitCraft/config.toml`), or
`Esc` to discard the change and restore the previous theme.

## v0.11.1 — 2026-04-26

After applying changes from the edit-message popup, the "Changes applied" status now flashes for 2 seconds via `WritingStatusBar.ShowMessageForDuration` and then restores the prior compose status, instead of sticking until the next user action.

## v0.11.0 — 2026-04-26

The "edit AI message" flow is now a popup instead of a separate full-screen state. Same shortcut (`Ctrl+E`), but only available once the AI has produced a response.

- New `internal/tui/edit_message_popup.go` (`editMessagePopupModel`): textarea seeded with `commitTranslate`. `ctrl+s` emits `editMessageAppliedMsg` (writes back to `commitTranslate`), `esc` closes without applying, `ctrl+d` deletes the current line, `enter` is a regular newline.
- `update_writing.go::CreateIaCommit`-sibling `Edit` handler now: if `commitTranslate` is empty, surfaces a red status-bar error ("There is no AI response yet…") and returns; otherwise opens the popup. No state change — the compose view stays mounted underneath.
- Removed the old full-screen edit flow: `stateEditMessage`, `updateEditingMessage`, `buildEditingMessageView`, `editingMessageKeys`, `model.msgEdit`, and `msgEditHeaderView` / `msgEditFooterView`. References cleaned up in `update.go`, `view.go`, `tabs.go`, `compose_status.go`, `keys.go`, `model.go`.

### Usage

In compose, after running the AI flow (`Ctrl+W`), press `Ctrl+E` to open the edit popup. Edit freely (newlines via `Enter`, `Ctrl+D` to drop the current line), then `Ctrl+S` to apply or `Esc` to cancel. Pressing `Ctrl+E` before the AI has responded triggers an error in the top status bar instead of opening the popup.

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
