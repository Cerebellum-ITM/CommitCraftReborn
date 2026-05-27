# Unit 15: agent-cli-improvements (umbrella plan)

## Goal

Branch-level plan for `feat/agent-cli-improvements`. The branch's purpose is
to make the headless `commitcraft ai …` surface a better tool for AI agents
that delegate commit work to a sub-agent. Each item below is its own commit
inside the branch and gets a dedicated spec written **right before** its
implementation (option B — detailed specs evolve with what we learn from
the previous unit).

The branch is grouped along two axes:

- **Axis A — Ergonomics for the headless CLI**: deterministic gates,
  programmatic verification, and offline helpers that let the agent
  decide what to do without having to read the full diff.
- **Axis B — Branch / release messages**: expose the merge-commit and
  release-notes generation paths that currently live only in the TUI.

## Items

### Axis A — Ergonomics

1. **Pre-flight context check** — `ai context [--strict] [--model <id>]`.
   Estimates the Change Analyzer payload against the configured (or
   overridden) model's cached context window. ✅ Shipped as `c79dfac`
   (v0.56.0). The `--model` flag remains as a follow-up (item 3).

2. **Programmatic message verification** — `ai verify --id <ID>`.
   Runs deterministic rules against a draft's `final_message`: AI
   residue strings, title format (`[TAG] scope:`), title length,
   code-fence wrappers, mention-line duplication, empty pieces. JSON
   output with `has_issues` + `findings[]`. Exit 0 clean, exit 4 when
   findings are present. **This is the next unit.**

3. **Context check against alternative models** — `ai context --model <id>`
   flag. Lets the agent compare "would this fit in `llama-3.3-70b-versatile`?"
   without rewriting config. Small add on top of unit 16.

4. **Commit ↔ draft linking** — `ai link-commit --id <draft> --hash <h>` +
   `ai show --commit <hash>`. Persists the git hash on the draft so the
   keypoints (and per-stage telemetry) are recoverable by hash after the
   fact, not just by draft id. Closes the "recover keypoints from a past
   commit" loop.

5. **Reject generic titles** — heuristic gate in `ai generate` (or
   surfaced as a finding in `ai verify`). Catches titles like `update X`,
   `document Y`, `add Z guides` — generic verbs that signal the model
   didn't anchor on the keypoints.

6. **Dry-run mode** — `ai generate --dry-run`. Runs the pipeline (or a
   subset) without persisting a draft row. For agents that want to
   experiment with keypoint phrasings without polluting the drafts list.

### Axis B — Branch / release messages

7. **Merge-commit drafts** — `ai merge --branch <name>`. Generates a
   merge-commit message from `git log main..<branch>` with the project's
   `[MERGE] branch_name: TITLE` schema. Persists as a draft so `ai edit`
   / `ai regenerate` work the same way as on staged commits.

8. **Release-notes drafts** — `ai release --version vX.Y.Z`. Exposes the
   same generator the TUI's release mode uses (commit range → AI changelog
   refiner → final notes), persisting as a draft. Optional follow-up:
   `ai release publish --id <ID>` as a **separate, opt-in** subcommand
   that wraps the `gh release create` step. Keeping publish separate is
   intentional — the agent can drive everything up to promote without
   ever needing GH credentials.

## Order

The order is the implementation order, but each unit's spec is written
fresh when we arrive at it. Units 1 and 16 (`ai verify`) are the only
ones with fixed specs so far (15 retroactively documents item 1; 16
covers `ai verify`).

The dependency graph is shallow:

- Unit 16 (`ai verify`) depends on nothing new.
- Unit 17 (`ai context --model`) is trivial; can ship any time after unit 1.
- Unit 18 (`ai link-commit`) needs a small schema migration; otherwise standalone.
- Unit 19 (`reject generic titles`) is easier to land as a `verify` rule
  rather than a `generate` gate — sequencing it after unit 16 saves work.
- Unit 20 (`--dry-run`) is standalone.
- Unit 21 (`ai merge`) and unit 22 (`ai release`) share infrastructure
  for "feed a commit range to the analyzer" — implement merge first,
  then release reuses the helper.

## Verify when done (branch-level)

- [ ] Every unit ships with a `## vX.Y.Z` CHANGELOG entry and version bump.
- [ ] `go build ./...` and `go vet ./...` clean at branch tip.
- [ ] The skill at `~/.claude/skills/commitcraft/SKILL.md` is updated in
      its own paired commit (in the skill repo, also on
      `feat/agent-cli-improvements`) whenever a unit changes the
      contract.
- [ ] After all units land, `feat/agent-cli-improvements` merges to
      `main` via the eventual `ai merge` flow (dogfood).
