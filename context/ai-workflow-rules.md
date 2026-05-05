# AI Workflow Rules

## Approach

Build CommitCraft incrementally using a spec-driven workflow. Before any feature change, read the relevant context file(s) and the spec under `context/specs/NN-name.md`. Implement exactly what the spec specifies — no more, no less. Do not infer product behavior; if it is not in the spec or in `project-overview.md`, stop and ask.

## Scoping Rules

- Work on one unit at a time. A unit produces one visible, verifiable result (a new state, a new popup, a new headless subcommand, a migration step).
- Prefer small, verifiable increments. If a change cannot be exercised end-to-end with a single `make build && ./bin/commitcraft...` round, split it.
- Do not combine unrelated system boundaries in one step. Touching `internal/storage/` and `internal/tui/` in the same change is fine when wiring a feature; touching `internal/api/` *and* `internal/cli/ai/` *and* the release flow in one step is too broad.
- A bug fix does not need surrounding refactors. Don't bundle cleanup into an unrelated PR.

## When to Split Work

Split if the change combines:

- A new TUI state plus a new database migration plus a new Groq endpoint.
- Multiple new headless subcommands at once (do them one at a time).
- Theme changes plus behavior changes (theme tweaks ship on their own commit).
- Anything not clearly specified in the active spec.

## Handling Missing or Ambiguous Requirements

- Do not invent behavior. If a key binding, copy text, color, or flow is unspecified, **stop and ask the user** — don't pick a default.
- If a requirement is ambiguous, resolve it by editing the relevant context file (`project-overview.md`, `ui-context.md`, etc.) *before* writing implementation code.
- If a requirement is missing entirely, add it as an open question in `progress-tracker.md` and surface it to the user before continuing.

## Protected Areas

Do not modify these without an explicit instruction:

- `internal/storage/database.go::createTables` — original schema only. New schema goes through `applySchemaMigrations`.
- `go.sum`, `go.mod` — only touch when adding a deliberate dependency referenced in the active spec.
- `bin/` — built binaries; never edit by hand.
- `.commitcraft.toml` — repo-local config; contains user secrets (`GH_TOKEN`).
- `internal/config/prompts/*.prompt.tmpl` — embedded defaults. Edit these only when the change is itself a prompt-engineering task; otherwise let users override via `~/.config/CommitCraft/prompts/`.
- Any `charm.land/...` cached library under `~/go/pkg/mod/` — read-only references for documentation, never modify.

## Hard Constraints from `architecture.md`

These are invariants — do not violate them under any circumstance:

1. `View` is pure. Focus / blur / async work happens in `Update`, never in `View`.
2. DB schema changes go through `applySchemaMigrations`, never `createTables`.
3. No raw colors in render code. Use `model.Theme`.
4. Help-line and popup hints render through `theme.AppStyles().Help` (`ShortKey` / `ShortDesc` / `ShortSeparator`).
5. The headless `commitcraft ai ...` path never spawns the TUI.
6. CommitCraft never runs `git push`. Only `git commit` and `git commit --amend`.
7. Reword never amends with an empty message.
8. Every version bump in `cmd/cli/main.go` requires a `CHANGELOG.md` entry.

## Keeping Docs in Sync

When implementation changes any of the following, update the matching context file in the same change:

- New TUI state, new system boundary, new external dependency → `architecture.md`.
- New invariant or storage decision → `architecture.md` (Invariants / Storage Model).
- New theme token, new layout pattern → `ui-context.md`.
- New Go convention or new file-organization rule → `code-standards.md`.
- Feature scope shift → `project-overview.md`.

## Verification Before Moving to the Next Unit

Run all of these:

1. `go build ./...` passes with no errors.
2. `make build` (Makefile target) produces a binary in `bin/`.
3. The unit's "Verify when done" checklist in `context/specs/NN-*.md` is fully checked.
4. Manual TUI smoke test for any UI-touching change: launch `./bin/commitcraft_<host>`, walk the affected flow, watch for crashes, confirm theme rendering.
5. No invariant in `architecture.md` was violated.
6. `cmd/cli/main.go` version is bumped and `CHANGELOG.md` has a new top entry in English.
7. `progress-tracker.md` updated: unit moved to Completed, next unit set as In Progress / Next Up.

## Commit & Release Discipline

- Never run `git commit` directly. Use the `commitcraft` skill (CommitCraft itself, in `commit_craft_reborn` mode if running locally) — it handles staging, message generation, and the final commit. **Never push.**
- `make install-hooks` must be run once after clone so the formatter `pre-commit` hook is active.
- Cross-compile via the existing `make` targets; do not invent new build commands.
