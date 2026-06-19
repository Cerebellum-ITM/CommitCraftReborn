# Changelog

All notable changes to CommitCraft are documented here. Newest version on top.

## v0.68.1 — 2026-06-19

Documentation: replaced the JSON-output CLI demo GIFs with a single hero GIF of
the **real TUI** generating a commit message. No production code changes.

- The hero records the actual Compose flow: new commit → pick type (`^T`) →
  pick scope (`^P`) → describe the change → generate with `^W` → the AI
  suggestion panel fills in.
- Recorded fully offline: `demo/setup-sandbox.sh` now also starts
  `demo/mock-groq.py`, a local Groq stand-in, and points the app at it via
  `COMMITCRAFT_GROQ_BASE_URL` (added in v0.68.0). No network, no API key, no
  quota; invented data, faithful UI.
- Removed the old `context`/`tags`/`keys` JSON GIFs and their tapes; updated
  the root `README.md` and `demo/README.md` accordingly.

### Usage

Regenerate from the repo root: `vhs demo/tapes/hero.tape`. Requires `vhs`,
`ttyd`, `ffmpeg`, `python3`, `sqlite3`, `curl`, `jq`, and a Nerd Font. See
[`demo/README.md`](demo/README.md).

## v0.68.0 — 2026-06-19

Added a `COMMITCRAFT_GROQ_BASE_URL` environment override for the Groq API root.

- `internal/api/groq.go` now resolves the base URL through `groqBaseURL()`,
  which reads `COMMITCRAFT_GROQ_BASE_URL` and falls back to the real
  `https://api.groq.com/openai/v1` when unset. Both the chat-completions and
  models endpoints honor it.
- Enables pointing the app at an OpenAI-compatible gateway, a self-hosted
  proxy, or a local mock (used by the demo recordings) without recompiling.
- Fully backward compatible: with the variable unset, behavior is unchanged.

### Usage

```sh
export COMMITCRAFT_GROQ_BASE_URL="http://127.0.0.1:8899/openai/v1"
commitcraft            # all Groq calls now hit that base URL instead
```

Unset the variable (the default) to talk to the real Groq API.

## v0.67.1 — 2026-06-19

Documentation: animated demo GIFs for the README, recorded with
[VHS](https://github.com/charmbracelet/vhs). No code changes.

- Added `demo/` with VHS tapes (`demo/tapes/`), the rendered GIFs (`demo/gifs/`),
  a `setup-sandbox.sh` that sandboxes `HOME` to `/tmp` (invented keys, throwaway
  git repo, seeded model-context metadata), and a `demo/README.md`.
- GIFs are real recordings of the actual binary running **offline** — the hero
  uses delegate mode (`ai generate --agent` → `ai submit`), so no Groq calls are
  made and nothing private is shown.
- Embedded a hero GIF at the top of `README.md` plus inline GIFs for
  `ai context`, `ai list-tags`, and `ai key show`/`swap`.

### Usage

Regenerate from the repo root: `vhs demo/tapes/<name>.tape` (or loop over
`demo/tapes/*.tape`, skipping `_setup.tape`). Requires `vhs`, `ttyd`, `ffmpeg`,
`jq`, `sqlite3`, and a Nerd Font. See [`demo/README.md`](demo/README.md).

## v0.67.0 — 2026-06-13

Agent **delegate mode** for the headless `ai` CLI. When CommitCraft is driven by an AI agent (the `commitcraft` skill), the message-generation stages can now **bypass the Groq API entirely**: instead of making 3–4 serial API calls, the CLI emits a **prompt bundle** (the same prompts, filled with the diff/keypoints/tag/scope) and the already-running agent produces the message itself, then returns it through the new `commitcraft ai submit` subcommand. This removes the API-latency/queue bottleneck from the agent flow. Backward compatible — with no `[agent]` config and no `--agent` flag, the Groq pipeline is unchanged.

- New `[agent]` config table: `mode` (`""`/`groq` default, or `delegate`) and `strategy` (`single` default, or `staged`).
- Per-call flags on `generate` / `regenerate` / `merge` / `release`: `--agent`, `--no-agent`, `--agent-strategy single|staged`. `--agent` + `--no-agent` together is a usage error.
- **Hybrid architecture**: `single` fills one new unified prompt (`agent_commit.prompt` / `agent_release.prompt`) for a one-pass result; `staged` hands over the original per-stage prompts for the agent to follow internally. Both submit exactly once — no extra round-trips.
- New `commitcraft ai submit`: reads the agent's message as JSON from stdin (or `--input-file`), re-reads the staged diff, composes `final_message`, runs the deterministic verifier, and persists the draft. The response carries a `verify` block so the agent sees quality findings inline. `id>0` updates an existing draft (regenerate); `kind:"release"` persists a `[MERGE]`/`[RELEASE]` row.
- The unified prompts and bundle `instructions` both restate the standing rule: the commit message is always **English**.
- Covers `generate`, `regenerate`, `merge`, `release`, and the changelog refiner (when changelog is enabled, the bundle injects `CHANGELOG_CONTEXT` so the agent produces the entry + mention line too).

### Usage

```sh
# enable globally in ~/.config/CommitCraft/config.toml
[agent]
mode     = "delegate"   # "" | "groq" (default) | "delegate"
strategy = "single"     # "single" (default) | "staged"

# or per call:
commitcraft ai generate --agent -k "keypoint" -t ADD -s scope
#   → emits a delegate bundle (prompts + diff + inputs); NO Groq call, NO draft.

# the agent writes the message following the bundle prompt(s), then:
echo '{"kind":"commit","tag":"ADD","scope":["scope"],"keypoints":["keypoint"],
       "title":"...","body":"..."}' | commitcraft ai submit
#   → persists the draft, returns the envelope + a `verify` block.

commitcraft ai promote --id <ID>     # unchanged

# regenerate / release variants:
commitcraft ai regenerate --id <ID> --agent          # bundle carries action=regenerate + id
echo '{"kind":"commit","id":<ID>,"title":"...","body":"..."}' | commitcraft ai submit
commitcraft ai merge --agent --branch feat/foo       # kind:"release" bundle
commitcraft ai generate --agent --agent-strategy staged ...   # per-stage prompts
```

## v0.66.0 — 2026-05-29

Dual Groq API key support. Two key slots — **user** and **ai** — with exactly one active at a time, managed via the new `commitcraft ai key` subcommand. Both the TUI and every `commitcraft ai …` command resolve their key from the active slot. Lets you keep a second key on hand and swap to it manually when the free tier rate-limits the first one (the original blocker for the agent-driven `ai merge`).

- `.env` storage: `GROQ_API_KEY` (user slot, unchanged), `GROQ_API_KEY_AI` (ai slot), `GROQ_ACTIVE_KEY` (`user`|`ai`, defaults to user). Existing single-key setups are unaffected.
- No silent fallback: if the active slot is empty, the usual "API key not provided" error fires — `ai key show` reveals the state.
- On a Groq `429`, `ai generate` / `regenerate` / `merge` / `release` now return `code: "rate_limited"` with a hint naming the active slot and the swap command, instead of a generic `api_error`. No automatic retry.
- `.env` mutation logic now lives in `config.SaveEnvVar` (single source of truth; the TUI delegates to it).

### Usage

```sh
commitcraft ai key                      # show active slot + which slots are set (JSON, no secrets)
commitcraft ai key set --slot ai        # prompts for the key without echo (or pass --value <key>)
echo "$KEY" | commitcraft ai key set --slot ai   # scripted: reads the key from stdin
commitcraft ai key swap                 # toggle user<->ai (errors if the target slot is empty)
commitcraft ai key use --slot user      # activate a slot explicitly
```

When a run is rate-limited you'll see `{"code":"rate_limited", ...}`; run `commitcraft ai key swap` and retry.

## v0.65.0 — 2026-05-29

Add `--dry-run` flag to `commitcraft ai generate`. Runs the full 3-stage AI pipeline (real Groq calls) and returns the same JSON output as a normal generate, but skips all DB writes — no draft row, no telemetry. Returns `"id": 0` and `"status": "dry_run"` in the envelope. Designed for agents that want to iterate on keypoint phrasings without polluting the drafts list.

### Usage

```sh
commitcraft ai generate --dry-run -k "keypoint 1" -t ADD -s ai
# → JSON with id=0, status=dry_run, final_message populated
# → no row created in the DB
```

`ai verify --id <id>` does not apply to dry-run output (no id). Inspect `final_message` directly from the JSON. When satisfied, re-run without `--dry-run` to persist.

## v0.64.0 — 2026-05-28

Add `generic_title` warning rule to `ai verify`. Flags title text (the portion after `[TAG] scope: `) that starts with a generic action verb and has ≤ 3 words total — patterns like `"update docs"`, `"fix bug"`, `"add feature"` that signal the model ignored the keypoints. Severity is warning, not error; the agent can decide to patch or accept. Titles with 4+ words or non-generic leading words pass cleanly.

### Usage

No new flags. The finding appears automatically in `commitcraft ai verify --id <ID>` output when the title text is too generic:

```json
{
  "rule": "generic_title",
  "severity": "warning",
  "message": "Title text is likely too generic (\"fix bug\"). Add specifics about what changed.",
  "location": "title"
}
```

Fix with `commitcraft ai edit --id <ID> --title "[FIX] scope: fix null-pointer in release dispatch"`.

## v0.63.0 — 2026-05-28

Add `--model <id>` flag to `commitcraft ai context`. Lets an agent (or the user) compare whether the staged diff fits inside an alternative model's context window without editing the config. The payload estimate is model-agnostic (chars/4 heuristic against the fixed system prompt), so only the context-window lookup changes. If the supplied model ID is not in the local `groq_models_cache`, `context_window` returns 0 and `fits` / `usage_pct` are null — same behavior as today for any uncached model.

### Usage

```sh
# Check against a specific model without changing config
commitcraft ai context --model llama-3.3-70b-versatile

# Combine with --strict to gate on the alternative model
commitcraft ai context --model llama-3.3-70b-versatile --strict
```

## v0.62.0 — 2026-05-28

Render the `AI` / `TUI` source pill on the Releases view, mirroring what already exists on every row of the commits History tab. TUI-created releases show the `TUI` pill; drafts produced by `commitcraft ai release` / `ai merge` now show the `AI` pill instead of going unmarked. Pure rendering change — the underlying `releases.source` column was added in v0.61.0.

### Usage

No configuration or key changes. The pill appears automatically on every row of the Releases view. Legacy releases (pre-v0.61.0) were backfilled to `source = 'tui'` by the v0.61.0 migration and render as `TUI`.

## v0.61.0 — 2026-05-28

Realign storage of release-pipeline drafts. Until v0.60.0 the headless `ai release` and `ai merge` subcommands persisted to the `commits` table — a shortcut documented as a caveat in their CHANGELOG entries that bit users immediately: `[RELEASE]` / `[MERGE]` rows showed up in the TUI's commits History tab when they were supposed to live in the Releases view. This release moves both subcommands to the `releases` table where the TUI already keeps every row produced by the release pipeline.

The CLI surface is unchanged. `ai release --version`, `ai merge --branch`, and the shared `ai show / edit / verify / promote / link-commit / list` subcommands all behave the same from the agent's perspective. Implicit dispatch handles the table routing: each shared subcommand looks up the id in `commits` first and falls back to `releases` on miss. The JSON envelope gains a `kind` field (`"commit"` or `"release"`) plus explicit `branch` / `version` (omitempty) so consumers can disambiguate without parsing scope.

**Collision-safe lookups via `--kind`**: because the two tables auto-increment from their own sequences, the same numeric id can exist in both (e.g. `commits.id = 40` and `releases.id = 40`). Without help, dispatch hits `commits` first — fine when no collision exists, dangerous when one does (the wrong row gets edited/promoted). To eliminate the risk, every shared subcommand accepts an optional `--kind commit|release` flag that forces the lookup to one table. Agents that persist the `(id, kind)` pair from the JSON envelope of `ai release` / `ai merge` should pass `--kind release` to every subsequent call; agents that work only with `ai generate` rows can pass `--kind commit` (or rely on the auto-probe, which favors commits anyway).

Storage changes (idempotent migrations via `applySchemaMigrations`):

- `releases.source TEXT NOT NULL DEFAULT 'tui'` — discriminator for the upcoming TUI badge (unit 21).
- `releases.status TEXT NOT NULL DEFAULT 'completed'` — existing TUI rows backfill to completed; headless drafts start at `'draft'` and flip via `ai promote`.
- `releases.commit_hash TEXT NOT NULL DEFAULT ''` — `ai link-commit` now works on MERGE rows the same way it does on regular commits (a merge produces a real git commit whose hash is worth recalling).

New `storage` methods: `GetReleaseByID`, `GetReleasesByStatus`, `SaveReleaseDraft`, `FinalizeRelease`, `LinkReleaseHash`. Existing `CreateRelease` / `GetLatestRelease` / `GetReleases` queries updated to include the three new columns.

`ai edit` for release rows accepts only `--title` and `--body`. Passing `--scope`, `--tag`, or `--changelog` returns exit code 2 with `unsupported_field_for_release` — those fields don't map cleanly onto release rows (scope is implicit from branch/version, type rarely changes, releases don't carry a CHANGELOG mention).

**Orphaned legacy rows**: drafts created by the previous units that wrote to the `commits` table (e.g. ids 999, 1000, 1002, 1003, 1015 in the dev environment) stay where they are. They remain accessible via `ai show --id <old>` because the dispatch hits `commits` first, but they no longer surface in the TUI's Releases view. No retroactive migration — the cleanest cut is to leave history alone and let `ai release` / `ai merge` produce correctly-placed rows going forward.

### Usage

The agent's flow is unchanged:

```sh
commitcraft ai release --version v0.61.0                     # writes to releases table
commitcraft ai verify --id <id> --kind release               # disambiguates collisions
commitcraft ai edit --id <id> --kind release --title "..."   # title/body only on releases
commitcraft ai promote --id <id> --kind release              # FinalizeRelease

commitcraft ai merge --branch feat/foo                       # also releases table now
commitcraft ai link-commit --id <id> --kind release --hash "$(git rev-parse HEAD)"

commitcraft ai show --id <id> --kind release                 # JSON has kind="release"
commitcraft ai list                                          # commits + releases merged
```

The follow-up (unit 21) adds the `UI / AI` source badge to the TUI's Releases view, mirroring what already exists on the commits History tab.

## v0.60.0 — 2026-05-27

Close the loop between a CommitCraft draft and the git commit it became. Adds a `commit_hash` column to the `commits` table (via `applySchemaMigrations`, default `''`), a new `commitcraft ai link-commit --id <draft> --hash <git>` subcommand to write the hash onto an existing row, and a new `commitcraft ai show --commit <hash>` lookup that resolves a short or full hash to the row whose keypoints/telemetry came from it.

The motivation: today the only way to recover a past commit's keypoints is to remember the draft id. Three weeks later, looking at `git log` and wondering *"what was I thinking when I made this commit?"* required guessing. With linking in place, `git log` is the only id you need — `ai show --commit <hash>` returns the full JSON envelope (keypoints, summary, stage telemetry).

Implementation:

