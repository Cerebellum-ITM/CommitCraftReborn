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

- None — Units 01, 02, 03, 04, 08 shipped on `feat/release-flow-cleanup`.

## Next Up

- Back on the original plan: Units 05 (`wire-final-output-to-viewport`), 06 (`converge-report-screen`), 07 (`release-upload-empty-bin-guard`).
- See `context/specs/00-build-plan.md` for the full ordered list (now 8 units).

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

- 2026-05-04 — Branch renamed `feat/release-merge-reword` → `feat/release-flow-cleanup` so it carries the 7-unit audit (the original reword work landed in commits `a70c8b7`, `7000c3d`, `c7a9654`).
- 2026-05-04 — **Unit 01 shipped** as `e1ccab5 [REM] release: remove cosmetic release/merge toggle`. v0.49.0.
- 2026-05-04 — **Unit 08 shipped** (about to commit). Real per-stage retry: `aiengine/release.go` split into `RunReleaseBody` / `RunReleaseTitle` / `RunReleaseRefine` primitives, `RunRelease` becomes a sequencer. TUI gets `iaReleaseCascade(model, from)`, `pipelineReleaseRetryStage(from)`, and `IaReleaseBuilderResultMsg.From` so partial cascades only push history for stages that actually re-ran. Wired `r`/`1`/`2`/`3`/`H`/`pgup`/`pgdn` in `updateReleaseBuildingText`. **Footgun discovered**: the active keymap in `stateReleaseBuildingText` is `releaseKeys()`, which doesn't define `Toggle` / `RerunStageN` / `PgUp` / `PgDown` — those fields are zero-value bindings, so `key.Matches` always returns false. Switched to raw `msg.String()` matching for the new bindings to avoid lying. Captured as a memory. v0.51.0.
- 2026-05-04 — **Unit 04 shipped** (about to commit). Guarded `Enter` in `stateReleaseBuildingText` behind `pipeline.allDone()` so the create-release popup can't open while a stage is running, was cancelled, or failed. Status bar surfaces a `LevelWarning` with the gating reason. v0.50.1.
- 2026-05-04 — **Unit 03 shipped** (about to commit). Tab/Shift+Tab now cycle stage cards in `stateReleaseBuildingText` via existing `pipelineModel.cycleFocus` (added a mirror `cycleFocusBackward`); Esc walks back to picker (cancels a running pipeline first); final card now lights up for release preset and pulls content from `releaseFinalOutput`. Status bar + `?` popup updated. Discovered mid-implementation that the release pipeline never had `r` / `1`-`3` / `H` / `pgup`-`pgdn` wiring — captured as Unit 08 (`release-pipeline-stage-controls`) instead of expanding scope. Stripped those keys from the popup/hints to avoid lying about non-functional bindings until Unit 08 ships. v0.50.0.
- 2026-05-04 — **Unit 02 shipped** (about to commit). Root cause was NOT what the spec hypothesized: the sentinel `"\x00release-choose-selected-only\x00"` lost its NUL bytes through `textinput.Model.SetValue`, so `releaseChooseListFilter` never recognized it and fell into a fuzzy match against bogus text. Fix: plain-ASCII sentinel. Defense-in-depth additions (cursor→underlying index translation in toggle handler, source-of-truth re-stamp of `Selected` from `selectedCommitList`, canary log) kept as guardrails. Lesson: when stuffing data through `textinput`, assume control characters get stripped. v0.49.1.
- 2026-05-04 — Spec for Unit 01 (`remove-merge-toggle`) authored at `context/specs/01-remove-merge-toggle.md`. Scope strictly limited to the cosmetic toggle (`model.releaseMode`, `m` key, `ReleaseInput.Mode`, picker pill, pipeline-panel mode suffix, architecture doc/Mermaid). Out of scope: `model.releaseType` and `storage.Release.Type` (post-pipeline classification, persisted).
- 2026-05-04 — Brownfield init via `/spec-driven-dev init --from-code`. Six context files seeded from code reading; `CLAUDE.md` updated to reference the spec-driven workflow alongside existing project conventions.
- 2026-05-04 — Release-flow audit walkthrough Pantalla 1 → 2 → 3. Captured 7 actionable units in `context/specs/00-build-plan.md`. Bug summary:
  - **P1:** `gh release create` fails with `fork/exec /bin/sh: argument list too long` when `bin/` is empty (root cause: notes passed inline as argv). Same bug surfaces from Pantalla 3 entry point.
  - **P2:** `ctrl+e` "Selected only" shows empty list even with selections. `m` toggle release/merge had no real semantics — to be removed entirely. `Tab` from P3 should cycle pipeline stage cards, not return to picker.
  - **P3:** `Enter` while pipeline running advances to the next popup before stage 3 completes. Output viewport stays empty after pipeline finishes (final output not wired). `-r` direct launch skips the summary/preview view that `-w` rewrite-as-release passes through.
- Heads-up: `.commitcraft.toml` (gitignored) contains a `GH_TOKEN`. Treat the file as secret-bearing; never log it, never commit it.
