# Progress Tracker

Update this file after every meaningful implementation change.

## Current Phase

- **Brownfield adoption** — project predates the spec-driven workflow. v0.49.0 in production. Adopting `context/` retroactively to lower the cost of further changes.

## Current Goal

- Map current flows into the six context files so future feature work can route through `/spec-driven-dev spec`. No new feature work in flight.

## Completed (visible in code, pending user confirmation)

> Marked as completed because they are already shipped in the binary. Confirm with the user that this list is accurate before treating it as canonical.

- TUI commit flow: `stateWritingMessage` → AI pipeline (Ctrl+W) → draft → scope → type → confirm → output → `git commit`.
- 3-stage AI pipeline (body → title → format) with per-stage retry, history, and live model picker.
- SQLite persistence for commits, drafts, AI calls, model rate-limits, model cache.
- Drafts: autosave (Ctrl+S), exit-time autodraft, `Ctrl+D` toggle in main list.
- Reword mode (`-w <hash>`) with safe empty-message handling for lazygit.
- Direct stdout mode (`-o`) for shell pipelines.
- Release mode (`-r`): commit picker → AI changelog refiner → Make build → GitHub upload.
- Headless `commitcraft ai ...` subcommands: `generate`, `regenerate`, `edit`, `show`, `list`, `promote`, `list-tags`, `list-addable-tags`, `add-tag`, `stage-partial`. JSON envelopes.
- Theming: `harmonized` (default), `charmtone`, `gruvbox`, `tokyonight`. Optional Nerd Fonts.
- Customizable commit-type palettes via TOML. Custom prompts via `~/.config/CommitCraft/prompts/`.
- Mention popup, command palette, keybindings popup, logs popup, version popup, model picker, scope/type pickers, tag pickers, diff view, edit-message popup, delete confirm, config popup, stage history popup.
- Pre-commit formatter chain (`gofumpt → goimports-reviser → golines`) installed via `make install-hooks`.
- Reword from existing release added to startup chooser (recent: c8a344e, 8044483, 8efb076, d00eb47).

## In Progress

- `feat/agent-cli-improvements` — branch-level umbrella in `context/specs/15-agent-cli-improvements.md` (8 units across two axes: CLI ergonomics + branch/release messages).
  - Unit 15 (umbrella plan): code-complete.
  - Item 1 (`ai context [--strict]`): shipped as v0.56.0, commit `c79dfac`.
  - Unit 16 (`ai verify`): shipped as v0.57.0, commit `cdd3671`.
  - Unit 17 (`ai merge`): code-complete, awaiting paired skill update.
  - Items 3-6, 8: spec-on-demand per option B; specs are written when the unit is implemented, not upfront.

## Next Up

- Smoke test unit 16, then update the skill at `~/.claude/skills/commitcraft/SKILL.md` to point step 6 ("Review the output") at `ai verify --id <ID>` instead of subjective review.
- Unit 17 candidates (any order): `ai context --model <id>`, `ai link-commit`, generic-title rejection, `ai generate --dry-run`, then move to Axis B with `ai merge` / `ai release`.
- Eventually merge `feat/agent-cli-improvements` → `main` (dogfooding `ai merge` once unit 21 lands).

## Open Questions

- Performance/quality bar for "success criteria" beyond functional. Currently only criterion 5 in `project-overview.md` is TBD.
- Is there a roadmap of pending improvements the user wants captured as units now, or do we wait for them to come up organically?
- Should the headless CLI gain a non-JSON `--plain` mode for direct shell use, or is JSON-only the contract?
- Is the GitHub release upload meant to support repos other than `Cerebellum-ITM/CommitCraftReborn`, or is it pinned by design?

## Architecture Decisions

- **Pure-Go SQLite** (`modernc.org/sqlite`) — no CGO so cross-compilation stays trivial.
- **Headless path short-circuits the TUI** in `main.go` before any Bubble Tea bootstrap, keeping `commitcraft ai ...` free of TUI flag noise (`-r`, `-o`, `-w`).
- **Schema migrations through `applySchemaMigrations`**, never `createTables` — protects existing user DBs.
- **Reword exit safety**: blank `FinalMessage` after exit prints stderr notice and exits 0, never `git commit --amend -m ""` (footgun documented in `main.go:142-154`).
- **Pre-commit hook formats Go files** with the conform.nvim chain to keep diffs clean across editors.