- **Schema migration** adds `commit_hash TEXT NOT NULL DEFAULT ''` to `commits`. Idempotent via the existing duplicate-column guard. Existing rows backfill to empty string; they remain queryable by id and stay absent from `--commit` lookups until linked.
- **`storage.Commit.CommitHash`** field; `commitJSON` envelope gains `commit_hash` (with `omitempty` so unlinked rows don't surface a noisy empty field).
- **`db.LinkCommitHash(id, hash)`** writes the column; `db.GetCommitsByHashPrefix(prefix)` reads it. Prefix must be ≥4 chars to avoid pathological scans; the CLI handles the ambiguity-check on top.
- **`git.ResolveCommitHashAt(workspace, rev)`** — workspace-aware sibling of `ResolveCommitHash`, used by `ai link-commit` to honor a `--workspace` flag while still working from any cwd.
- **`ai show`** now takes either `--id` or `--commit` (mutually exclusive). Ambiguous hash prefix → exit 1 with `ambiguous_hash` and the candidate ids in the error JSON.

### Usage

Manual link after a commit:

```sh
git commit -m "..."
commitcraft ai link-commit --id <draft_id> --hash $(git rev-parse HEAD)
```

Recall a commit's keypoints later:

```sh
commitcraft ai show --commit cdd3671 | jq .keypoints
```

The skill is updated in a paired commit to run `ai link-commit` automatically after every successful `git commit`, so the user never has to think about ids again.

Exit codes:

- `0` — link succeeded (a stderr warning still appears if overwriting a prior link).
- `1` — bootstrap / not-found / db errors; `ambiguous_hash` on `show --commit` falls here.
- `2` — usage errors (missing `--id` / `--hash`, both `--id` and `--commit` on `show`, hash rev-parse failure).

## v0.59.0 — 2026-05-27

Add `commitcraft ai release --version <vX.Y.Z> [--from <ref>] [--to <ref>]`: a headless subcommand that generates a `[RELEASE]` draft from the commits in `<from>..<to>` using the existing `aiengine.RunRelease` pipeline (same engine `ai merge` and the TUI release mode use). Persists as a normal `storage.Commit` row with `Type="RELEASE"` and `Scope=<version>` so `ai edit` / `ai show` / `ai verify` / `ai promote` work unchanged.

This unit only **drafts** the release notes. Publishing (`gh release create`, tag push, binary upload) stays a follow-up subcommand (`ai release publish --id <ID>`) so the agent can stop at promote without needing GH credentials.

Defaults:

- `--from` falls back to the most recent tag (via `git tag --sort=-v:refname`). If the repo has no tags, `--from` becomes required (exit 2 with `no_base_ref`).
- `--to` defaults to `HEAD`.

Storage caveat documented for the user: the TUI's release flow persists to a separate `releases` table (`storage.Release`); the headless `ai release` writes to the regular `commits` table. The two surfaces don't see each other's drafts today — a future unit can bridge them, but the divergence is intentional for v1 because it lets us reuse every existing subcommand on the new drafts without schema work.

Refactor: extracted `projectToReleaseCommits` + `serializeCommitRange` + `lastTagAt` from `merge.go` into a new `internal/cli/ai/range_helpers.go` so `release.go` reuses them without duplication.

### Usage

```
commitcraft ai release --version v0.59.0
commitcraft ai verify --id <id>
commitcraft ai edit --id <id> --body -    # if trimming commits manually
commitcraft ai promote --id <id>

# Eventually (next unit):
commitcraft ai release publish --id <id>  # not implemented yet
```

Exit codes: `0` success, `1` runtime errors (api, db, git, no commits), `2` usage errors (missing `--version`, no tags + no `--from`, invalid refs).

## v0.58.0 — 2026-05-27

Add `commitcraft ai merge --branch <source> [--into <target>]`: a headless subcommand that generates a `[MERGE]` draft from the commits between `<into>..<branch>` using the existing `aiengine.RunRelease` pipeline (same 3-stage body → title → refine flow the TUI uses for release notes). Persists as a normal `storage.Commit` row with `Type="MERGE"` and `Scope=<branch>` so every existing subcommand (`ai edit`, `ai show`, `ai promote`, `ai verify`) works on the draft unchanged.

Two new git helpers in `internal/git/git.go`:

- `VerifyRev(workspace, rev)` — wraps `git rev-parse --verify <rev>^{commit}` to reject missing or non-commit refs.
- `GetCommitsBetween(workspace, target, source)` — runs `git log --reverse --pretty=format:%h%x00%ad%x00%s%x00%b%x1f <target>..<source>` and parses each record into a `CommitRange`. Used as input to the release pipeline.

`ai regenerate` does **not** yet support merge drafts (it routes through the commit pipeline and would produce garbage). For tweaks use `ai edit`; for a clean re-run invoke `ai merge` again. A future unit will teach `ai regenerate` to dispatch on draft type.

### Usage

```
commitcraft ai merge --branch feat/agent-cli-improvements
commitcraft ai verify --id <id>
commitcraft ai edit --id <id> --title "..."          # if needed
commitcraft ai promote --id <id>

git checkout main
git merge --no-ff feat/agent-cli-improvements \
  -m "$(commitcraft ai show --id <id> | jq -r .final_message)"
```

Flags:

- `--branch <name>` (required) — source branch to summarize.
- `--into <name>` (default `main`) — target branch the merge is going into.
- `--workspace <path>` (default cwd) — repo path.

Exit codes: `0` success, `1` runtime errors (api, db, git, no commits in range), `2` usage errors (missing/invalid flags, ref not found).

## v0.57.0 — 2026-05-27

Add `commitcraft ai verify --id <ID>`: a deterministic, offline checker that scans a draft's composed `final_message` (the same text that would go into `git commit`) for AI-residue phrases, structural defects, and trailer hygiene problems. The verifier lives in `internal/aiengine/verify.go` as a pure function `VerifyFinalMessage(string) VerifyReport` so the TUI can surface findings later without re-implementing rules.

Rule set (all deterministic, no Groq call, no diff dependency):

- `ai_residue_phrase` (error) — known leakage strings like `here is the commit message`, `paragraph 1`, `as an ai`.
- `template_placeholder` (error) — literal `<title>` / `{body}` / `<keypoints>` etc. surviving into the message.
- `code_fence_wrapper` (error) — title or body wrapped in triple backticks.
- `title_format_missing_tag` (error) — first line does not start with `[TAG]`.
- `title_format_missing_scope` (warning) — has `[TAG]` but not the `[TAG] scope: …` shape.
- `title_too_long_hard` (error) — title > 100 chars.
- `title_too_long_soft` (warning) — title > 72 chars (GitHub convention).
- `empty_title` (error) / `empty_body` (warning) / `title_equals_body` (error).
- `duplicate_line_in_body` (warning) — any non-empty, non-separator line repeated verbatim.

Hallucinated-paths / wrong-language checks are intentionally **out of scope** for this iteration; they'll come back when the upcoming `ai link-commit` work persists a per-draft file list.

### Usage

```
commitcraft ai verify --id 42
commitcraft ai verify --id 42 --strict-warnings
```

Output is a JSON `VerifyReport` on stdout:

```json
{
  "has_errors": false,
  "has_warnings": true,
  "findings": [
    {
      "rule": "duplicate_line_in_body",
      "severity": "warning",
      "message": "Line repeated 2× in body: Updated CHANGELOG.md",
      "location": "body"
    }
  ]
}
```

Exit codes:

- **0** — clean, or only warnings without `--strict-warnings`.
- **1** — bootstrap error or draft not found.
- **2** — usage error (missing `--id`).
- **4** — at least one finding with `severity: error`, or any finding when `--strict-warnings` is set. (Exit code 3 is reserved for `ai context --strict` so the two gates remain distinguishable.)

## v0.56.0 — 2026-05-27

Add a pre-flight context-size estimator for the Change Analyzer stage (stage 1 of the AI pipeline). The estimator lives in `internal/aiengine/context_estimate.go` as a pure function — no Groq call, no DB write — and is exposed first via a new headless CLI subcommand `commitcraft ai context`. The reusable `aiengine.EstimateChangeAnalyzer` will back a TUI indicator in a follow-up; this iteration is CLI-only.

The estimator reproduces the exact payload `CallChangeAnalyzer` would send (`change_analyzer.prompt` + `"DEVELOPER_POINTS:\n…\nGIT_CHANGES:\n…"`), measures it in characters, derives an `est_tokens = ceil(chars/4)` heuristic, and compares it against the cached `context_window` of `Prompts.ChangeAnalyzerPromptModel` (read from `groq_models_cache` via `db.LoadModelsCache`). The diff is fetched through the same `git.GetStagedDiffSummary` cap (`Prompts.ChangeAnalyzerMaxDiffSize`) so any truncation that would hit the real pipeline is surfaced in the output too.

### Usage

```
commitcraft ai context
```

Prints a single JSON object on stdout:

```json
{
  "model": "meta-llama/llama-4-scout-17b-16e-instruct",
  "context_window": 131072,
  "system_prompt_chars": 4321,
  "user_input_chars": 78240,
  "total_chars": 82561,
  "est_tokens": 20641,
  "usage_pct": 15.74,
  "fits": true,
  "diff_truncated": false,
  "diff_max_bytes": 80000
}
```

Add `--strict` for agent-driven flows that want a deterministic gate:

```
commitcraft ai context --strict
```

Same JSON output, same exit 0 on success, but exits **3** when `fits: false` so the caller can branch on the exit code instead of parsing JSON. The strict flag is a no-op when `fits` is `null` (model not in the local cache) — we don't gate on unknown context windows.

When the configured model is absent from the local `groq_models_cache`, `context_window` is `0` and `usage_pct`/`fits` are `null` — the caller decides whether to refresh the cache or skip the gate. When there are no staged changes, the command emits `{"code": "no_staged_diff", "error": "…"}` on stderr and exits 1.

## v0.55.0 — 2026-05-22

Migrate every main-matcher shortcut dispatch in `internal/tui/update_*.go` from raw `msg.String()` checks to `key.Matches` against the active `KeyMap`. Touches the `ctrl+f` filter-mode cycler (workspace history + release history + release commit picker), the `esc`/`enter` handlers inside the focused filter bar, the `pgup`/`pgdown`/`ctrl+up`/`ctrl+down` panel-scroll dispatchers, the release-pipeline stage controls (`r`, `1`/`2`/`3`, `H`, `pgup`/`pgdown` — the original Unit 08 workaround), the commit-pipeline `H` (stage history), and the compose-tab per-focus handlers (commit-type cycle, scope clear, keypoints nav, pipeline-models nav). Adds two new `KeyMap` fields (`CycleFilterMode` for `ctrl+f`, `ClearField` for `x`/`backspace`/`delete`) and populates `mainListKeys()` / `releaseMainListKeys()` / `releaseKeys()` / `writingMessageKeys()` / `pipelineKeys()` accordingly so every binding the handlers reference has a real key.Binding behind it. Rewrites `keybindings_popup.go` so the four per-state popup builders read the key strings via `binding.Help().Key` from the active `KeyMap` instead of duplicating literals — a binding rename now propagates to the `?` popup automatically.

The remaining `msg.String()` hits in `internal/tui/` are intentional carve-outs per the new "Keyboard Dispatch" rule in `context/code-standards.md`: mention `@` extraction (`update_writing.go:43`), the `e`/`enter` scope-picker shortcut that isn't advertised in `?`, popup-close handlers in transient popups (`diffview_popup.go`, `mention_popup.go`, `keybindings_popup.go`), the scroll inside `history_dual_panel.go`, and the global guards in `update.go:1049-1115` (`ctrl+x` / `ctrl+l` / `ctrl+k` / `ctrl+1-3`).

### Usage

No user-visible behavior change. All shortcuts that worked in v0.54.1 continue to work in v0.55.0; the `?` popup now stays in sync with the keymap automatically and surfaces the actual bindings (so e.g. the stale `d / x` delete label is gone — only `d` is shown, matching the real `Delete` binding).

## v0.54.1 — 2026-05-22

Internal scaffolding for Unit 14's `msg.String()` → `key.Matches` migration. Populates `releaseKeys()` and `viewPortKeys()` with the release pipeline stage controls (`r`, `1`/`2`/`3`, `H`, `pgup`/`pgdown`) that have been matched via raw `msg.String()` since Unit 08 because the corresponding `KeyMap` fields were zero-valued in those variants. Adds a new `History` field to the `KeyMap` struct and wires it through `ShortHelp` / `FullHelp`. Writes the project's "use `key.Matches` with the active keymap as single source of truth" rule into `context/code-standards.md` so the historical footgun (`feedback_keymatch_zero_binding.md`) cannot be reintroduced silently. No user-visible change — dispatch in `update_release.go:374` still uses the `msg.String()` switch; Unit 14 will migrate it in one swap.

## v0.54.0 — 2026-05-20

Post-v0.53.0 polish on the release configuration popup plus a new in-TUI popup for the changelog feature. Ships as one tag.

**Late additions (2026-05-20 follow-up on the same `feat/release-config-polish` branch):**

- **Token field "g" mystery solved**: the GH_TOKEN row was rendering a lonely `g` (the first rune of the `"ghp_..."` placeholder) because bubbles' `textinput.Model` only paints `Width()+1` runes of the placeholder, and we never set a width. The mask itself worked correctly — what looked like an unmasked typed character was actually the static placeholder. Dropped the placeholder entirely; the hint underneath already explains what to put in the field.
- **Both config popups +20 %**: the release-config and changelog-config popups now use `width * 3/5` (floor `72`, ceiling `108`) and `height * 9/10` so the new auto_build / build_tool / build_target rows have breathing room.
- **In-popup list picker on Enter**: focusing the `Build tool` or `Build target` field and pressing Enter pops an inline list of choices (cached at popup creation: `make` for the tool, every non-phony Makefile target in source order for the target). Arrow keys navigate, Enter commits, Esc dismisses. The list is rendered exactly where the textinput row would be, so spatial context stays put.
- **Nerd-font icons on every configuration surface**: new symbol fields `ConfigureRelease`, `ConfigureChangelog`, `BuildTool`, `TokenIcon`, `BranchIcon` (with ASCII fallbacks). The command palette entries and both popup titles plus every field label now lead with the matching glyph.

**Unit 11 — release config popup polish.**

- Three new fields after Binary assets path: `Auto build` (space-toggled true/false), `Build tool`, `Build target`. The Build tool / target defaults are auto-detected by scanning the workspace `Makefile` for a `build_release` / `release` / `build` target (preference order).
- Explicit `EchoCharacter = '*'` on the GH_TOKEN field so the mask survives any future bubbles default change. Token entry stays write-only — saved values are never echoed back.
- Footer hint now renders through `theme.AppStyles().Help` (`ShortKey` / `ShortDesc` / `ShortSeparator`) instead of a flat muted line, matching every other on-screen help row. The hint only advertises `space toggle` when the Auto build field is focused.
- Dropped the `Ctrl+X` carve-out on the release config popup. Hard-quit now works from any field inside the popup, restoring the global muscle-memory shortcut. The version-decrement shortcut on the version popup (a different state) is unaffected.

**Unit 12 — configure changelog popup.**

- New `internal/tui/changelog_config_popup.go` mirroring the polished release popup pattern. Five fields: `Enabled` (space-toggled), `Path`, `Bump strategy` (validated against `patch`/`minor`/`major`), `Prompt file`, `Prompt model`.
- `DetectChangelog` auto-detects the CHANGELOG path (`CHANGELOG.md` / `CHANGELOG` / `HISTORY.md` / `RELEASE_NOTES.md`), the last `## vX.Y.Z` heading, and the file style (`keep-a-changelog` vs free-form headings).
- New command-palette entry "Configure changelog" opens the popup on demand. No auto-open — the changelog AI stage is opt-in, so it stays out of every other flow.
- `UpdateLocalConfigChangelog` writes the `[changelog]` block into `.commitcraft.toml` while keeping the runtime-only `Prompt` field out of disk.

### Usage

- **Release config popup**: open via command palette → "Configure release". Tab cycles fields. On the `Auto build` field press `space` to flip `true` / `false`; if you turn it on, `make $build_target` runs before each upload. `Ctrl+X` quits the TUI from anywhere. Save with `Enter` on the last field or `Ctrl+S` from any field.
- **Changelog config popup**: open via command palette → "Configure changelog". `Enabled = true` turns on the optional 4th AI stage that drafts a CHANGELOG entry styled to match your existing file. `Bump strategy` must be `patch`, `minor`, or `major`; anything else is rejected with an inline error and the disk write is skipped. `Prompt file` / `Prompt model` are optional overrides; leave both blank to inherit the built-in prompt and the active pipeline model.

## v0.49.0 — 2026-05-04

Add a third option to the `commitcraft -w <hash>` startup chooser: **"Rewrite using existing release"** lets the user reword the target commit with the message of an already-saved release entry from the workspace's SQLite DB, skipping the AI pipeline entirely. The chooser items now use dedicated nerd-font codicons (`cod-edit`, `cod-git_merge`, `cod-history`) with ASCII fallbacks for non-nerd-font terminals so each row carries a glyph that matches its action. A second popup is opened when the user picks the new option, listing one row per release as `<id> · <date> [<TYPE>] <branch> · <title>`; on selection the row is composed into `[TYPE] <branch>: <title>\n\n<body>`, copied into `RewordHash`/`FinalMessage`, and the TUI quits so `main.go`'s post-TUI hook calls `git.RewordCommit`.

### Usage

- `commitcraft -w <hash>` → "Rewrite using existing release" → pick one of the listed release entries → the original commit is rewritten with the formatted release message via amend (HEAD) or interactive rebase (historical).
- If the workspace has no release entries yet the chooser re-opens with a status hint so the user can pick a different option without restarting.
- Chooser glyphs: pencil for the regular reword, git-merge for "Rewrite as release/merge", history for "Rewrite using existing release". Themes without nerd fonts get `✎`, `Y`, `↺` respectively.

## v0.48.2 — 2026-05-04

Size the `commitcraft -w <hash>` startup chooser popup to fit every option. The previous height defaulted to `model.height/2` with a floor of `10`, which clipped the last entry once a third option ("Rewrite as release/merge") was added on small terminals. The popup now derives its minimum height from the item count (`len(items)*2 + 8` to cover spacing + title + borders + padding), still grows up to half the terminal when there's room, and clamps to `model.height-2` so it never overflows.

## v0.48.1 — 2026-05-04

Preserve merge topology when rewording historical merge commits. The previous `RewordCommit` path always used `git rebase -i <hash>^` with a `pick → reword` sed, but `rebase -i` skips merge commits in the TODO by default — the sed matched nothing and the rebase silently linearised the merge, dropping its second parent. Now `RewordCommit` detects merges via `git rev-list --parents` and switches to `git rebase -i --rebase-merges <hash>^` with a `merge -C <hash> → merge -c <hash>` sed, so the editor is invoked to inject the new message while both parents survive. Non-merge commits and HEAD continue to use the existing amend / pick-reword paths.

> Note — the v0.49.x / v0.50.x / v0.51.x entries below come from the `feat/release-flow-cleanup` branch where the release-flow cleanup landed iteratively. Those internal version bumps were never published as standalone tags; the cumulative work shipped to users in v0.53.0.

## v0.53.0 — 2026-05-19

Closed out the release-flow cleanup with two changes that together let a user configure and ship a GitHub release entirely from the TUI, without ever editing `.commitcraft.toml` by hand.

**Unit 07 (slim) — release-upload status feedback.** `UploadReleaseToGithub` now returns whether the release went out with zero asset files attached, and the status bar surfaces a `LevelInfo` line "Release uploaded to GitHub · no asset files attached" in that case. The original `ARG_MAX` crash and the empty-`bin/` walk were already fixed by `v0.51.2`; this commit just tells the user when the upload was deliberately notes-only.

**Unit 10 — release configuration onboarding.** A new in-TUI popup replaces the manual `release_config = { ... }` TOML block:

- `GH_TOKEN` moved out of `.commitcraft.toml` into `~/.config/CommitCraft/.env` (joining `GROQ_API_KEY`). On first start CommitCraft scans the global and local TOMLs for any legacy `GH_TOKEN = "..."` line, writes the value into `.env` (mode `0o600`), and strips it from the TOML so it can never be checked into a public repo by mistake.
- `internal/tui/release_config_detect.go` auto-detects sensible defaults: `owner/repo` from `git remote get-url origin`, current branch from `git symbolic-ref`, a patch-bumped version from `git describe --tags --abbrev=0`, and a binary-assets path picked from the first of `bin/`, `build/`, `dist/` that exists.
- A new multi-field popup (`release_config_popup.go`) renders the five fields with Tab / Shift+Tab focus cycling, `ctrl+a` / `ctrl+x` to bump version segments, Enter to save the final field. The token field uses `EchoPassword` and never echoes the saved value back. Save writes the TOML fields via `UpdateLocalConfigRelease` and the token via `SaveGhTokenToEnv`.
- "Create release in repository" and "Create release in Github" now pre-flight the upload: if `Repository` or `GH_TOKEN` is missing the popup auto-opens first; on save the upload chain resumes into the version editor.
- A new command-palette entry "Configure release" opens the popup on demand at any time.
- The legacy `stateSettingAPIKey` view was rebuilt to match the new popup style (left-aligned title, labeled input, italic hint, single rounded border) so the two configuration surfaces are visually consistent.

### Usage

- **First start after upgrade**: any `GH_TOKEN` line in your existing `.commitcraft.toml` is automatically migrated to `~/.config/CommitCraft/.env` and removed from the TOML. No action required.
- **Configure a release for the first time**: from any state, open the command palette and pick "Configure release". The popup pre-fills sensible defaults (auto-detected from your repo). Edit any field, then press Enter on the last field (or `Ctrl+S` anywhere) to save.
- **Upload a release**: from `stateReleaseMainMenu`, pick "Create release in repository" (or "Create release in Github" after the pipeline finishes). If the repo URL or token is missing, the configuration popup opens first; once saved, the upload resumes through the version editor and `gh release create`.
- **Notes-only release**: leave the "Binary assets path" field blank, or point it at an empty directory. The upload completes with notes only and the status bar reports "Release uploaded to GitHub · no asset files attached".

## v0.51.4 — 2026-05-13

Fixed the loading panel ("Loading releases / resolving commit subjects…") staying on screen after a successful GitHub release upload, with copy that described the history-sync flow instead of the upload that was actually running. Root cause: `update.go`'s `Create release in Github` path discarded the `tea.Cmd` returned by `createRelease`, so the release-history sync that clears `releaseLoading` never ran. Fix is three-part: (1) preserve the `loadCmd` so the sync runs; (2) add a separate `releaseUploading` flag that the build/upload pipeline toggles, cleared on success, error, and version-popup cancel; (3) `renderReleaseLoading` swaps the panel title/subtitle to "Uploading release to GitHub / building & pushing assets…" while `releaseUploading` is true.

## v0.51.3 — 2026-05-13

Fixed the release pipeline's final card occasionally rendering blank after a successful run. The cascade goroutine was mutating `releaseBodyOutput`, `releaseTitleOutput`, `releaseFinalOutput`, `releaseText`, and `commitLivePreview` directly from inside the `tea.Cmd` closure, which raced against the `View()` pass triggered when `applyPipelineResult` flipped the stages to `done`. The cascade now returns those strings via `IaReleaseBuilderResultMsg.Body / Title / Final`, and the `Update` handler writes them on the Bubble Tea main goroutine — so the final card paints its content in the same turn it becomes visible.

## v0.51.2 — 2026-05-07

Fixed crash in `UploadReleaseToGithub` when uploading a release from a repository with no binary assets. The root cause was a `filepath.Walk` over the entire working directory when `binary_assets_path` was empty, producing a command string that exceeded the OS `ARG_MAX` limit (`argument list too long`). The fix guards the walk behind a non-empty path check and an `os.Stat` existence check, so releases without assets upload cleanly. The command is now built via `exec.Command("gh", args...)` instead of `sh -c`, so the shell `ARG_MAX` limit can never be hit regardless of asset count.

### Usage

No configuration change needed. Repositories without `binary_assets_path` set (or where the configured directory does not exist) now create the GitHub release without attaching any files.

## v0.51.1 — 2026-05-04

Added granular release pipeline primitives and TUI support for partial retries, reducing redundant computation. The release pipeline view now supports per-stage controls for retrying and scrolling through stage output. Internally, the release pipeline logic was refactored into separate primitives for each stage.

## v0.51.0 — 2026-05-04

Added per-stage controls to the release pipeline view (`stateReleaseBuildingText`) so it has parity with the commit pipeline tab: `r` retries the whole pipeline, `1` / `2` / `3` retry from body / title / refine respectively (cascading downstream), `pgup` / `pgdn` scroll the focused stage's output viewport, and `H` opens the focused stage's history popup. Internally `aiengine/release.go` was split so each stage is callable on its own (`RunReleaseBody`, `RunReleaseTitle`, `RunReleaseRefine`); `RunRelease` now composes those primitives. The TUI dispatches partial cascades via `pipelineReleaseRetryStage(from)`, mirroring `pipelineRetryStage` for commits, and `IaReleaseBuilderResultMsg` carries the originating stage so the result handler only pushes history for stages that actually re-ran.

### Usage

- In the release pipeline view, after the initial run finishes you can:
  - Press `r` to re-run all 3 stages from scratch.
  - Press `1`, `2`, or `3` to re-run from a specific stage; downstream stages cascade. `3` only re-runs the refine stage and is the cheapest.
  - Press `pgup` / `pgdn` to scroll the focused stage's output (cycle the focus with `Tab` / `Shift+Tab`).
  - Press `H` to open the focused stage's history popup (older outputs from prior retries).

## v0.50.1 — 2026-05-04

Guard `Enter` in the release pipeline view (`stateReleaseBuildingText`) so it can't open the create-release menu while a stage is still running, was cancelled, or failed. Pressing `Enter` before `pipeline.allDone()` returns true now leaves the popup closed and surfaces a `LevelWarning` status-bar message ("pipeline still running · wait for stage 3 to finish before creating"). Without the guard, the user could reach the type=MERGE/RELEASE picker before the polished output existed and persist a release with whatever partial body the cards happened to hold. The `?` popup row for `↵` reflects the gate.

## v0.50.0 — 2026-05-04

Reworked focus inside the release pipeline view (`stateReleaseBuildingText`). `Tab` and `Shift+Tab` now cycle through the stage cards (body → title → refine → final, when populated) instead of bouncing the user back to the commit picker. "Back to picker" moved to `Esc`, which still cancels a running pipeline when one is in flight. The final-output card now lights up for both commit and release presets — content for release flows through the existing `releaseFinalOutput` field, with a "create release" hint instead of "accept & commit". Status bar and `?` popup advertise the new bindings.

### Usage

- `Tab` / `Shift+Tab` while looking at the release pipeline cycles between stage 1 (body), stage 2 (title), stage 3 (refine), and the final card (after stage 3 finishes).
- `Esc` walks back to the commit picker, preserving the prior selection set and cached pipeline output. While the pipeline is still running, `Esc` cancels it (no behavioural regression vs. the commit pipeline tab).
- `Enter` opens the create-release menu as before.

## v0.49.1 — 2026-05-04

Fixed `ctrl+e` "Selected only" mode in the release commit-picker showing an empty list even when commits were marked. Root cause: the sentinel value handed to the bubble list's `FilterInput` was `"\x00release-choose-selected-only\x00"`, but `textinput.Model.SetValue` silently strips control characters — so `releaseChooseListFilter` received the bare core string and the equality check against the sentinel never matched, dropping the path that returns selected items into a fuzzy match against arbitrary text and producing zero hits. The sentinel is now a plain-ASCII token that survives the round-trip. The toggle handler additionally resolves the underlying items index by hash before calling `SetItem` so the next time anyone leans on the bubble's filter the Selected flag stays glued to the right commit, and `applyReleaseChooseModeFilter` re-stamps `Selected` from `selectedCommitList` (the source of truth) before applying the filter as a defense-in-depth. A canary log warns if the visible set ever ends up empty despite live selections.

## v0.49.0-rfc — 2026-05-04 (feat-branch milestone)

Removed the cosmetic `release` ⇄ `merge` toggle from the release commit-picker. The `m` key, the `m:release` pill on the picker title bar, and the `· <mode>` suffix on the pipeline left-panel footer are gone. `ReleaseInput.Mode` was deleted from the AI engine — it only entered the debug log and never branched prompt content, so the pipeline output is unchanged. The persisted release classification (`storage.Release.Type` = `RELEASE` / `MERGE`, picked in the type popup *after* the pipeline) is unaffected. Numbered `v0.49.0` on the feat branch; the real published v0.49.0 (above) shipped from `main` on the same day with the unrelated reword-chooser change. This branch line is preserved here for historical traceability — the work itself reached users via v0.53.0.

## v0.48.0 — 2026-05-04

Wire up the "Rewrite as release/merge" branch of the `commitcraft -w <hash>` startup chooser so it actually rewords the original commit. Previously `setupReleaseReword` discarded the hash and dropped the user into the regular release flow, where finishing a release inserted a row in the SQLite `releases` table but never touched git history — clicking "Merge Commit" or "Release Commit" did nothing to the selected commit. The hash now travels through the flow on a new `releaseRewordHash` field and `createRelease` finalizes it as a reword: it composes `[TYPE] <branch>: <title>\n\n<body>`, copies the hash back into `RewordHash`, sets `FinalMessage`, and quits so `main.go`'s post-TUI hook calls `git.RewordCommit`. Cancelling the picker (Esc) clears the preserved hash so unrelated subsequent release creations don't silently reword.

### Usage

- `commitcraft -w <hash>` → pick "Rewrite as release/merge" → choose commits → run AI pipeline → press Enter → "Create item in CommitCraft" → "Merge Commit" / "Release Commit" → pick branch (for MERGE). The original commit is rewritten with the formatted release message via amend (HEAD) or interactive rebase (historical).
- "Create release in Github" while in this flow short-circuits through the same reword path; the GitHub upload chain is skipped because the user opted into reword, not publish.
- The status bar shows "Reword <short> as release/merge · pick commits to compose the message" while you're in the picker so you remember which mode you're in.
- Press Esc out of the picker to abandon the flow without touching the commit.

## v0.47.2 — 2026-05-04

Guard the reword path so it only runs when the user actually produced a message. Previously, launching `commitcraft -w <hash>` (the lazygit `R` binding) and then exiting the TUI without completing the AI pipeline still called `git.RewordCommit` with an empty `FinalMessage`, which either aborted the amend with "empty commit message" or wiped the original commit's message — both bad outcomes when triggered from a custom shortcut. Now reword fires only when both `RewordHash` is set and `FinalMessage` has non-whitespace content; cancelled flows print a clear "Reword cancelled — commit <short> left unchanged." notice to stderr and exit 0, leaving the commit intact and lazygit's status line clean.

### Usage

- `commitcraft -w <hash>`: complete the AI pipeline and press Enter on the compose view to reword. Cancelling at any earlier step (Esc, Ctrl+X, quit, missing scope/keypoints) is now safe — the commit is left untouched and you get a notice on stderr.
- No new keys; behavior change only.

## v0.47.1 — 2026-05-04

Isolate the persistent tab bar per app mode so the release flow no longer crosses into commit-mode handlers. `stateReleaseBuildingText` now maps to `TabPipeline` (its rendered view IS the pipeline) instead of `TabCompose`, and `defaultStateForTab(TabPipeline)` lands on `stateReleaseBuildingText` when `AppMode == ReleaseMode`. Before this fix, pressing Ctrl+3 after a release pipeline run dropped the user into the commit-mode `statePipeline`; pressing Enter there fired `createCommit` against the empty commit-mode fields (release output lives in `releaseBodyOutput/Title/Final`), producing an empty `stateOutput` report.

### Usage

- After running the release AI pipeline, Ctrl+2 / Ctrl+3 now shuttle between the release picker (`stateReleaseChoosingCommits`) and the release pipeline view (`stateReleaseBuildingText`) without leaving release mode.
- The tab bar correctly highlights "Pipeline" while the release pipeline view is open instead of falsely showing "Compose".
- No new keys; behavior change only.

## v0.47.0 — 2026-05-03

Surface builtin commit-type tags through the headless CLI so an external agent can discover and add them without opening the TUI. The discovery is split across two endpoints so each answers a single question, and the agent never has to filter response fields client-side. The append helper that the TUI tag picker uses was relocated to `internal/config` so the CLI no longer pulls in the TUI package.

### Usage

- `commitcraft ai list-tags` — returns only the tags `generate --tag` will accept (default + global + local, honoring `behavior=replace`). Each entry: `{ tag, description, source }` where `source ∈ {default, global, local}`.
- `commitcraft ai list-addable-tags` — returns the builtin tags the code knows about (from `commit.GetAddableCommitTypes`) that aren't yet in the local config. Output is independent of `commit_types.behavior`, so the list is consistent even when global/local use `replace`. Each entry: `{ tag, description }`.
- `commitcraft ai add-tag --tag TEST --tag UI` — appends the named builtin tags to the workspace `.commitcraft.toml` (creating it from the default template if needed). Repeatable; `-t` is a shorthand. Unknown tags exit `2` with `invalid_input`. Returns `{ added, skipped, config_path }`.

Workflow for an agent: pick a tag from `list-tags`; if nothing fits, check `list-addable-tags`, run `add-tag` on the chosen one, then call `generate`.

## v0.46.1 — 2026-05-03

The popup presents a four-color palette of optional tags defined in `.commitcraft.toml`. Users may select multiple tags with the `space` key, toggle all selections with `a`, confirm with `enter`, or cancel with `esc`. This feature allows users to add tags with a four-color palette in code but aren't selectable until they appear in `.commitcraft.toml`.

## v0.46.0 — 2026-05-03

Add a new "Add commit tag types" popup (`Ctrl+Y` from the History
list) that lets the user surface tags which already have a four-color
palette in code but aren't selectable until they appear in
`.commitcraft.toml` — for instance `UI`, `STYLE`, `TEST`, `PERF`,
`CHORE`, `CI`, `BUILD`, `REVERT`, `SEC`. The popup multi-selects with
`space`, supports `a` to toggle all, and on `enter` appends every
chosen tag — with its description and the four hex colors — to the
local config. If `.commitcraft.toml` doesn't exist yet, it's created
from the default template first so the append always succeeds. Tags
already present are silently skipped, and `model.finalCommitTypes` is
reloaded in-place so the new tags become selectable immediately.

### Usage

- From the History list, press `Ctrl+Y` to open the picker.
- Move with `↑↓`, toggle with `space`, toggle all with `a`.
- `Enter` saves; `Esc` cancels without writing.
- Already-configured tags are filtered out of the list, so re-opening
  the popup after a save shows only the still-addable extras.
- Help row in the keybindings popup (`?`) now lists `^y` under "App".

## v0.45.5 — 2026-05-03

Add a dedicated glyph for the `MERGE` commit type. Until now the tag
fell through `IconForCommitTag` to the generic bandage glyph, which
made merge rows in the Release inspect list visually
indistinguishable from `FIX`. The new entries map `MERGE` to
`nf-cod-git_merge` (``) — same Codicons family as the existing
`GitCommit` icon — and to `Y` as the no-nerd-fonts fallback.

### Usage

- No new keys. `MERGE` rows in the Release inspect commits list now
  render with the dedicated icon.

## v0.45.4 — 2026-05-03

Aesthetic refresh of the delete-confirm popup so it lines up with the
rest of the UI vocabulary (chips, themed help, warning palette). The
long sentence is replaced with a compact warning-tinted header
(`DELETE` chip + `<glyph> commit/release #ID` title), an italic-muted
quoted preview of the row's text (truncated to fit), a one-line "This
action cannot be undone." note, and the existing themed help footer.
The border now follows `theme.Warning` consistently. The glyph picks
`GitCommit` for commits and `Tag` for releases.

### Usage

- No new keys; `enter` confirms, `esc` cancels as before.

## v0.45.3 — 2026-05-03

Fix the History list snapping back to "completed" after deleting a
row while in draft mode. `UpdateCommitList` always queried
`status = "completed"`, silently kicking the user out of the draft
view. The function now takes a `status` parameter, and the delete
handler picks `"draft"` or `"completed"` based on `model.draftMode`
so the list stays in the same mode after a successful delete.

### Usage

- No user-facing change. The draft / completed toggle now survives
  deletions.

## v0.45.2 — 2026-05-03

Fix the autodraft hook firing when the user exits from Release mode.
The release states (`stateReleaseChoosingCommits`,
`stateReleaseBuildingText`) map to `TabCompose` for tab-bar purposes,
so on quit `autodraftIfNeeded` would `SaveDraft` a half-empty row into
the `commits` table — surfaced as a misleading
`Exit in Compose — draft saved` notice. Autodraft is a normal-commit
feature only; release mode persists through its own flow.

### Usage

- No user-facing change. Quitting from Release mode no longer
  produces a stray draft row in `commits`.

## v0.45.1 — 2026-05-03

Move the History list's commit counter into the filter bar. The
bubbles list's bottom statusbar (which rendered `X commits`) is
hidden, and the count now lives at the right of the filter bar
prefixed with the `GitCommit` glyph: `[TITLE] > …    5/8 commits`.
Singular/plural noun is picked from the total.

### Usage

- No new keys. The number to the right of the filter bar reflects
  visible / total commits in the current workspace.

## v0.45.0 — 2026-05-03

Track each commit row's origin so the History list can tell apart
records born inside the TUI (a human driving the interface, even when
the AI pipeline is involved) from records created by the headless
`commitcraft ai` subcommand (Claude or another agent). Adds a `source`
column to `commits` (default `'tui'`), sets `'ai'` on `ai generate`,
and renders a small pill — `TUI` (slate) or `AI` (purple, matching
the existing AI palette) — between the message and the date in the
main History row.

### Usage

- No new keys or flags. Existing rows show `TUI` (the column default
  for legacy data); new rows from `commitcraft ai generate` show `AI`.
- The pill sits between `scope: title` and the date column in the main
  History list.

## v0.44.0 — 2026-05-02

Added `commitcraft ai edit` so headless agents can patch a draft's
generated text in place instead of re-running pipeline stages every
time they spot a wording issue. The subcommand only writes to the
draft row; it does not call Groq, does not touch `ai_calls`, and
recomposes `final_message` from the new title/body, preserving the
existing changelog mention line when present.

### Usage

`commitcraft ai edit --id <draft-id> [flags]`

- `--title <s>` — replace `IaTitle`. Use `-` to read from stdin.
- `--body <s>` — replace `IaCommitRaw`. Use `-` to read from stdin.
- `--changelog <s>` — replace the changelog entry. Use `-` for
  stdin, or the literal `CLEAR` to drop the entry.
- `--tag <t>` — override the commit type tag (must be a known tag).
- `--scope <s>` — override the commit scope.

At least one editable flag is required; passing only `--id` is an
error. Multiple `-` flags share a single stdin read. The command
prints the same JSON shape as `ai show`/`ai regenerate` so it slots
into existing agent loops.

## v0.43.4 — 2026-05-01
- Fixed branch listing in release-mode scope picker to prevent decorations and stray prefixes, ensuring clean branch names are displayed.

## v0.43.3 — 2026-05-01

- Fixed branch listing in release-mode scope picker so worktree-checked-out branches no longer appear with a leading `+ ` (and detached-HEAD / ANSI-colored output no longer leak into names). `GetGitBranches()` now uses `git for-each-ref refs/heads/` with `%(refname:short)` instead of parsing `git branch --list`.

## v0.43.2 — 2026-05-01

- Fixed the "Exit in Compose — draft saved" notice to no longer appear after a successful flow completion by skipping the autodraft step when a final message has already been produced.

## v0.43.1 — 2026-05-01

Fix the "Exit in Compose — draft saved" notice firing after a successful flow.

- The autodraft-on-quit hook now checks `model.FinalMessage` first. When
  the user exits because they completed the flow (Enter on `stateOutput`,
  the confirm screen, the release builder, the reword view, etc.) no
  draft is written and no exit notice is printed. The notice still
  appears for genuine mid-flow quits from Compose/Pipeline.

## v0.43.0 — 2026-04-30

Multi-stage AI pipeline for Release Mode, mirroring the commit pipeline.

- **3 release stages.** The single-call release builder is now a 3-stage
  pipeline: `release_body` (assembles the body from selected commits) →
  `release_title` (composes the title from body + commits) →
  `release_refine` (polishes tone, hierarchy, and formatting). Each
  stage is a separate Groq call with its own customizable prompt and
  model.
- **Pipeline tab cards.** When `stateReleaseBuildingText` is active the
  Pipeline tab UI renders the run with the same per-stage cards used
  by the commit pipeline: spinner, progress underline, telemetry
  (tokens / latency / TPM bar), and per-stage history (`H` to compare
  prior generations). The left panel shows the selected commits
  instead of the diff file tree.
- **Telemetry persisted.** A new `release_ai_calls` SQLite table mirrors
  `ai_calls` and stores per-stage tokens/latency/request-id so the
  release history dual panel surfaces the same telemetry strip that
  the commit history already shows.
- **Release/Merge label.** A discrete pill on the workspace-commits
  panel border (`m` to toggle) tags the run as either `release` or
  `merge`. Cosmetic only — both modes use the same prompt set; the
  value flows through to logs.

### Usage

- The release builder is still triggered with `Enter` from
  `stateReleaseChoosingCommits`. It transitions to
  `stateReleaseBuildingText` with the Pipeline cards animated.
- Press `m` while picking commits to toggle the release/merge pill.
- Customize prompts per stage via `~/.config/CommitCraft/config.toml`:
  - `release_body_prompt_file` / `release_body_prompt_model`
  - `release_title_prompt_file` / `release_title_prompt_model`
  - `release_refine_prompt_file` / `release_refine_prompt_model`
      Defaults live in `~/.config/CommitCraft/prompts/release_*.prompt` and
      are written from the embedded templates on first run.
- Inspect the new telemetry table with
  `sqlite3 ~/.config/CommitCraft/commits.db "select * from release_ai_calls"`.
- The single legacy `release_prompt_*` config keys are replaced; remove
  them from your TOML if present.

## v0.42.3 — 2026-04-30

Three follow-up fixes for the release commit picker:

- **List frozen after build → back transition.** Tab/Shift+Tab in
  `stateReleaseBuildingText` was running `switchFocusElement`, which
  toggles the legacy `focusListElement` / `focusViewportElement` pair.
  When that landed back on `stateReleaseChoosingCommits`, the picker's
  focus dispatch (which only knows about `focusReleaseChoose*`) had no
  case to match, so the commit list looked permanently dead. Tab/Shift
  now reset to `focusReleaseChooseCommitList`, and the picker also
  defensively coerces any stray focus on entry through a new
  `isReleaseChooseFocus` guard.
- **Selected-only swap was a no-op.** `list.filterItems` short-circuits
  to "show everything" when `FilterInput.Value()` is empty, bypassing
  our custom `Filter` entirely — so toggling Selected-only with no
  query never hid the unselected rows. We now feed the list a
  non-empty sentinel string when the query is empty + Selected-only is
  on; `releaseChooseListFilter` recognises the sentinel and treats it
  as an empty term, then drops the unselected items as expected.
- **Rewrite popup label.** `rewordChooseAsRelease` reads
  `Rewrite as release/merge` (was `Open release mode`) so the choice
  matches the action it triggers.

## v0.42.2 — 2026-04-30

Aligned the release commit picker's filter wiring with the
commit-mode workspace history pattern:

- `ReleaseFilterBar` now carries a `kind` (history vs picker). The
  picker constructor `NewReleasePickerFilterBar` flips a dedicated
  `currentReleasePickerFilterMode` package var so cycling `ctrl+f`
  inside the picker no longer leaks into the release main menu (and
  vice-versa).
- The picker's mode set switched from the release-history axes
  (TITLE/TYPE/VERSION/BRANCH) to picker-relevant axes
  **TITLE / HASH / TYPE / TAG**.
- `WorkspaceCommitItem.FilterValue` switches on the active picker
  mode, mirroring `HistoryCommitItem.FilterValue`. New helper
  `extractCommitTypeTag` parses the leading `[XXX]` token off the
  subject for the TYPE mode.
- Cycling the filter mode now triggers an immediate refilter so the
  visible set updates without waiting for a keystroke.

### Usage

- `Ctrl+F` while the picker is on the Compose tab cycles through
  TITLE → HASH → TYPE → TAG. The pill colour and label change to
  reflect the active axis. Type into the bar and the list filters
  against the corresponding field.

## v0.42.1 — 2026-04-30

Real fixes for the regressions reported on top of v0.42.0:

- The workspace commit picker was driven into `list.Filtering` state,
  which routes every key event into the bubble's internal filter
  textinput. Symptoms: cursor reset on add, list stops responding
  after a focus cycle, a second "Filter:" header pops on top of the
  custom bar when `Ctrl+E` toggles the mode. Now we use
  `list.FilterApplied` (filter active, navigation enabled) and
  `SetShowFilter(false)` so the second header never renders.
- `Ctrl+A` no longer re-runs `applyReleaseChooseModeFilter`. SetItem
  alone returns the filter-refresh command, so the cursor stays put
  when toggling a commit; in **Selected only** mode the filter
  pipeline still picks up the change automatically.
- Files-changed picker default mode now shows just the filename. The
  dim parent directory only appears in the `Ctrl+E`-swapped full-path
  render.
- `?` opens a dedicated keybindings popup for
  `stateReleaseChoosingCommits` (`releaseChooseCommitsKeybindings`)
  with the picker-specific shortcuts. The popup is suppressed while
  the filter bar is focused so `?` can be typed into the query.

## v0.42.0 — 2026-04-30

Follow-up bug fixes for the workspace commit picker
(`stateReleaseChoosingCommits`):

- **Filter input** now actually accepts text. The list bubble's
  built-in `/` keybinding was intercepting the keystroke before our
  custom filter bar saw it; both `KeyMap.Filter` and `KeyMap.ClearFilter`
  are now disabled on the workspace list, so `/` reaches the bar
  consistently.
- **Selected only** mode no longer pipes commit hashes through the
  `SetFilterText` channel (which collided with the user's typed
  filter). A custom `list.Filter` plus a mode-aware `FilterValue` now
  hide unselected items while still letting the typed filter narrow
  the visible subset.
- **Duplicate row count** removed: the list's built-in status bar +
  pagination strip (`5 commits` / `1/N`) duplicated the counter
  rendered by the custom filter bar; both are now hidden.
- **`Ctrl+E` is context-aware**: on the files list it toggles the
  picker between filename+dim-dir and full relative-path rendering;
  anywhere else it still flips **All commits ⇄ Selected only**.

### Usage

- `/` focuses the filter input; type to narrow the workspace list.
- `Ctrl+E` while focused on the files list flips between filename and
  full-path display. `Ctrl+E` from any other zone swaps the
  All/Selected indicator on the top border.

## v0.41.0 — 2026-04-30

Release Mode UX fixes for the workspace commit picker
(`stateReleaseChoosingCommits`):

- The persistent tab strip now tracks Release Mode. Release-compose
  states map to the `Compose` tab; `stateReleaseMainMenu` stays on
  `History`. `Ctrl+2` in Release Mode opens the release commit picker
  instead of the commit-mode compose screen.
- The `-w` startup popup's `Open Release Mode` entry lands directly on
  the commit picker (Compose tab) instead of the release main menu.
- The `All commits / Selected only` indicator moved out of the panel
  body and onto the top border (inline with the title), giving the
  workspace commit list ~9 visible rows instead of ~1.
- `Ctrl+E` now toggles the indicator from any focus — the binding was
  advertised in the bar but unbound. The mode-bar focus stop was
  removed from the Tab cycle, so navigating focus never lands on a
  zone that ignores up/down.

### Usage

- In Release Mode press `Ctrl+1` for History, `Ctrl+2` for the commit
  picker, `Ctrl+3` for Pipeline.
- In the commit picker: `Ctrl+E` swaps **All commits ⇄ Selected only**;
  `Ctrl+F` cycles the filter mode; `Ctrl+A` adds the cursor's commit
  to the selection. Tab/Shift+Tab cycle Filter → CommitList → MsgVp →
  Files → Diff.

## v0.40.1 — 2026-04-30

Headless `ai promote` now writes and stages `CHANGELOG.md` when the
draft carries a stage-4 entry and `[changelog].enabled = true`,
mirroring the TUI's write-on-accept timing. Previously only the TUI
ran `changelog.Prepend` + `git add`, so headless callers got a
`final_message` advertising a CHANGELOG update that never happened
on disk. The path is re-detected from `c.Workspace` + config at
promote time (no schema change). I/O failures surface as typed JSON
errors (`changelog_target_missing`, `changelog_write_error`,
`changelog_stage_error`) after the row is already `completed`, so
re-running `ai promote --id N` is idempotent and retries the
write+stage. Closes #2.

### Usage

`commitcraft ai promote --id <N>` writes+stages the changelog by
default when applicable. Pass `--no-changelog-write` to keep the
old behavior (text emitted in `final_message`, file untouched) — useful
when the caller wants to manage the CHANGELOG itself.

## v0.40.0 — 2026-04-30

Headless `ai generate` / `ai regenerate` / `ai show` / `ai promote`
now emit `final_message` with the same `[TAG] scope: …` header the
TUI's post-commit view shows, so the field can be piped straight
into `git commit -F -` without caller-side reassembly. The header
respects `commit_format.type_format` from the user's TOML config
(default `[%s]`); when `Type` or `Scope` is empty, `final_message`
falls back to the bare title+body so partial drafts never produce
malformed headers like `[ADD] : foo`. TUI and headless now share a
single `commit.FormatFinalMessage` helper — closes #1.

`FormatFinalMessage` treats missing `tag`, `scope` or `message` as a
programmer error (`commit.ErrIncompleteCommit`); headless callers
surface it as a structured `incomplete_commit` JSON error and exit
1, while the TUI logs the error and degrades to the raw `MessageEN`
so the user is never left with a blank screen.

### Usage

- No flag changes. Run `commitcraft ai generate -k "…" -t ADD -s wt`
  and `jq -r '.final_message'` will now start with `[ADD] wt: …`.
- To override the wrapper, set under `[commit_format]` in
  `~/.config/CommitCraft/config.toml` or `.commitcraft.toml`:
  `type_format = "<%s>"` → `final_message` becomes `<ADD> wt: …`.
- `body` and `title` fields are unchanged, so consumers that wanted
  the unprefixed text still have it.
- If a draft is missing `type` or `scope`, `ai show/promote/...`
  now exit 1 with `{"code":"incomplete_commit", ...}` on stderr
  instead of silently emitting a half-formed `final_message`.

## v0.36.2 — 2026-04-30

New `--refresh-diff` flag on `ai regenerate` that re-reads
`git diff --cached` from the commit's stored workspace and persists
the new snapshot before the pipeline runs. Without the flag the
stored snapshot is reused (existing behavior — keeps iteration cheap
when the message is bad but the code didn't change). Added a
path-aware `git.GetStagedDiffSummaryAt(workspace, maxDiffChars)` so
the refresh works from any directory.

### Usage

`commitcraft ai regenerate --id <N> --refresh-diff` re-reads the
staged diff from the workspace recorded on the commit row and
overwrites `diff_code` before running the pipeline. Requires a full
regenerate — combining with `--stage <body|title|changelog>` returns
`invalid_input` because only stage 1 (the change analyzer) consumes
the diff.

## v0.36.1 — 2026-04-30

`ai regenerate --id N` now uses the commit's stored `Workspace` as the
pipeline working directory instead of the caller's cwd. The changelog
refiner was previously detecting whichever `CHANGELOG.md` happened to
sit next to the shell when `regenerate` was invoked, which polluted
cross-workspace iterations (a draft created in repo A but regenerated
from repo B's cwd would acquire B's changelog mention). With this fix
the refiner always targets the repo that owns the commit.

### Usage

No flag changes. `regenerate --id <N>` is now safe to invoke from any
directory; the engine reads the commit's stored workspace internally.
Legacy rows without a workspace fall back to cwd.

## v0.36.0 — 2026-04-29

Two new headless surfaces driven by the upcoming Claude Code skill: a
`commitcraft ai list-tags` subcommand that emits the resolved tag set
as JSON (with `source` attribution per tag), and a `--stage` flag on
`ai regenerate` that re-runs only one stage of an existing draft. The
per-stage cascade mirrors the TUI's retry shortcuts, so an agent can
fix a broken title without re-spending tokens on the change analyzer.

### Usage

- `commitcraft ai list-tags` — prints `[{"tag","description","source"}]`
  on stdout. `source` is `default` for built-in tags, `global` for
  entries from `~/.config/CommitCraft/config.toml`, and `local` for
  entries from the workspace's `.commitcraft.toml`. Useful for an
  external agent picking the tag that best matches the staged diff.
- `commitcraft ai regenerate --id <N> --stage <body|title|changelog>` —
  re-runs only the named stage, reusing the upstream outputs stored on
  the draft. Cascade: `body` → body+title+changelog; `title` →
  title+changelog; `changelog` → only the refiner. Existing
  `ai_calls` rows for stages that don't re-run are preserved.

## v0.35.1 — 2026-04-29

New `commitcraft ai promote --id <N>` subcommand that flips a draft's
status to `completed` via `storage.FinalizeCommit`. Does not execute
`git commit` — the caller takes the printed `final_message` and
commits it themselves.

### Usage

`commitcraft ai promote --id <N>` validates that the draft exists and
has a non-empty `final_message`, then updates its status to
`completed` and prints the refreshed JSON. Errors: `not_found`
(unknown id), `invalid_input` (missing id or empty draft), `db_error`.

## v0.35.0 — 2026-04-29

Headless `commitcraft ai …` subcommand suite so an external agent can
drive the AI commit-message pipeline without a TUI. The same three
stages (change analyzer → commit body → commit title, plus the
optional changelog refiner) now live in `internal/aiengine` and are
shared by the TUI and the headless CLI; the TUI shim in
`internal/tui/ai_pipeline.go` delegates to the engine and copies stage
telemetry back onto the pipeline cards. New helper
`storage.GetCommitByID` for fetching a single draft.

### Usage

The headless flow is rooted at `commitcraft ai`:

- `commitcraft ai generate -k "<keypoint>" [-k …] -t <TAG> -s <scope> [-s …] [--no-changelog]` —
  reads `git diff --cached`, runs the pipeline, persists a `draft` row
  in the local SQLite DB, and prints the resulting JSON to stdout
  (`id`, `final_message`, `summary`, `body`, `title`, `stages` …). At
  least one keypoint and one scope are required; the tag is validated
  against the resolved type list.
- `commitcraft ai regenerate --id <N> [-k …] [-t …] [-s …] [--no-changelog]` —
  re-runs the pipeline against the diff snapshot stored on the draft
  (so the staged set can change between runs without affecting the
  iteration). Any of `-k/-t/-s` overrides the stored value before the
  pipeline fires; the row is updated in place and the new JSON is
  printed.
- `commitcraft ai show --id <N>` — prints the JSON for an existing
  draft/commit, including per-stage telemetry rebuilt from `ai_calls`.
- `commitcraft ai list [--status draft|completed]` — prints a JSON
  array of rows in the current workspace with id, status, type, scope,
  title snippet and creation timestamp. Defaults to `draft`.

Errors are emitted as JSON on stderr (`{"error":"…","code":"…"}`) with
codes `invalid_input`, `no_staged_diff`, `not_found`, `bootstrap_error`,
`db_error`, `api_error`. The headless command never executes
`git commit` — promoting a draft is intentionally out of scope.

## v0.34.2 — 2026-04-29

Clearer labels and dedicated icons for the reword menus, so the
user-facing strings reflect the actual semantics (which message goes
into git, and whether a new CLI DB row is produced).

- Post-commit menu (`update_commit.go:318`):
  - `"Reword commit"` → `"Reword with this message"` (replay icon
      `nf-md-replay`, U+F0459).
  - `"Commit and reword"` → `"Reword with new AI run"` (database-plus
      icon `nf-md-database_plus`, U+F01BA, signaling that a new row is
      created in the CLI DB).
- `-w <hash>` popup (`model.go:575`):
  - `"Reword as commit"` → `"Reword this commit"`.
  - `"Reword as release"` → `"Open release mode"` (it discards the
      hash and switches to release mode; the old label suggested it was
      a reword variant).
- New `Theme.Symbols.ReuseMessage` and `Theme.Symbols.NewDbRecord`
  fields with nerd-font glyphs and ASCII fallbacks (`↻`, `db+`).
- Status-bar messages on commit-pick screens reworded to mirror the
  new menu labels.

### Usage

The reword entry points are unchanged, only their labels and icons.
From the post-commit menu: pick `Reword with this message` to apply
the saved AI output to a git hash without re-running the pipeline, or
`Reword with new AI run` to produce a fresh CLI DB row from the
commit's diff and then reword. From the `-w` popup: `Reword this
commit` enters the standard reword flow; `Open release mode` is the
escape hatch into release mode.

## v0.34.1 — 2026-04-29

Reword flow polish: both reword paths now land on the Compose panel and
the Pipeline view shows the loaded commit's per-file diffs.

- `setupCommitReword` (CLI `-w <hash>` → "Reword as commit") and the
  "Commit and reword" menu branch in `updateRewordSelectCommit` now
  transition to `stateWritingMessage` with `focusComposeSummary` and
  `writingMessageKeys()`, instead of dropping the user into the prefix
  selector.
- Both paths call `loadPipelineFilesFromDb(model, diff)` after fetching
  `git.GetCommitDiffSummary`, so `pipelineDiffList` and `dbFileDiffs`
  are populated and the Pipeline tab renders the file list and per-file
  diff for the targeted commit.
- Renamed the `useDbCommmit` field (note the triple `m`) to
  `usePreloadedDiff` across the codebase. The flag never meant "data
  came from the SQLite DB" — its real semantics are "use the diff
  already loaded in `model.diffCode` instead of running `git diff
--staged`", which covers both DB-loaded drafts and git-hash-loaded
  reword targets.

### Usage

Run `commit_craft -w <hash>` (or pick "Commit and reword" from the
main menu and choose a commit) and you now arrive directly on the
Compose panel with the commit's diff visible in the Pipeline tab.
Write key points and trigger the AI pipeline with `Ctrl+W` as usual.

## v0.34.0 — 2026-04-29

Two new persistent pills on the right side of the main status bar that
mirror the existing CHANGELOG / scope-stale pattern:

- **Reword mode** — purple pill (`#3f2d5c` / `#dccaf0`) with the
  `nf-cod-git_pull_request_create` glyph (U+EBBC). Visible whenever the
  TUI is targeting an existing commit hash (`-w` flag on launch or a
  reword pick from the popup). Disappears as soon as the reword flow
  finishes or the user switches to release mode.
- **Local config detected** — slate-teal pill (`#1f3a44` / `#bcd9e3`)
  with the `nf-seti-config` glyph (U+E615). Visible whenever a
  `.commitcraft.toml` file exists in the working directory and is
  overriding the global config. Detected once at startup.

Both icons are sourced from the active theme (`Symbols.Reword`,
`Symbols.LocalConfig`), so the nerd-font / no-nerd-font branch is the
same one the rest of the UI uses. ASCII fallbacks are `rw` and `cfg`.

### Usage

No new keys. Launch with `-w <hash>` (or pick "Reword" from the popup)
to see the reword pill. Drop a `.commitcraft.toml` in the workspace
root before launching to see the local-config pill.

## v0.33.0 — 2026-04-29

Autodraft on quit. Quitting the TUI from the COMPOSE or PIPELINE tab now
persists the in-memory buffer as a draft before exiting, so the next
launch can resume the work-in-progress instead of losing it.

- New helper `autodraftIfNeeded` runs on every graceful quit path
  (`Ctrl+X` global, popup `q`, error-path exits, reword/output flows).
  Filters by `tabForState`: only fires when the user is in COMPOSE or
  PIPELINE.
- Idempotent against manual `Ctrl+D` saves: `SaveDraft` upserts by ID,
  and the UPDATE branch leaves `status` untouched, so finalized commits
  are never silently downgraded to drafts.
- Skips empty buffers — no junk drafts when the user quits without
  having typed anything.
- DB errors are logged and swallowed so a SQLite failure can never
  block the user from exiting the TUI.
- Popup models that previously returned `tea.Quit` now emit a
  `programQuitMsg` intercepted at the top of `Update`, routing through
  the same autodraft hook.
- Limitations: only graceful quits are intercepted. `SIGKILL`, terminal
  close, or `kill -9` still drop the in-memory state.
- After the TUI tears down, a charm-log INFO line is printed to stderr
  with a git glyph prefix and the saved draft id, e.g.
  `INFO  3:14PM  Exit in Compose — draft saved draft_id=42`.

### Usage

No new keys. Just press `Ctrl+X` (or any other quit path) while in
COMPOSE / PIPELINE — on next launch the draft will be listed in the
History tab with the buffers you had open. Look at stderr after the
process exits for the confirmation line.

## v0.32.2 — 2026-04-29

- Fixed formatter output corruption caused by concurrent `gum` invocations in the pre-commit hook.
- Improved the reliability of the formatting process by capturing and processing `goimports-reviser` output in a sequential manner.

## v0.32.1 — 2026-04-29

Refresh the Tag palette popup layout so it reads as a real reference
card instead of a dense single-line table.

- Two-line column header (`TYPE`, `BG / BLOCK`, `FG / BLOCK`,
  `BG / MSG`, `FG / MSG`) under a horizontal separator.
- Each tag now renders on two stacked rows: a small filled swatch
  painted with the actual color sits above the `#RRGGBB` label, so
  the value reads as both swatch and number at once.
- The preview moves to its own column with extra horizontal breathing
  room and renders chip + dummy commit subject side-by-side, matching
  how the row looks in the History list.
- Drops the per-row description line.

## v0.32.0 — 2026-04-29

Add a Ctrl+K command palette and ship two starter commands.

- New `command_palette_popup.go` renders a filterable list popup with
  the same style vocabulary as the type picker. `Ctrl+K` opens it from
  any state when no other popup is on screen; `esc` (or another `^k`)
  closes it. Driven by a small `paletteCommand` registry — adding a
  command means appending one entry in `builtinCommands` and routing
  its ID in `update.go`.
- New `tag_palette_popup.go` renders every effective commit-type tag
  in a table: chip · bg(block) · fg(block) · bg(msg) · fg(msg) · live
  preview row using the actual chip/message styles, with a per-tag
  dummy commit subject so the palette reads like a real history line.
  Scrollable via the embedded viewport.
- Seed commands:
  - **Generate local config file** — runs `CreateLocalConfigTomlTmpl`,
      no-op + warning toast if `.commitcraft.toml` already exists.
  - **Show tag palette** — opens the table above against
      `model.finalCommitTypes` (built-ins + user customs).
- `keybindings_popup.go` now lists `^k command palette` under the App
  group on every state that surfaces the popup.

### Usage

Press `Ctrl+K` from anywhere (compose, history, release, pipeline) to
open the palette. Type to filter, `↑↓` to navigate, `enter` to run,
`esc` to dismiss. From the palette, pick **Show tag palette** to inspect
how each tag will render with its current colors — useful after editing
`bg_block`/`fg_block`/`bg_msg`/`fg_msg` in the TOML config.

## v0.31.1 — 2026-04-29

Make the autogenerated config self-document the per-tag color overrides
introduced in v0.31.0.

- `ensureGlobalConfigExists` writes an expanded header explaining the
  four `bg_block`/`fg_block`/`bg_msg`/`fg_msg` keys, the fallback chain,
  and a worked example.
- `CustomCommitType` no longer marks the four color tags `omitempty`,
  so every `[[commit_types.types]]` entry serializes with the keys
  visible.
- `commit.GetDefaultCommitTypes` and `GetDefaultLocalCommitExamplesTypes`
  now seed the canonical hex values from `commitTypePalette`. The
  generated `~/.config/CommitCraft/config.toml` and `.commitcraft.toml`
  ship as a populated, editable template instead of empty rows.
- Side-effect: built-in tags (FIX, ADD, …) are now technically
  overridable from TOML — the seeded values match the built-in palette,
  so editing them is the supported path to change them.

## v0.31.0 — 2026-04-29

Per-tag commit-type colors are now user-configurable. The four-color
palette (`bg_block` / `fg_block` for the chip, `bg_msg` / `fg_msg` for
the message row) can be set per entry in `[[commit_types.types]]` and
the renderer honors them across the type popup, MasterList and compose
pills. Built-in tags keep their hardcoded palette unless redeclared.

- `internal/commit/types.go`: `CommitType` now carries `BgBlock`,
  `FgBlock`, `BgMsg`, `FgMsg` instead of the old single `Color` string.
- `internal/config/types.go`: `CustomCommitType` exposes the four hex
  fields as TOML keys; new `CommitTypePalette` mirror struct + the
  `CommitFormat.CommitTypePalettes` map carries them through.
- `internal/tui/styles/commit_type_palette.go`: new
  `RegisterCustomCommitTypePalettes` overlay applied per-field on top
  of the alias-resolved built-in palette and theme fallback. Invalid
  hex values (anything not starting with `#`) are warned to stderr and
  ignored — the slot keeps its built-in color.
- Dead `commitTypeColor` plumbing on `Model`, `setCommitTypeMsg`,
  `CommitTypeItem.Color()`, and `lookupTypeColor` removed.

**Breaking:** the legacy `color = "..."` key on `[[commit_types.types]]`
is no longer parsed (the TOML decoder silently drops it). It never
affected rendering, so visual output is unchanged for users who only
set `color`. To color a custom tag now, use the four new keys.

### Usage

Add a custom tag with its own palette in `~/.config/CommitCraft/config.toml`
or in the project-local `.commitcraft.toml`:

```toml
[[commit_types.types]]
tag         = "EXP"
description = "Experimental work"
bg_block    = "#264653"
fg_block    = "#e9f5db"
bg_msg      = "#1b2f37"
fg_msg      = "#a8c5b3"
```

All four color keys are optional. Omitting one keeps that slot at the
built-in palette color (or theme neutral, if the tag has no built-in
entry). Hex values must start with `#` (e.g. `#RRGGBB`); other formats
are ignored with a warning on startup.

## v0.30.5 — 2026-04-29

Tighter status bar messaging on AI / build failures: the bar shows a
generic hint with the stage number; the full error goes to the log.

- New `extractPipelineStage` helper parses the `stage N (…):` prefix
  added in v0.30.4 and yields just `stage N`.
- `IaCommitBuilderResultMsg` now renders `AI pipeline failed on stage N
· check logs · ^w to retry` (drops to `AI pipeline failed · …` when
  no stage can be extracted).
- `IaResleaseBuilderResultMsg` and `releaseBuildResultMsg` show
  `AI release failed · check logs · …` and `Build failed · check logs`
  respectively. Detailed errors stay in the log via `model.log.Warn`.

## v0.30.4 — 2026-04-29

Fix: an empty Groq response (200 OK with no `choices`) was promoted to a
fatal `model.err`, which `view.go` renders as a full-screen error and
makes the TUI feel like it crashed.

- `internal/api/groq.go` distinguishes the two empty-response shapes
  ("no choices" vs "empty content") and includes the model name in the
  error so the status bar message is actionable.
- `internal/tui/ai_pipeline.go` wraps each pipeline error with
  `stage 1 (change analyzer):` / `stage 2 (commit body):` /
  `stage 3 (commit title):` so the failing stage is obvious without
  parsing the inner string.
- `internal/tui/update.go` no longer assigns `model.err` from
  `IaCommitBuilderResultMsg`, `IaResleaseBuilderResultMsg`, or
  `releaseBuildResultMsg`. The cause is logged + shown in the status
  bar and the user can re-run the pipeline / build without restarting.

### Usage

When a stage fails (rate-limit, empty Groq response, network blip), the
TUI now stays alive: the status bar turns red with `AI pipeline failed
— stage 2 (commit body): groq returned 200 OK but no choices …`, and
you can press the usual retry shortcut (e.g. `^w` to re-run the full
pipeline, `2`/`3` to re-run from a specific stage on the pipeline view)
without losing your draft.

## v0.30.3 — 2026-04-29

- Updated the pre-commit hook to pipe the output of `goimports-reviser` through a custom handler, enabling styled log messages and respect for the `GUM_LOG_LEVEL` setting.

## v0.30.2 — 2026-04-29

Fix: page-key scrolling on the inspect panel was being reset to top on
every render.

- `HistoryDualPanel.SetSize` and `ReleaseDualPanel.SetSize` are called
  from each `View()` pass; they unconditionally invoked `refreshContent`
  which calls `GotoTop()` on both viewports, wiping the user's scroll
  position immediately after `pgup`/`pgdown` updated it. Both now early
  return when the dimensions haven't changed, so resize-time refresh
  still happens but render-time calls are inert.

## v0.30.1 — 2026-04-29

Page-key scrolling now reaches the inspect panel from the master list on
both the workspace history and the release history screens.

- `pgup` / `pgdown` (and the `ctrl+↑` / `ctrl+↓` aliases) pressed while
  the master list is focused are intercepted before bubbles' list paging
  and routed to the dual panel's active right-hand viewport (Body or
  Stages, matching the current inspect mode).
- `ReleaseDualPanel` and `HistoryDualPanel` now also accept `ctrl+up` /
  `ctrl+down` as aliases for `pgup` / `pgdown`. `ReleaseHistoryView`
  gains an `UpdatePanel` helper mirroring `HistoryView.UpdatePanel`.

### Usage

On the History tab and the Release main menu, `pgup` / `pgdown` (or
`ctrl+↑` / `ctrl+↓`) now scrolls the right-hand viewport of the inspect
panel without leaving the master list.

## v0.30.0 — 2026-04-29

Cache + prefetch for the release history selection, with an inline
spinner on cache misses.

- `ReleaseHistoryView` now owns a per-release-ID cache of the resolved
  `git.LookupCommitMessages` map plus an in-flight set so the same
  fetch can't be spawned twice. Re-selecting a previously visited
  release is instant; the cache lives for the session.
- Cursor moves through the master list trigger a background prefetch
  of the ±2 neighbour releases as a `tea.Batch` of fetch commands, so
  steady scrolling stays inside the cache. Already-cached or
  in-flight neighbours are skipped.
- Cache misses on the selected release now keep the previous dual
  panel content on screen and light the existing `WritingStatusBar`
  spinner. The spinner stops the moment the resolved message lands;
  stale prefetch results that arrive after the cursor has moved on
  warm the cache without touching the visible chrome.
- New `releaseCommitsResolvedMsg` carries `(releaseID, release,
messages, calls, fromSelected)` so the dispatch loop can tell
  on-demand fetches (which redraw + stop the spinner) from prefetch
  fetches (which only seed the cache).
- The full-screen `releaseHistorySyncMsg` path now also seeds the
  cache and triggers a neighbour prefetch when it lands, so the very
  first cursor move after entering release mode is already warm.

### Usage

No new keys. Navigation in the release master list is the same; the
spinner appears automatically on the right of the status bar while a
release's commit messages are being fetched and disappears once the
data is on screen.

## v0.29.2 — 2026-04-29

Tag-icon refresh in the Release inspect commits list.

- Wider gap between the per-tag glyph and the short hash in each
  commit row (`tag_icon  hash`) so the icon reads as a distinct
  token from the colored hash pill.
- New glyphs: ADD → `nf-oct-diff_added`, DEL → `nf-fa-delete_left`,
  DOC → `nf-fa-book_journal_whills`, WIP → `nf-fa-hammer`,
  STYLE → `nf-seti-stylelint`. UI gets its own
  `nf-fa-window_restore` glyph (still aliased to STYLE for colour).
- `IconForCommitTag` now resolves the direct tag before falling
  back through `commitTypeAliases`, so tags like UI can carry their
  own icon while still inheriting the alias palette.

## v0.29.1 — 2026-04-29

Final layout pass on the Release inspect commits list.

- Commit row collapses to `tag_icon  hash`. The per-tag glyph
  (`styles.IconForCommitTag`) carries the tag identity on its own —
  no chip, no separator — so each row stays a single compact token.
  Untagged commits fall back to the generic `nf-cod-git_commit`
  glyph. The hash keeps the tag's pill colour; the icon stays on
  the neutral muted/FG palette.
- The preview title row now leads with the `nf-cod-git_commit`
  glyph (muted) before the `[TAG]` pill so the preview header
  echoes the inspect-list row identity without repeating the
  per-tag glyph.

## v0.29.0 — 2026-04-29

Per-tag icons + uniform tag chips in the Release inspect commits list.

- New `styles.IconForCommitTag(tag, useNerdFonts)` returns a per-type
  Nerd Font glyph (with ASCII fallbacks): `nf-oct-diff_added` for ADD,
  `nf-oct-diff_removed` for DEL, `nf-fa-bandage` for FIX,
  `nf-cod-book` for DOC, `nf-fa-spinner` for WIP,
  `nf-fa-paint_brush` for STYLE, `nf-cod-refresh` for REFACTOR,
  `nf-fa-flask` for TEST, `nf-fa-tachometer` for PERF, `nf-md-broom`
  for CHORE, `nf-fa-cogs` for BUILD, `nf-cod-server_process` for CI,
  `nf-fa-undo` for REVERT, `nf-fa-shield` for SEC. Aliases (`UI →
STYLE`, `REL → BUILD`, etc.) inherit the alias's icon.
- The release commit row now renders the tag itself as a
  uniform-width chip (same `CommitTypeChipInnerWidth` + center
  align + padding the History MasterList uses for its type block),
  so every row's chip lines up perfectly regardless of tag length.
- `Theme.UseNerdFonts` exposed on the theme struct so renderers can
  branch on font support without re-plumbing the config object
  through the model. Set by `applySymbols`.

### Usage

- Open Release mode and inspect any release. Each commit row reads
  `commit_icon  hash  -  tag_icon  TAG_CHIP`, where the chip is
  always the same width whether the tag is `ADD` or `REFACTOR`, and
  every tag carries its own glyph.

## v0.28.3 — 2026-04-29

Release inspect commits row: icons now sit on a neutral palette and the
tag glyph has breathing room from its label.

- `commit_icon` / `tag_icon` switch to `theme.Muted` (and `theme.FG`
  bold when the row is active) instead of inheriting the pill colour.
  The coloured pill is now exclusively the hash + tag duo, so the
  icons read as decoration around the chip rather than part of it.
- Added a second space between `tag_icon` and `TAG` so the bandage
  glyph — wide on most nerd-font terminals — doesn't crowd the
  coloured tag token.

## v0.28.2 — 2026-04-29

Release inspect commits list: brought the tag back as a sibling to the
hash, both wearing the same chip palette, and prefixed each side with
its own nerd-font glyph.

- New `Symbols.GitCommit` and `Symbols.Tag` entries — ``
  (nf-cod-git_commit) and `` (nf-fa-bandage) when nerd fonts
  are on; `*` and `#` as ASCII fallbacks otherwise.
- Commit rows now render as `commit_icon hash - tag_icon TAG` with
  every coloured token sharing the same `CommitTypeMsgStyle` /
  `CommitTypeBlockStyle` chip (strong palette under the cursor, dim
  otherwise). The dim `-` separator stays neutral so the eye reads
  the two halves as one composite chip. Untagged commits collapse to
  `commit_icon hash`.

## v0.28.1 — 2026-04-29

Fix the loading popup flashing top-left before centering on Release-mode
boot.

- `View()` now short-circuits to an empty (alt-screen) frame while
  `model.width` / `model.height` are still zero. bubbletea delivers
  the initial `tea.WindowSizeMsg` shortly after `Init()`; the previous
  first frame ran with zero dimensions, so `lipgloss.Place(0, 0, …)`
  rendered the loading panel flush at (0, 0) and re-centered only once
  the size landed. Skipping that frame removes the visible jump — the
  user only ever sees the centered version.

## v0.28.0 — 2026-04-29

Loading screen on Release mode entry so the slow git+db lookups don't
flash a half-painted UI.

- New `releaseLoading` flag on the Model + `releaseHistorySyncMsg` for
  the async result. While true, `view.go` renders a centered loading
  panel (`release_loading.go`) instead of the release chrome — a
  rounded box with the bubbles spinner glyph, a "Loading releases"
  title, a muted "resolving commit subjects…" subtitle, and the
  workspace path as a hint.
- `enterReleaseHistoryLoading(model)` is the helper used at every
  entry point (`startup` via `Init`, post-create-release transition,
  CommitMode→ReleaseMode swap, reword setup) — flips the flag and
  returns the batch of commands that drive the loading state:
  `startReleaseHistorySync` (which runs the per-hash git lookups +
  ai_calls fetch off the main goroutine) plus the spinner tick.
- The model-level spinner is now ticked through a global
  `spinner.TickMsg` handler in `update.go` while
  `releaseLoading` is set, so the loading frame animates smoothly.
  The pipeline tab keeps owning its own spinner — `spinner.Update`
  only consumes ticks for the matching id, so the two never collide.
- Cursor moves inside `stateReleaseMainMenu` keep the synchronous
  sync (the per-release lookup is still N git calls but the user is
  already inside the screen — a brief pause feels expected, and
  swapping the dual panel out for a loading frame on every move
  would be jarring).

### Usage

- Enter Release mode (`ctrl+s` from the workspace, or boot the app
  with `--release`-equivalent state). You'll see a centered
  "Loading releases" panel with the spinner glyph until the lookup
  finishes; the regular framed view takes over the next frame.

## v0.27.1 — 2026-04-29

Release inspect list: simplified rows + visual separator above the commits.

- The synthetic top entry stays type-aware (`[release]` / `[merge]`)
  but now carries a muted `· output` suffix so the user reads it as
  "the AI output of this record" instead of confusing it with one of
  the inner commits. The right header echoes the same identity —
  `release output` / `merge output`.
- Added a non-selectable separator row between the output entry and
  the first commit. `CycleLeftCursor` skips entries flagged as
  `isSeparator`, so `ctrl+]` from the output row lands on the first
  commit (and `ctrl+[` from the first commit jumps back to the
  output row) without resting on the gap. The row paints as a muted
  rule sized to the inner column.
- Commit rows compacted to `· short_hash` only — and the tag's
  palette (`CommitTypeMsgStyle` / `CommitTypeBlockStyle`) is applied
  to the hash itself, so each row reads as a colored chip rather
  than carrying a separate `[TAG]` badge. The scope pill and title
  are gone from the list; subject and body live in the right
  viewport, where the user actually reads them.
- The "commits" header counter now skips the output entry and the
  separator so it reflects the real number of commits in the release.

## v0.27.0 — 2026-04-29

Release inspect panel: redesigned commits list and commit preview.

- Inspect list rows now show a bare 7-char short hash (sliced — never
  appended with `…`).
- When a commit subject leads with a `[TAG]` cue (the project's
  convention: `[ADD]`, `[UI]`, `[FIX]`, …), the tag is extracted and
  rendered as a pill using the same `CommitTypeMsgStyle` /
  `CommitTypeBlockStyle` palette the History list uses for commits, so
  visual identity stays consistent across screens. The bracketed prefix
  is stripped from the displayed subject so the chip and the title
  don't repeat the tag.
- Right preview is now multi-section for commit entries: bold subject
  (with the tag pill restored at the front when present) → blank →
  muted rule → blank → body. Empty bodies fall back to a muted
  `(no body)` so the panel never collapses.
- Right header for commit entries no longer parrots the hash. It now
  simply reads `commit  preview` — the hash lives only in the inspect
  list on the left, where it's actually useful for picking. Release /
  merge entries keep their existing labels.
- `git.LookupCommitMessages` replaces `LookupCommitSubjects` — same
  per-hash strategy but `--format=%s%x00%b` so subject + body come
  back in a single call. The dual panel now hydrates both fields per
  selection.

### Usage

- On the Release tab, select any release. The Commits / Body inspect
  shows each commit with its short hash, optional `[TAG]` pill and
  remaining subject. Cycle entries with `ctrl+]` / `ctrl+[`. The
  right viewport shows the title, a separator, and the full commit
  body.

## v0.26.1 — 2026-04-29

Fix the empty subjects in the Release inspect panel's commit list.

- `git.LookupCommitSubjects` was using a single bulk
  `git log --no-walk <hashes…>` call. The moment one revision is
  invalid (rebased away, shallow clone, typo), git exits non-zero and
  the entire batch returns empty, leaving every commit row in the
  inspect panel rendering `(no subject)`.
- Switched to per-hash `git show -s --format=%s <hash>`, dropping
  stderr and inserting `""` for unresolved hashes. Releases usually
  carry well under 50 commits, so the per-hash cost is below
  user-perceptible while making the lookup robust against stale
  history.

## v0.26.0 — 2026-04-29

Brought the Release main menu in line with the History (workspace)
redesign: framed view with filter, master list, mode bar and dual inspect
panel. The create-release flow stays untouched.

- New `release_filter_bar.go` with mode pills `TITLE` / `TYPE` /
  `VERSION` / `BRANCH`. `TYPE` filters by `REL` / `MERGE`.
  `currentReleaseFilterMode` is the shared cursor read by
  `HistoryReleaseItem.FilterValue`.
- `release_main_menu_list.go` rewritten with a dense single-line
  delegate that mirrors `HistoryCommitDelegate` — id + type chip +
  version pill + title + date. REL rides the `ADD` (green) palette,
  MERGE rides `STYLE` (purple).
- `release_dual_panel.go` mirrors `HistoryDualPanel`. Modes:
  - **Commits / Body**: left list shows a synthetic `[release]` /
      `[merge]` row first (✦ glyph) and one row per commit hash from
      `Release.CommitList`, enriched with the subject resolved via
      `git.LookupCommitSubjects`. Right viewport renders the body of
      the active entry — release body when `[release]` is active,
      commit subject otherwise.
  - **Stages / Response**: single `[1] ✦ Release Builder` entry today.
      Right side reuses the same header → blank → output → blank →
      rule → blank → telemetry layout. Telemetry strip wired through
      `renderStageStatsLine` reading from `stageStats[4]`. The
      create-release flow doesn't flush ai_calls yet, so the strip
      prints `(no telemetry stored)` until phase C.
- `release_history_view.go` orchestrates filter + master list + mode
  bar + dual panel inside one rounded outer frame, identical to
  `HistoryView`. `HistoryModeBar.SetLabels` reused so the same
  component renders "Commits / Body" / "Stages / Response" pills
  instead of the workspace defaults.
- New keymap on `releaseMainListKeys`: `ctrl+e` swap inspect mode,
  `ctrl+]` / `ctrl+[` cycle entries (wraps around), `R` jumps back to
  the synthetic release row, `?` opens the popup with the new
  `releaseKeybindings` group, `/` focuses filter, `ctrl+f` cycles
  modes. Helper bar trimmed accordingly.
- `git.LookupCommitSubjects(hashes)` does one `git log --no-walk`
  call to map abbreviated hashes back to subjects; cached on the
  dual panel per release so cycling stays git-call-free.
- `syncReleaseHistorySelection` is the release counterpart of
  `syncHistoryViewSelection`. Hooked at every transition into
  `stateReleaseMainMenu` (model init, post-create-release,
  CommitMode→ReleaseMode swap, reword setup, esc back from
  build-text) so the dual panel is hydrated from frame zero.
- Tiny pre-existing fix while in the area: the post-create-release
  transition was setting `mainListKeys()` instead of
  `releaseMainListKeys()`; corrected.

### Usage

- Switch to Release mode (`ctrl+s` from the workspace) to land on
  the new framed view.
- `/` focuses the filter input; `ctrl+f` cycles between
  `TITLE/TYPE/VERSION/BRANCH`.
- `ctrl+e` toggles inspect mode between `Commits / Body` and
  `Stages / Response`.
- In `Commits / Body`, `ctrl+]` / `ctrl+[` walk the commits list
  (wrap-around). `R` jumps to the synthetic `[release]` row from any
  commit. The right viewport shows the release body or the selected
  commit's subject.
- In `Stages / Response`, the single `Release Builder` stage shows
  the same telemetry strip as the workspace pipeline; once
  `release_ai_calls` lands, it'll start filling in.
- `?` opens the keybindings popup with the full release shortcut set.

## v0.25.1 — 2026-04-29

History inspect polish in Stages/Response mode.

- Stage list now uses the same titles the Pipeline tab uses
  (`Change Analyzer`, `Commit Body`, `Commit Title`,
  `Changelog Refiner`) so both views read identically.
- Each list row gets a leading `✦` glyph: muted on idle rows, brand
  Secondary on the active one — the cursor now reads at a glance
  even before the user notices the bold name.
- Right panel layout rebanded into three legible sections separated by
  blank rows + a muted rule: header → blank → output → blank → rule →
  blank → telemetry. `SetSize` reserves 6 rows of chrome so the
  viewport never overlaps the strip.

## v0.25.0 — 2026-04-29

History inspect: per-stage telemetry strip in `Stages / Response` mode.

- New row rendered between the right header and the stage viewport
  showing tokens (in / out / total), wall-clock duration and the per-call
  TPM consumption bar — the same format the live pipeline cards use,
  rendered through the existing `renderStageStatsLine`. Switches with
  the user's `ctrl+]` / `ctrl+[` cursor.
- `HistoryDualPanel` caches the full `[]storage.AICall` for the
  inspected commit on every selection sync (`stageStats [4]pipelineStage`)
  so cycling stages is O(1) and we replace the previous one-shot
  "does the changelog stage exist?" probe with a single SQLite read
  that doubles as the data source for the telemetry strip.
- `SetCommit` signature changed: now takes `[]storage.AICall` instead
  of a `hasChangelog bool`. `hasChangelog` is derived inside, falling
  back to `c.IaChangelog != ""` for legacy rows whose ai_calls flush
  failed.
- `SetSize` reserves an extra row in stages mode so the telemetry strip
  never eats the viewport. KeyPoints/Body mode keeps the original
  budget.
- `loadCommitAICalls` (in `update_commit.go`) replaces
  `commitUsedChangelogStage`; it logs and returns nil on lookup
  failures so a transient SQLite hiccup just hides telemetry instead
  of breaking the panel.

### Usage

- On the History tab, switch to `Stages / Response` (`ctrl+e`) and
  cycle stages with `ctrl+]` / `ctrl+[` — the row directly below the
  stage name shows that stage's tokens, latency and TPM usage. Stages
  without stored telemetry render `(no telemetry stored)`.

## v0.24.0 — 2026-04-29

Persist the changelog refiner output on the commit row so the History
inspect panel can show stage 4 with its actual content.

- New `commits.ia_changelog` column added through
  `applySchemaMigrations`. Existing rows backfill to `''` so legacy
  commits keep working.
- `storage.Commit` gains `IaChangelog string`; every CRUD path
  (`GetCommits`, `CreateCommit`, `SaveDraft` insert + update,
  `FinalizeCommit`) now reads/writes the column.
- TUI plumbing: the model's `iaChangelogEntry` is mirrored onto
  `currentCommit.IaChangelog` at the same points the other AI fields
  are flushed (`transitions.commitMessage`, draft save in
  `update_writing.go`). When a commit is opened from the History tab
  (Edit / Continue draft), `iaChangelogEntry` is now rehydrated from
  the row.
- `HistoryDualPanel.SetCommit` uses `c.IaChangelog` as the stage 4
  output. `commitUsedChangelogStage` short-circuits on the persisted
  text and only consults `ai_calls` as a fallback for legacy commits
  saved before the column existed (those still get the list entry but
  the right viewport prints the placeholder).

### Usage

- Rerun the AI pipeline so the changelog refiner produces an entry,
  then save the commit (or save it as a draft). On the History tab
  you'll now see the actual rendered text under `[4] changelog` in
  Stages / Response mode.

## v0.23.0 — 2026-04-29

History inspect-panel polish: rebound the swap-mode key, made the AI
stages list wrap, and exposed the optional stage 4 (changelog) when it
ran for the inspected commit.

- Swap inspect mode now lives on `ctrl+e` (was `ctrl+m`). Helper bar,
  popup hint, mode-bar pill caption and ascii diagrams in the source
  comments updated to match.
- `HistoryDualPanel.CycleLeftCursor` now wraps the stages list at both
  ends (modular index) so `ctrl+]` past the last stage lands on the
  first one and vice versa. Key-points navigation keeps its clamped
  behaviour (hard limits read better there).
- `SetCommit` accepts a `hasChangelog` flag and appends a 4th entry
  ("changelog") to the stages list when the inspected commit has a
  stored ai_calls row tagged as `changelog`. The changelog output text
  isn't persisted yet, so the right viewport prints a muted
  "(stage output not stored)" placeholder; the entry is still useful
  because it confirms the refiner ran for that commit.
- `update_commit.commitUsedChangelogStage` looks up `ai_calls` on every
  cursor sync to derive the flag.

### Usage

- On the History tab: `ctrl+e` swaps between `KeyPoints / Body` and
  `Stages / Response`. In the latter, `ctrl+]` / `ctrl+[` walk the
  stages list and now cycle around the ends. If the changelog stage
  ran, a 4th `[4] changelog` entry appears in the list automatically.

## v0.22.2 — 2026-04-29

Themed the keybindings popup hint and codified the rule.

- `keybindings_popup.go` now renders its bottom hint through
  `theme.AppStyles().Help` (`ShortKey` / `ShortDesc` / `ShortSeparator`),
  matching `scope_popup.go`. No more flat-muted hint line.
- `CLAUDE.md` gains a Code Convention entry making this the project rule
  for all future popups and on-screen key-hint lines.

## v0.22.1 — 2026-04-29

Made the bottom help bar width-aware so it never overflows on narrow
terminals while keeping the `?` popup hint always visible.

- `renderHelpEntries` now budgets against `model.width` minus the
  `Padding(0, 2)` chrome. Trailing entries that don't fit are dropped,
  and the `?` entry (when present) is pulled out of the flow and pinned
  to the right edge so the popup remains discoverable at every width.

## v0.22.0 — 2026-04-29

Added stage cycling on the workspace inspect panel and a `?` popup to keep
the bottom hint line uncluttered.

- New `ctrl+]` / `ctrl+[` shortcuts on the History tab cycle the dual
  panel's left cursor in both inspect modes (key points and stages /
  response).
