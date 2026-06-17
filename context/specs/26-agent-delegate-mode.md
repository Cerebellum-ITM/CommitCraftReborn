# Unit 26: agent-delegate-mode

## Goal

Add a **delegate mode** to the headless `ai` CLI so that, when CommitCraft is
driven by an AI agent (the `commitcraft` skill), the message-generation stages
**do not call the Groq API**. Instead the CLI emits a **prompt bundle** (the
same prompts, filled with the diff/keypoints/tag/scope) and the calling agent —
which is already running — produces the commit message itself, then hands it
back through a new `commitcraft ai submit` subcommand that validates, verifies,
and persists the draft. This removes the serial-API-latency bottleneck (3–4
sequential Groq calls, plus queue time) from the agent-driven flow.

Activation is **global config + per-call flag**; the architecture is **hybrid**:
a `single` strategy (one unified prompt → one inference) by default, and a
`staged` strategy that hands over the original per-stage prompts for the agent
to follow internally. **Both strategies submit exactly once** — "staged" does
not mean multiple CLI round-trips; it only changes which prompt text the agent
receives. Scope covers `generate`, `regenerate`, `merge`, `release`, and the
changelog refiner.

> **Non-negotiable:** the produced commit message (title **and** body) is always
> **English**, in every repo and session. Both the unified prompt and the
> bundle `instructions` state this explicitly.

## Design

### Why delegate beats more Groq calls

The 4-stage decomposition (`change_analyzer → commit_body → commit_title →
changelog_refiner`) is a crutch for a weak model (`llama-3.1-8b-instant`); the
diff is lost after stage 1 and the title is generated two hops away from the
code. A capable agent does better with the full diff present and produces a
coherent title+body in one pass — and pays **zero** API latency. Delegate mode
turns the CLI into a prompt-and-persist harness around the already-running
agent.

### Activation (config + flag)

New `[agent]` table in `~/.config/CommitCraft/config.toml`:

```toml
[agent]
mode     = "delegate"   # "" | "groq" (default) | "delegate"
strategy = "single"     # "single" (default) | "staged"
```

Per-call overrides on `generate` / `regenerate` / `merge` / `release`:

| Flag                       | Effect                                                  |
| -------------------------- | ------------------------------------------------------- |
| `--agent`                  | Force delegate mode on for this call.                   |
| `--no-agent`               | Force the normal Groq pipeline (overrides config).      |
| `--agent-strategy single\|staged` | Override the strategy for this call.             |

`--agent` + `--no-agent` together is a usage error. Absent both, the config
`mode` decides. Existing setups with no `[agent]` table default to `groq` —
**fully backward compatible**, the Groq path is untouched.

### Two strategies, one submit

- **single** (default): the CLI fills **one new unified prompt**
  (`agent_commit.prompt.tmpl`) that merges the analyzer + body + title (+
  changelog) instructions into a single coherent system prompt, paired with a
  user block carrying `TAG / MODULE / DEVELOPER_POINTS / GIT_CHANGES` (+
  changelog context when active). Emitted as `bundle.unified`.
- **staged**: the CLI fills the **existing** per-stage prompts and emits them as
  `bundle.stages[]` (`summary`, `body`, `title`, `[changelog]`), with an
  `inputs` block and `instructions` describing the data flow (stage 1 reads
  keypoints+diff; stage 2 reads your stage-1 output + tag/scope; etc.). The
  agent reasons through them internally and still submits once.

A true per-stage-round-trip mode is explicitly **out of scope** — it would
reintroduce the serial-latency cost this unit removes. Noted as possible future
work.

### Flow (skill-facing)

```
1. commitcraft ai generate --agent -k "…" -t [TAG] -s scope
     → emits a delegate bundle JSON (prompts + diff + inputs + instructions).
       NO Groq call. NO draft persisted. mode="delegate", id=0.
2. (agent) reads the bundle, writes the message following the embedded prompt(s).
3. (agent) echo '<submit-json>' | commitcraft ai submit
     → re-reads the staged diff, composes final_message, runs verify,
       persists the draft, prints the standard commit envelope + a `verify` block.
4. commitcraft ai promote --id <ID>      (unchanged)
5. git commit … ; commitcraft ai link-commit …   (skill, unchanged)
```

`regenerate --agent` emits a bundle built from the stored draft (carries
`action:"regenerate"` + `id`); `submit` with the same `id` updates that draft
instead of creating a new one. `merge --agent` / `release --agent` emit a
release bundle (`kind:"release"`) filled from the commit range; `submit` with
`kind:"release"` persists to the `releases` table.

## Implementation