## Session Notes

- 2026-05-27 — **Unit 17 (`ai merge`) code-complete** on `feat/agent-cli-improvements`. New `commitcraft ai merge --branch <source> [--into main]` subcommand. Reuses `aiengine.RunRelease` (existing 3-stage release pipeline) on the commits returned by two new git helpers — `VerifyRev` and `GetCommitsBetween(workspace, target, source)`. Persists as a normal `storage.Commit` row with `Type="MERGE"`, `Scope=<branch>`, `Source="ai"`. Composed `final_message` matches the project's existing convention (`[MERGE] feat/foo: Title\n\nBody`). End-to-end smoke test on this very branch produced draft id=999 — verified clean except for `title_too_long_soft` warning (Llama-4-scout's title was 95 chars; agent can decide whether to `ai edit` it shorter or accept). `ai regenerate` not yet wired for merge drafts (would route through commit pipeline incorrectly) — documented as a known limitation, future unit will dispatch on draft type. v0.58.0. Spec at `context/specs/17-ai-merge.md`.
- 2026-05-27 — **Unit 16 (`ai verify`) code-complete** on `feat/agent-cli-improvements`. New deterministic checker `aiengine.VerifyFinalMessage` exposed via `commitcraft ai verify --id <ID>`. Eleven rules covering AI residue, template placeholders, code-fence wrappers, title format (`[TAG] scope:` with hard/soft length thresholds), empty/equal title-body, and duplicate body lines. Exit codes: 0 clean (or warnings-only by default), 4 errors (or any finding under `--strict-warnings`); 3 stays reserved for `ai context --strict`. Unit tests at `internal/aiengine/verify_test.go` cover every rule. Smoke-tested against draft 993 (the `ai context` commit) → clean. v0.57.0. Spec at `context/specs/16-ai-verify.md`.
- 2026-05-27 — **Branch `feat/agent-cli-improvements` opened**, umbrella plan in `context/specs/15-agent-cli-improvements.md`. Item 1 (`ai context [--strict]`) shipped as commit `c79dfac` (v0.56.0). Paired skill repo branch at `feat/agent-cli-improvements` documents the sub-agent invocation pattern + step 1.5 context gate + recoverability section.
- 2026-05-22 — **Unit 14 code-complete** on `feat/keymatches-migration`. Migrated every main-matcher `msg.String()` in `internal/tui/update_*.go` to `key.Matches`: `ctrl+f` filter cycle (workspace + release history + release picker), filter-bar `esc`/`enter`, panel-scroll (`pgup`/`pgdown`/`ctrl+up`/`ctrl+down`), release-pipeline stage controls (the original Unit 08 block at `update_release.go:374`), commit-pipeline `H` (`pipeline_update.go:123`), and the compose per-focus handlers (commit-type cycle, scope clear, keypoints nav, pipeline-models nav). Added `CycleFilterMode` + `ClearField` fields to `KeyMap`, populated `mainListKeys()` / `releaseMainListKeys()` / `releaseKeys()` / `writingMessageKeys()` / `pipelineKeys()` with the missing bindings. Rewrote `keybindings_popup.go` so the four per-state builders take the active `KeyMap` and pull key strings via `binding.Help().Key`; updated the caller in `update.go:1143` to pass `model.keys`. Final audit: `grep msg.String()` in `internal/tui/` returns only the documented carve-outs (mention `@`, scope-picker `e`/`enter`, transient popup closes, scroll inside history dual panel, global guards in `update.go`). v0.55.0.
- 2026-05-22 — **Unit 13 code-complete** on `feat/keymatches-migration`. Added `History` field to `KeyMap`, populated `releaseKeys()` + `viewPortKeys()` with the release pipeline stage controls (`r`/`1`/`2`/`3`/`H`/`pgup`/`pgdown`) that have been matched via raw `msg.String()` since Unit 08, extended `ShortHelp`/`FullHelp` for `History`, wrote the "key.Matches with keymap as single source of truth" rule into `context/code-standards.md`. v0.54.1. No behavior change yet — dispatch in `update_release.go:374` still uses the string switch; Unit 14 will migrate it.
- 2026-05-22 — **v0.54.0 shipped.** Merged `feat/release-config-polish` → `main` as `b854f66 [MERGE] feat/release-config-polish: Release config & changelog popups (v0.54.0)`. Cross-compiled three binaries via `make build_release`, tagged `v0.54.0` (annotated), pushed `main` + tag, published GitHub release with all three binaries at https://github.com/Cerebellum-ITM/CommitCraftReborn/releases/tag/v0.54.0. Feature branch deleted (local + remote).
- 2026-05-22 — **Units 11 + 12 code-complete** on `feat/release-config-polish`. Four commits land the popup polish plus the changelog popup: `21a0479` (initial Units 11+12), `10ff474` (token-mask root cause + popup sizing + list picker + nerd-font icons), `82477b9` (configured-state indicator on GH_TOKEN + palette icon spacing), `<this commit>` (spec post-implementation notes). Spec 11's Component B hypothesis (EchoMode masking failure) was wrong — the "g" was the bubbles `placeholderView` rendering one rune from `"ghp_..."` because no width was set. Fix: drop the placeholder; add a `✓ stored — type to replace` row when `detected.GhTokenSet`. Awaiting user review before merging to main as v0.54.0.
- 2026-05-19 — **Units 05, 09 already shipped** on this branch (commits `409e1f2`, `44646d1` + spec docs `d79b98b`, `a90307d`).
- 2026-05-19 — **Unit 06 withdrawn.** Started writing spec/impl, then a re-trace of `update.go:554`'s `case "Release Commit"` revealed it's dispatched from the *post-pipeline* popup in `stateReleaseBuildingText` (`update_release.go:434`), not from `stateReleaseMainMenu`. `releaseText` is always populated by the cascade before `createRelease` runs. The "bug" was a misread of the call graph. Reverted all changes; build plan annotated.
- 2026-05-19 — **Unit 07 trimmed.** `v0.51.2` (`bd41cf7`) already replaced `sh -c` with `exec.Command("gh", ...)`, guarded the asset walk on empty path, and switched to `--notes-file`. Remaining work is one `LevelInfo` status-bar message when uploading without assets.
- 2026-05-19 — **Unit 10 added.** Release configuration onboarding popup. Carries: (a) `GH_TOKEN` migration from TOML to `~/.config/CommitCraft/.env`, (b) auto-detect helpers for repo/branch/version/assets-path, (c) multi-field config popup with Tab-cycled fields, (d) auto-open before upload if any required field missing, (e) command-palette entry "Configure release", (f) refresh of the legacy `stateSettingAPIKey` view to use the same styled component for consistency.
- 2026-05-04 — Branch renamed `feat/release-merge-reword` → `feat/release-flow-cleanup` so it carries the 7-unit audit (the original reword work landed in commits `a70c8b7`, `7000c3d`, `c7a9654`).
- 2026-05-04 — **Unit 01 shipped** as `e1ccab5 [REM] release: remove cosmetic release/merge toggle`. v0.49.0.
- 2026-05-04 — **Spec-driven scaffolding imported** as `11fe653 [DOC] context: import spec-driven-dev scaffolding`. The five remaining context files (project-overview, ui-context, code-standards, ai-workflow-rules, progress-tracker) plus `context/specs/` (build plan, template, specs 01/02/03/04/08) plus the new "Spec-Driven Workflow" section in `CLAUDE.md` all land here. Until this commit, `context/` had been generated but kept untracked across sessions; persisting it so a fresh clone gets the project context without re-running the discovery flow.
- 2026-05-04 — **Unit 08 shipped** as `19e985f [ADD] release: wire per-stage retry into release pipeline view`. Real per-stage retry: `aiengine/release.go` split into `RunReleaseBody` / `RunReleaseTitle` / `RunReleaseRefine` primitives, `RunRelease` becomes a sequencer. TUI gets `iaReleaseCascade(model, from)`, `pipelineReleaseRetryStage(from)`, and `IaReleaseBuilderResultMsg.From` so partial cascades only push history for stages that actually re-ran. Wired `r`/`1`/`2`/`3`/`H`/`pgup`/`pgdn` in `updateReleaseBuildingText`. **Footgun discovered**: the active keymap in `stateReleaseBuildingText` is `releaseKeys()`, which doesn't define `Toggle` / `RerunStageN` / `PgUp` / `PgDown` — those fields are zero-value bindings, so `key.Matches` always returns false. Switched to raw `msg.String()` matching for the new bindings to avoid lying. Captured as a memory. v0.51.0.
- 2026-05-04 — **Unit 04 shipped** as `b51b076 [FIX] release: block create-release popup until pipeline finishes`. Guarded `Enter` in `stateReleaseBuildingText` behind `pipeline.allDone()` so the create-release popup can't open while a stage is running, was cancelled, or failed. Status bar surfaces a `LevelWarning` with the gating reason. v0.50.1.
- 2026-05-04 — **Unit 03 shipped** as `5c644b1 [ADD] release: cycle pipeline stage cards with Tab`. Tab/Shift+Tab now cycle stage cards in `stateReleaseBuildingText` via existing `pipelineModel.cycleFocus` (added a mirror `cycleFocusBackward`); Esc walks back to picker (cancels a running pipeline first); final card now lights up for release preset and pulls content from `releaseFinalOutput`. Status bar + `?` popup updated. Discovered mid-implementation that the release pipeline never had `r` / `1`-`3` / `H` / `pgup`-`pgdn` wiring — captured as Unit 08 (`release-pipeline-stage-controls`) instead of expanding scope. Stripped those keys from the popup/hints to avoid lying about non-functional bindings until Unit 08 ships. v0.50.0.
- 2026-05-04 — **Unit 02 shipped** as `e2086a0 [FIX] release: fix Selected-only filter showing empty list`. Root cause was NOT what the spec hypothesized: the sentinel `"\x00release-choose-selected-only\x00"` lost its NUL bytes through `textinput.Model.SetValue`, so `releaseChooseListFilter` never recognized it and fell into a fuzzy match against bogus text. Fix: plain-ASCII sentinel. Defense-in-depth additions (cursor→underlying index translation in toggle handler, source-of-truth re-stamp of `Selected` from `selectedCommitList`, canary log) kept as guardrails. Lesson: when stuffing data through `textinput`, assume control characters get stripped. v0.49.1.
- 2026-05-04 — Spec for Unit 01 (`remove-merge-toggle`) authored at `context/specs/01-remove-merge-toggle.md`. Scope strictly limited to the cosmetic toggle (`model.releaseMode`, `m` key, `ReleaseInput.Mode`, picker pill, pipeline-panel mode suffix, architecture doc/Mermaid). Out of scope: `model.releaseType` and `storage.Release.Type` (post-pipeline classification, persisted).
- 2026-05-04 — Brownfield init via `/spec-driven-dev init --from-code`. Six context files seeded from code reading; `CLAUDE.md` updated to reference the spec-driven workflow alongside existing project conventions.
- 2026-05-04 — Release-flow audit walkthrough Pantalla 1 → 2 → 3. Captured 7 actionable units in `context/specs/00-build-plan.md`. Bug summary:
  - **P1:** `gh release create` fails with `fork/exec /bin/sh: argument list too long` when `bin/` is empty (root cause: notes passed inline as argv). Same bug surfaces from Pantalla 3 entry point.
  - **P2:** `ctrl+e` "Selected only" shows empty list even with selections. `m` toggle release/merge had no real semantics — to be removed entirely. `Tab` from P3 should cycle pipeline stage cards, not return to picker.
  - **P3:** `Enter` while pipeline running advances to the next popup before stage 3 completes. Output viewport stays empty after pipeline finishes (final output not wired). `-r` direct launch skips the summary/preview view that `-w` rewrite-as-release passes through.
- Heads-up: `.commitcraft.toml` (gitignored) contains a `GH_TOKEN`. Treat the file as secret-bearing; never log it, never commit it.