- New `?` popup (`keybindings_popup.go`) lists the full keymap for the
  current state — currently wired for `stateChoosingCommit`. Closes with
  `?`, `esc`, or `q`. The global `?` handler keeps the bubbles
  `help.ShowAll` toggle as a fallback for states without a dedicated
  popup. Skipped while the History filter input is focused so `?` can
  still be typed into the filter.
- Workspace help line gains a `^]/^[` cycle entry (label switches
  between "cycle keypoint" and "cycle stage" depending on inspect mode),
  a `^m` swap-mode reminder and a `?` "more" hint.

### Usage

- On the History tab: press `ctrl+]` / `ctrl+[` to walk the inspect
  panel's left cursor (key point or stage). Press `?` to open the full
  keybindings popup; close it with `?` or `esc`.

## v0.21.2 — 2026-04-29

Themed the helper hint inside the scope popup so keys and descriptions
match the rest of the UI's help styling.

- `scope_popup.go` now renders the bottom hint by composing
  `theme.AppStyles().Help` (`ShortKey` for keys, `ShortDesc` for
  descriptions, `ShortSeparator` for the `·` divider) instead of a
  single muted-coloured string.

## v0.21.1 — 2026-04-28

Fixed the workspace filter pill rendering blank on the first frame.

- `currentMainFilterMode` is now explicitly initialised to
  `FilterModeTitle`. Go's zero-value would resolve to TITLE today via
  iota anyway, but the explicit assignment removes the ambiguity and
  documents the intended startup mode.
