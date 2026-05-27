# Unit 16: ai-verify

## Goal

Add `commitcraft ai verify --id <ID>`: a deterministic, offline checker
that scans a draft's composed `final_message` (the same text that would
go into `git commit`) for known AI-residue patterns, structural defects,
and trailer hygiene problems, and emits a structured JSON list of
findings. The agent (Haiku-driven sub-agent in the new skill flow) can
branch on the exit code instead of having to subjectively review the
text.

This replaces step 6 of the skill ("Review the output") with a
mechanical gate: the agent runs `ai verify`, and only falls back to
human-style judgement when `ai verify` reports clean but the agent
still suspects something.

## Design

No user-visible change to existing flows. The new subcommand is purely
additive. Output JSON is keyed snake_case to match the rest of the
`ai …` surface. Findings have a `severity` (`error` | `warning`) and a
`rule` slug so callers can match deterministically.

Verification rules (deterministic, no Groq call, no diff dependency):

| rule slug                  | severity | what it catches                                                                                  |
| -------------------------- | -------- | ------------------------------------------------------------------------------------------------ |
| `ai_residue_phrase`        | error    | Known leakage phrases in title or body (`here is the commit message`, `paragraph 1`, etc.).      |
| `code_fence_wrapper`       | error    | Title or body wrapped in triple backticks.                                                       |
| `title_format_missing_tag` | error    | First line does not start with `[TAG]` (uppercase, brackets).                                    |
| `title_format_missing_scope` | warning | Has `[TAG]` but not the `[TAG] scope:` shape.                                                    |
| `title_too_long_hard`      | error    | Title (first line) > 100 chars.                                                                  |
| `title_too_long_soft`      | warning  | Title > 72 chars (GitHub convention).                                                            |
| `empty_title`              | error    | First line is empty after trim.                                                                  |
| `empty_body`               | warning  | Body (everything after the first blank line) is empty after trim.                                |
| `title_equals_body`        | error    | Title and body, after trim, are byte-equal.                                                      |
| `duplicate_line_in_body`   | warning  | Any non-empty, non-separator line appears 2+ times verbatim in the body.                         |
| `template_placeholder`     | error    | Literal `<title>`, `<body>`, `<keypoints>`, `{title}`, `{body}` etc. survived into the message.  |

Out of scope for unit 16 (deferred):

- **Hallucinated paths/symbols**: requires either reading the original
  staged diff (gone after the workspace changes) or persisting a file
  list on the draft. Will revisit when we get to unit 18 (link-commit)
  which is the natural place to also persist the file set.
- **Wrong language detection**: needs heuristics or a small classifier;
  defer until we have enough false positives to justify it.

## Implementation

### `internal/aiengine/verify.go` (new, reusable)

Pure package-level function so the TUI could surface findings later
without re-implementing rules.

```go
type VerifyFinding struct {
    Rule     string `json:"rule"`
    Severity string `json:"severity"` // "error" | "warning"
    Message  string `json:"message"`
    Location string `json:"location,omitempty"` // "title" | "body" | "line:N"
}

type VerifyReport struct {
    HasErrors   bool            `json:"has_errors"`
    HasWarnings bool            `json:"has_warnings"`
    Findings    []VerifyFinding `json:"findings"`
}

func VerifyFinalMessage(finalMessage string) VerifyReport
```

The function parses `finalMessage` into first-line title + remainder
body (splitting on the first blank line), runs each rule, and
collects findings. Order in the returned slice: errors first, then
warnings, then by rule order in the table above.

Residue phrase list lives as a package-private `var aiResiduePhrases =
[]string{...}`. Lowercase + space-normalized comparison so trivial
casing variants are caught. Keep the list **conservative** at first —
better to under-flag than to false-positive on legitimate text. Starter
list:

- `here is the commit message`
- `here's the commit message`
- `here is a commit message`
- `here is your commit message`
- `i made the following changes`
- `as an ai`
- `as a language model`
- `paragraph 1`, `paragraph 2`, `paragraph 3` (literal labels)
- `summary paragraphs`
- `<title>`, `<body>`, `<keypoints>` (template-tag leakage)

Template placeholder rule (`template_placeholder`) is the curly-brace
and angle-bracket variants of the same idea, but matched as a
**word-boundary** regex (`{title}`, `{body}`, `<title>`, etc.) so we
don't false-flag on user text that legitimately contains words like
"title".

### `internal/cli/ai/verify.go` (new)

CLI wrapper:

```go
func runVerify(args []string) int {
    fs := flagSet("ai verify")
    id := fs.Int("id", 0, "Draft or commit id to verify (required).")
    if err := fs.Parse(args); err != nil { ... }
    if *id <= 0 { printErrorJSON("invalid_input", "--id required"); return 2 }

    boot, err := loadBootstrap()
    if err != nil { ... }
    defer boot.db.Close()

    c, err := boot.db.GetCommitByID(*id)
    if err != nil { printErrorJSON("not_found", err.Error()); return 1 }

    final, err := commit.FormatFinalMessage(
        boot.cfg.CommitFormat.TypeFormat, c.Type, c.Scope, c.MessageEN,
    )
    if err != nil { printErrorJSON("incomplete_draft", err.Error()); return 1 }

    report := aiengine.VerifyFinalMessage(final)

    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    _ = enc.Encode(report)
    if report.HasErrors {
        return 4
    }
    return 0
}
```

Exit codes:

- **0** — clean (no findings) or only warnings.
- **1** — bootstrap / not-found errors.
- **2** — usage error (missing or bad flags).
- **4** — at least one finding with severity=error. (Reserved exit
  3 for `ai context --strict` to keep the two gates distinguishable.)

`--strict-warnings` flag (optional bool): when set, warnings also
flip the exit code to 4. Off by default — agents should be allowed
to ship borderline drafts; the warning is informational.

### `internal/cli/ai/ai.go` — wire it up

Add `case "verify": return runVerify(rest)` next to the existing
`context` case. Add a line to the `usage` const:

```
  verify       Deterministic checks on a draft's final_message; exit 4 when errors are found.
```

### `cmd/cli/main.go` — version bump

Bump `version` to `v0.57.0` (minor: new user-visible subcommand).

### `CHANGELOG.md`

Add `## v0.57.0 — 2026-05-27` entry above v0.56.0 describing the new
subcommand. Include a `### Usage` block with an example invocation
and the exit-code semantics.

### `context/progress-tracker.md`

Add a Session Notes entry for unit 16 once shipped. Move unit 16
from In Progress → Completed.

## Dependencies

- Unit 15 (umbrella plan) — only as the parent document. No code dep.

## Verify when done

- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] `make build` succeeds; pre-commit hook runs clean.
- [ ] Smoke tests against a real draft:
  - [ ] Clean draft: `ai verify --id <good>` → exit 0, empty
        `findings`, `has_errors: false`.
  - [ ] Synthesized bad title via `ai edit --id <id> --title "Here is
        the commit message"`: `ai verify --id <id>` → exit 4, finding
        with `rule: ai_residue_phrase`, `severity: error`.
  - [ ] Inject a duplicate body line via `ai edit --id <id> --body
        "Cosa.\n\nCosa."` : exit 0 (only a warning), finding with
        `rule: duplicate_line_in_body`, `severity: warning`.
  - [ ] `--strict-warnings` on the same duplicate-line case → exit 4.
- [ ] `ai verify --id 999999` (nonexistent) → exit 1, JSON
      `{"code": "not_found", ...}` on stderr.
- [ ] `ai verify` (no flag) → exit 2, JSON `{"code": "invalid_input", ...}` on stderr.
