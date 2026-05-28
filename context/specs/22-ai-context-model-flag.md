# Unit 22: ai-context-model-flag

## Goal

Add `--model <id>` flag to `commitcraft ai context` so an agent (or the
user) can compare whether the staged diff fits inside an alternative model's
context window without editing the config.

## Design

`ai context` already computes the payload size and resolves the context
window from the groq_models_cache. The only missing piece is using a
caller-supplied model ID instead of (or in addition to) the configured
`Prompts.ChangeAnalyzerPromptModel`. The payload estimate itself
(`EstimateChangeAnalyzer`) doesn't depend on the model ID — it measures
chars and uses the system prompt text from config — so the flag only
affects:

1. Which model ID is displayed in the `model` field of the JSON output.
2. Which row is looked up in `groq_models_cache` via `lookupContextWindow`.

Everything else (diff fetch, truncation cap, `--strict` exit code logic)
stays identical.

## Implementation

File: `internal/cli/ai/context.go`

Add a `--model` string flag to the `flagSet` in `runContext`:

```go
modelOverride := fs.String(
    "model",
    "",
    "Override the model ID used for context-window lookup. Must match an entry in the local groq_models_cache.",
)
```

After `fs.Parse`, if `*modelOverride != ""`, use it instead of
`boot.cfg.Prompts.ChangeAnalyzerPromptModel` when:

- Setting `model` in `out` (the JSON struct).
- Passing the ID to `lookupContextWindow`.

The config model (`boot.cfg.Prompts.ChangeAnalyzerPromptModel`) is
still used by `EstimateChangeAnalyzer` (unchanged) because the
payload is the same regardless of which model will consume it — the
size estimate is model-agnostic.

When `--model` names an ID not found in the cache, `lookupContextWindow`
returns 0, `out.ContextWindow` is 0, `out.Fits` and `out.UsagePct` stay
nil — same behavior as today for any unknown model. The output makes the
situation clear via the `context_window: 0` field. No error emitted; the
zero-window case is documented in the skill.

### No changes needed to

- `aiengine/context_estimate.go` — model-agnostic.
- `contextJSON` struct — `model` field already exists.
- `lookupContextWindow` — already takes a modelID string.

## Dependencies

None beyond what unit 1 (`ai context --strict`) already delivered.

## Verify when done

- [ ] `go build ./...` + `go vet ./...` clean.
- [ ] `commitcraft ai context --model meta-llama/llama-4-scout-17b-16e-instruct`
      returns the same JSON as the plain `ai context` (since that's the
      configured model).
- [ ] `commitcraft ai context --model llama-3.3-70b-versatile` returns the
      correct `context_window` and adjusted `usage_pct` / `fits` for that
      model (assumes it's in the local cache from the TUI's model picker).
- [ ] `commitcraft ai context --model nonexistent-model-xyz` returns
      `context_window: 0`, `fits: null`, `usage_pct: null` — no crash, no
      error JSON.
- [ ] `--strict` still exits 3 when `fits: false` regardless of whether
      `--model` is set.
- [ ] Update `context/progress-tracker.md` to mark unit 22 complete.