- `HistoryFilterBar.View` now falls back to the TITLE meta entry when
  the meta-map lookup misses, so the pill is always labelled instead
  of rendering as an empty bar (which is what the "no initial value"
  symptom looked like in the UI).

### Usage

No behavior change for users — the bar simply always shows a labelled
pill from the very first paint.

## v0.21.0 — 2026-04-28

Added a cycleable filter-mode pill to the workspace filter bar.

- New `MainFilterMode` enum drives which commit field
  `HistoryCommitItem.FilterValue` exposes to the bubbles list filter:
  TITLE (default), ID, TYPE or SCOPE. The active mode is held in a
  package-level variable so cycling re-evaluates the filter pass live
  without rebuilding items.
- `ctrl+f` (handled in `updateChoosingCommit`) cycles the mode in the
  canonical title → id → type → scope order and re-applies the current
  query so the visible rows update immediately.
- `HistoryFilterBar.View` now renders a flat colour pill for the
  current mode in front of the filter input — `[MODE]  >  <input>` —
  using `CommitTypeMsgStyle` (the dim `BgMsg`/`FgMsg` palette) so each
  mode reads as its own quiet tag. Per-mode palettes: TITLE uses ADD
  (green), ID uses WIP (amber), TYPE uses STYLE (purple), SCOPE uses
  SEC (pink/red).