### A. Config — `internal/config/`

`types.go`:

- New `AgentConfig` struct: `Mode string `toml:"mode"`` + `Strategy string
  `toml:"strategy"``. Add `Agent AgentConfig `toml:"agent,omitempty"`` to
  `Config`.
- Extend `PromptsConfig` with `AgentCommitPromptFile/Prompt` and
  `AgentReleasePromptFile/Prompt` (`Prompt` fields `toml:"-"`, filled at load).
- `NewDefaultConfig`: `Agent{Mode:"", Strategy:"single"}`; prompt file defaults
  `prompts/agent_commit.prompt` and `prompts/agent_release.prompt`.
- Exported constants + normalizers: `AgentModeGroq`/`AgentModeDelegate`,
  `AgentStrategySingle`/`AgentStrategyStaged`, `NormalizeAgentMode`,
  `NormalizeAgentStrategy`.

`loader.go`: add the two new `//go:embed` vars (`agent_commit.prompt.tmpl`,
`agent_release.prompt.tmpl`), the two `case` arms in `createOrLoadPromptFile`,
and load both in `loadIaPrompts`.

### B. New prompts — `internal/config/prompts/`

- `agent_commit.prompt.tmpl` — unified single-pass system prompt. Merges:
  analyzer (correlate keypoints↔diff), body (WHY-not-WHAT, wrap 72, bullets,
  no tag/module echo), title (imperative, ≤50 chars, lowercase, no period,
  tone-by-tag). Instructs: output the commit **title** and **body** (and, when
  changelog context is supplied, a `changelog_entry` + a one-line
  `changelog_mention` containing the token `CHANGELOG.md`). **All output in
  English.** No prose around the result.
- `agent_release.prompt.tmpl` — unified single-pass prompt for `[MERGE]` /
  `[RELEASE]` notes from a commit list (mirrors release_body/title/refine
  intent). English-only.

### C. Delegate builders — `internal/aiengine/delegate.go` (new)

Pure prompt-filling, **no Groq**. Reuse the exact `fmt.Sprintf` user-input
shapes from `engine.go` / `release.go` so staged prompts match what Groq would
have received.

```go
type DelegateStage struct { Stage, System, User string }   // JSON: stage/system/user

type BundleInputs struct {
    Tag, Scope string; KeyPoints []string
    ChangelogActive bool
    Branch, Version string `json:",omitempty"`   // release
}

type DelegateBundle struct {
    Mode, Kind, Action, Strategy string
    ID int `json:",omitempty"`
    Inputs BundleInputs
    Unified *DelegateStage  `json:",omitempty"`  // single
    Stages  []DelegateStage `json:",omitempty"`  // staged
    Instructions  string
    SubmitExample string
}

func BuildCommitBundle(deps Deps, in Input, strategy, action string, id int) DelegateBundle
func BuildReleaseBundle(deps Deps, in ReleaseInput, releaseType, strategy, action string, id int, branch, version, commitList string) DelegateBundle
```