### Usage

`/` opens the filter input as before. From there, `ctrl+f` cycles the
mode pill (works any time on the workspace view, even with the input
unfocused or empty — the pill swaps colour/label and any active query
gets re-evaluated against the new field). Esc still clears + blurs;
Enter still blurs without clearing.

## v0.20.13 — 2026-04-28

Narrowed the main-view list filter to the commit title only.

- `HistoryCommitItem.FilterValue` now returns just `commit.MessageEN`
  (the visible title) instead of folding in `Type`, `Scope` and
  `KeyPoints`. The `/` filter on the workspace list now matches what
  the user reads on each row, instead of surfacing rows whose title
  doesn't contain the query just because the type or a key point did.

### Usage

`/` on the main view still opens the filter input; the matching rule
is just stricter now (title substring only).

## v0.20.12 — 2026-04-28

Differentiated the History view's mode-swap pills by border color.

- `HistoryModeBar.renderPill` now keeps the Secondary border only on the
  active pill; the idle pill drops to a Muted border, matching the
  dimmed-border convention used elsewhere in the TUI. Previously both
  pills shared the same Secondary border and only differed by text and
  background, which made the segmented unit harder to read at a glance.

### Usage

Purely visual; no new keys. ⌃M still toggles between
`KeyPoints / Body` and `Stages / Response`.

## v0.20.11 — 2026-04-28

Added a persistent git-branch pill next to the CWD pill in the top bar.

- New `statusbar.RenderBranchPill` renders a `GIT <branch>` two-segment
  pill using the INFO palette (`#2c4360`/`#d6e4f4` label,
  `#182230`/`#b8c5d4` body) so it sits visually next to the existing
  CWD pill but reads as its own tag.
- `Model.currentBranch` is resolved once at startup via
  `git.GetCurrentGitBranch` and reused on every render — no shell-out
  per frame. When git fails (e.g. not a repo) the branch is left empty
  and the branch pill is omitted, falling back to the previous CWD-only
  layout.
- Top tab bar splits the pill budget between the CWD (60%) and branch
  (40%) pills with a 1-cell gap between them. Below the combined
  width threshold the branch pill is dropped first; below the CWD
  threshold the spacer reverts to plain whitespace.

### Usage

Purely visual; no new keys or config. The branch pill reflects the
branch the TUI was launched from. If you switch branches outside
CommitCraft mid-session, restart the TUI to refresh the pill.

## v0.20.10 — 2026-04-28

Tightened the compose metadata header further and tweaked section labels.

- `commit type` and `scope` now share a single horizontal row when the
  compose panel is wide enough to fit both with a 4-cell gap. When the
  panel is narrower they fall back to the previous two-row stack so
  nothing gets clipped.