- single → fill `agent_commit.prompt` / `agent_release.prompt` into `Unified`.
- staged → fill the existing per-stage prompts into `Stages` (commit: summary
  uses `DEVELOPER_POINTS/GIT_CHANGES`; body/title carry placeholder notes for
  the agent's upstream outputs; changelog included only when active).
- When `in.ChangelogActive && cfg.Changelog.Enabled`, call `changelog.Detect`
  to add `FORMAT_SAMPLE` + `SUGGESTED_VERSION` + `DATE` to the prompt context
  (single: into the unified user block; staged: as the `changelog` stage). Best
  effort — skip silently if no CHANGELOG.
- `Instructions` states: delegate mode, do not call any API, English-only,
  produce the message from the prompt(s), then pipe the submit JSON to
  `commitcraft ai submit`. `SubmitExample` is a literal schema/command sample.

### D. `ai submit` — `internal/cli/ai/submit.go` (new)

Register in `ai.go` `Dispatch` (`case "submit": return runSubmit(rest)`) + usage
line. Reads a JSON object from **stdin** (or `--input-file <path>`; `-` = stdin):

```jsonc
{ "kind":"commit", "action":"generate", "id":0,
  "tag":"[ADD]", "scope":["cli"], "keypoints":["…"],
  "title":"…", "body":"…", "summary":"…",
  "changelog_entry":"…", "changelog_mention":"…",
  // release-only: "type":"MERGE|RELEASE", "branch":"…", "version":"…", "commit_list":"…"
}
```

- `scope` accepts a string **or** array (custom `flexStrings` unmarshal); joined
  with `\n` like `generate`.
- **commit** (`kind` empty/`commit`): validate `tag` against resolved types,
  require `title`+`body`. `id>0` → load existing draft, keep its `Diff_code`,
  update fields (regenerate). `id==0` → `validateAndStageDiff` for `Diff_code`
  (new draft). Compose `MessageEN = ComposeFinalMessage(title, body,
  changelog_mention)`. `Source="ai"`. `SaveDraft`. Then run
  `VerifyFinalMessage` on the formatted final and attach.
- **release** (`kind:"release"`): build `storage.Release` (`Type`, `Title`,
  `Body`, `Branch`/`Version`, `CommitList`, `Source="ai"`, `Status="draft"`),
  `id>0` updates. `SaveReleaseDraft`. Verify via `composeReleaseFinalMessage`.
- Output: the standard `commitJSON` / `releaseToJSON` envelope plus a new
  optional `Verify *aiengine.VerifyReport `json:"verify,omitempty"`` field
  (nil for every other subcommand). **Exit 0 on successful persist** regardless
  of verify findings — the draft is saved and recoverable; the agent reads
  `verify.has_errors` to decide whether to `ai edit`/re-submit before
  `promote`. Usage errors `2`, runtime errors `1`.

### E. Wire delegate short-circuit into the four subcommands

Shared helpers in a new `internal/cli/ai/agentmode.go`:
`resolveAgentMode(cfg, agentFlag, noAgentFlag) (bool, error)`,
`resolveStrategy(cfg, strategyFlag) (string, error)`, and `printJSON(any)`.

- `generate.go`: add `--agent`/`--no-agent`/`--agent-strategy`. After building
  `in` (diff already staged), if delegate → `printJSON(BuildCommitBundle(deps,
  in, strategy, "generate", 0))`, return 0. No Groq, no DB.
- `regenerate.go`: same flags. After loading the draft + overrides + diff
  resolution, if delegate → `printJSON(BuildCommitBundle(deps, inFromDraft,
  strategy, "regenerate", c.ID))`, return 0.
- `merge.go` / `release.go`: same flags. After resolving the commit range, if
  delegate → `printJSON(BuildReleaseBundle(...))`, return 0.

`commitJSON` gains the `Verify` field (omitempty) in `ai.go`.

### F. Versioning / docs

- Bump `cmd/cli/main.go` `version` to `v0.67.0`.
- `CHANGELOG.md`: `## v0.67.0 — 2026-06-13` with a `### Usage` block covering
  the `[agent]` config table, the `--agent`/`--no-agent`/`--agent-strategy`
  flags, and the `ai submit` JSON contract.
- `context/progress-tracker.md`: mark unit 26.
- **Skill update (paired, separate):** `~/.claude/skills/commitcraft/SKILL.md`
  gains a delegate-mode branch (generate `--agent` → produce message →
  `ai submit` → verify-from-payload → promote). Tracked here but applied to the
  skill repo, not this Go repo.

## Dependencies

- none new. Reuses `godotenv`/`toml` (config), `internal/changelog`
  (`Detect`/`SuggestNextVersion`), `aiengine.VerifyFinalMessage`,
  `ComposeFinalMessage`, and stdlib `encoding/json`/`flag`.

## Verify when done

- [ ] `go build ./...` + `go vet ./...` clean.
- [ ] With no `[agent]` table (or `mode=""`), `ai generate` behaves exactly as
      today (Groq path, draft persisted) — no regression.
- [ ] `ai generate --agent -k … -t … -s …` prints a `mode:"delegate"` bundle
      with the filled prompt(s) + staged diff, makes **no** Groq call, and
      persists **no** draft.
- [ ] `--agent-strategy single` emits `unified`; `staged` emits `stages[]`.
- [ ] `echo '<json>' | commitcraft ai submit` persists a `draft`, returns the
      standard envelope with a populated `verify` block, and is then promotable
      via `ai promote --id <ID>`.
- [ ] `ai submit` with `id>0` updates the existing draft (regenerate path) and
      preserves its stored diff.
- [ ] `ai submit` with `kind:"release"` writes a `releases` row visible to
      `ai show --id <ID> --kind release`.
- [ ] `--agent` + `--no-agent` together → usage error (exit 2).
- [ ] Delegate bundle `instructions` and the unified prompt both state the
      English-only rule.
- [ ] `ai merge --agent` / `ai release --agent` emit a `kind:"release"` bundle
      with the commit list, no Groq call.
- [ ] Version bumped to `v0.67.0`; `CHANGELOG.md` entry with `### Usage` added.
- [ ] `context/progress-tracker.md` updated for unit 26.