- Blurred section-label pills (`SectionPill`) now use the theme
  Secondary colour instead of Muted, so the labels stay readable as
  brand chrome rather than fading into the background when out of
  focus. Focused pills are unchanged (Primary background, BG text).

### Usage

Purely visual; no new keys. The two-row fallback kicks in
automatically below the width threshold — there is nothing to
configure.

## v0.20.9 — 2026-04-28

Collapsed the compose metadata header so each row uses a single line.

- `commit type` row now renders inline as `[label] [selected pill]`
  instead of stacking the label above a wrapped grid of every available
  type. Switching type still happens through the type popup, so the
  compose panel only displays the active selection — recovering the
  vertical space the previous grid consumed.
- `scope` row now renders inline as `[label] [file pill] [edit]` on a
  single horizontal line. The bordered chip is gone; the file is
  rendered as a flat coloured pill in the commit-type style with a
  per-file deterministic palette (FNV-hashed onto the existing
  commit-type colour set), so the same file always gets the same colour
  and different files are visually distinct.
- `edit` is also a flat pill now so it visually aligns with the file
  chip on the same row.

### Usage

No new keys. Picking a different commit type still goes through the
type popup; picking a scope still goes through the file picker. The
change is purely visual — the metadata block is now two lines tall
instead of six, leaving more room for the keypoints area.

## v0.20.8 — 2026-04-28

Unified the commit-type chip across every surface.

- New shared constant `styles.CommitTypeChipInnerWidth = 8` is the
  canonical content width of any commit-type chip. Chips render as
  `Width(8) + Padding(0,1) + Align(Center)`, so every pill measures
  exactly 10 cells regardless of tag length and the tag sits visually
  centered (no lopsided trailing whitespace). The cap fits every
  default tag fully — `REFACTOR` (the longest at 8 chars) is the
  upper bound; longer custom tags are hard-truncated by the caller.
- Compose-view commit-type pills (`commitTypeChip` in
  `compose_sections.go`) now use the shared palette helpers
  (`CommitTypeBlockStyle` / `CommitTypeMsgStyle`) and the same fixed
  Width + Center alignment — replacing the previous flow where the
  active pill used the per-type `hex` color and inactive pills had a
  ghost border. Active = block (strong); inactive = msg (dim).
- History MasterList and the commit-type popup selector picked up the
  same constant + alignment, so a tag's chip looks identical on every
  surface where it can appear.

### Usage

No new keybindings. Visual change only.

## v0.20.7 — 2026-04-28

Applied the History MasterList styling rules to the commit-type
selector — both the full-screen `stateChoosingType` list and the
in-place compose popup (`Ctrl+T`) now render rows the same way:

- Type chip uses the strong (block) palette under the cursor and the
  dim (msg) palette on the rest of the rows — fixed inner width
  (`commitTypeMaxTagLen=6`) so descriptions line up across rows.
- Description gets msg-palette + Bold under the cursor, plain Muted
  text otherwise (mirroring the History title behaviour).
- Cursor `❯` indicator preserved on the selected row in `theme.Secondary`.
- Per-type `Color` field from the user TOML config is no longer used —
  the new shared palette helpers
  (`styles.CommitTypeBlockStyle` / `CommitTypeMsgStyle`) drive the
  colors so every surface stays consistent.
- `CommitTypePopupContentWidth` updated to match the new fixed-chip
  row shape (`"❯ [chip] desc"`) so the popup sizes correctly.

## v0.20.6 — 2026-04-28

History MasterList ID and date columns now share the title's selection
treatment: msg-palette colors + Bold under the cursor, plain Muted text
on the rest of the rows. The whole row reads as a single styled unit
when focused.

## v0.20.5 — 2026-04-28

History MasterList type chip now follows the same selection rule as the
scope pill and title: strong (block) palette under the cursor, dim (msg)
palette on the rest of the rows. The chip is still always rendered, only
its intensity changes — so the cursor row reads as a fully lit-up
identity card while the surrounding list stays calm.

## v0.20.4 — 2026-04-28

Refined how the four-color commit-type palette is applied in the
History MasterList so the cursor row pops without making the rest of
the list visually noisy.

- Type chip stays **always active** (block colors, bold, fixed width).
  It is the row's primary identity marker so it does not dim with
  selection state.
- Scope and title now switch on selection:
  - **Selected row**: scope is rendered as a pill with the dimmer
      `bg msg` + `fg msg` colors (helper `CommitTypeMsgStyle` +
      `Padding(0,1)`); title uses the same msg colors with `Bold`.
  - **Unselected row**: original look restored — scope is plain
      `Secondary`-colored text with a `": "` separator and the title is
      `Muted` text without a background.

### Usage

No new keybindings. The cursor row is the only one that "lights up"
with the full palette, leaving the rest of the list as a clean stream.

## v0.20.3 — 2026-04-28

Activated colored chips/pills across the History list using the full
4-color commit-type palette.

- Added a `commitTypeAliases` map so legacy CommitCraft tags (`IMP`,
  `REM`, `REF`, `MOV`, `REL`) and the project-specific `UI` resolve to
  the closest semantic palette entry (`IMP`/`REF` → `REFACTOR`, `REM` →
  `DEL`, `MOV` → `CHORE`, `REL` → `BUILD`, `UI` → `STYLE`). Tags still
  unmatched fall back to the neutral theme palette.
- New shared helpers in `internal/tui/styles/commit_type_palette.go`:
  - `CommitTypeBlockStyle(theme, tag)` returns the bg/fg pair for chips
      and pills.
  - `CommitTypeMsgStyle(theme, tag)` returns the dimmer bg/fg pair for
      surfaces like the row title.
- Applied the new helpers to the MasterList delegate:
  - Type chip: bg block + fg block (already used; now goes through the
      helper).
  - Scope: rendered as a pill with the same bg/fg block colors as the
      type chip, padded `0 1` for visual breathing room.
  - Title: bg msg + fg msg dim companions of the type, with `Bold` on
      the selected row to keep the cursor distinguishable.

### Usage

Other panels can opt into the same palette by calling
`styles.CommitTypeBlockStyle(theme, tag)` /
`styles.CommitTypeMsgStyle(theme, tag)` on top of any base style.

## v0.20.2 — 2026-04-28

Visual polish on the History layout following the v0.20.0 redesign.

- Outer rounded frame now uses the brand primary color so the History
  view stands out against the surrounding chrome.
- MasterList type chips are width-stabilised: every chip occupies the
  same `Width(maxTypeTagLen)+Padding(0,1)` cells, so the message column
  starts at the same column on every row regardless of tag length. Tags
  longer than the cap (e.g. `REFACTOR`) are hard-truncated to fit.
- ModeBar pills restored to `●` / `○` bullets (active / idle) and now
  share the secondary brand color on the border. Active state is
  signalled through bold text + brand-primary fg, not via border color.
- DualPanel left column widened by 30 % (28 → 37 cells) so key points
  have room to breathe. The keypoint list now renders with the same
  `"  > "` prompt + secondary-color style used by the compose
  KeyPointsInput, keeping the History and Compose surfaces visually
  consistent.

### Usage

No new keybindings or config knobs.

## v0.20.1 — 2026-04-28

Render fixes for the History layout introduced in v0.20.0. The four-zone
view now draws as a single continuous rounded frame that spans the full
terminal width without overflowing vertically, and the FilterBar /
ModeBar no longer wrap their content.

- Fixed a lipgloss `Style.Width` footgun in `history_view.go`: passing
  the inner content width to the outer bordered frame caused lipgloss to
  word-wrap every row at `width − borderSize`, which split the FilterBar
  counter, doubled each `─` divider, pushed `^M swap` onto a second line,
  and made the whole stack ~2× taller than the terminal. The frame now
  receives the total width and an explicit `Height` so the rendered
  region matches the assigned surface exactly.
- `view.go` `stateChoosingCommit` now uses `model.width` and a manual
  height calc (mirroring `statePipeline`) instead of
  `availableWidthForMainContent` / `availableHeightForMainContent`,
  reclaiming the horizontal margin and the 20 % vertical shave that the
  shared calc applied for an `appStyle` that was never rendered.
- Replaced ambiguous-width Unicode glyphs in the History chrome with
  ASCII so the layout no longer drifts on terminals that render
  East-Asian-width variants of `›`, `·`, `…`, `⌃`, `●`, `○` as 2 cells.
  The dense MasterList delegate keeps its colored type chip; only the
  decorative markers changed.
- Hardened the FilterBar composition: the input view is hard-truncated
  with `ansi.Truncate` and the row is force-padded to exactly `f.width`
  cells so a tiny width-counting drift can never wrap the counter again.

### Usage

No new keybindings or config knobs. The History view (`stateChoosingCommit`)
should render correctly across normal terminal widths (≥ 80 cols) and
fill the full terminal height.

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
- Press `Ctrl+M` to swap the DualPanel between _KeyPoints / Body_ and
  _Stages / Response_.
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
  because that bucket header _does_ refresh on every call.

## v0.18.2 — 2026-04-28

Fix: rate-limit bars (compose RPD/TPM, picker REQ/TOK) sometimes rendered
fully consumed (100%) for models whose response omitted an
`x-ratelimit-remaining-*` header. The parser left the field at 0 and the
formula `used = limit - remaining` collapsed to `used = limit`.

- `api.RateLimits` gains `RequestsParsed` / `TokensParsed` flags. The
  parser only sets each flag when _both_ halves of the bucket
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
- On the Compose tab, focus the _pipeline models_ section and observe the two
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
- **Diff sub-block always renders.** Layout math in `renderPipelinePanel` was reserving stage card heights first (including focused-stage growth) and _then_ trying to fit the diff with the leftover, which collapsed to 0 once the final-commit card appeared. Order reversed: stages-at-default + `DiffMinHeight` are reserved up front; only the _leftover_ is spent on focused-stage growth. Diff now keeps a guaranteed floor (default 6 rows) even after the AI flow finishes.
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
  - `LevelInfo` · neutral hints, ready / idle states.
  - `LevelSuccess` · successful completion.
  - `LevelWarning` · recoverable issue.
  - `LevelError` · failure that blocks the user.
  - `LevelFatal` · unrecoverable error.
  - `LevelAI` · AI / model activity.
  - `LevelRun` · long-running op in progress.
  - `LevelDebug` · verbose-only trace.
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
