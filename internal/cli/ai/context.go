package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"os"

	"commit_craft_reborn/internal/aiengine"
)

// contextJSON is the wire shape for `commitcraft ai context`. ContextWindow
// is 0 and Fits is nil when the configured change-analyzer model is not
// present in the groq_models_cache table — the caller (typically an agent)
// can then decide whether to refresh the cache or skip the check.
type contextJSON struct {
	Model             string   `json:"model"`
	ContextWindow     int      `json:"context_window"`
	SystemPromptChars int      `json:"system_prompt_chars"`
	UserInputChars    int      `json:"user_input_chars"`
	TotalChars        int      `json:"total_chars"`
	EstTokens         int      `json:"est_tokens"`
	UsagePct          *float64 `json:"usage_pct"`
	Fits              *bool    `json:"fits"`
	DiffTruncated     bool     `json:"diff_truncated"`
	DiffMaxBytes      int      `json:"diff_max_bytes"`
}

// runContext computes the pre-flight payload size for stage 1 (Change
// Analyzer) against the currently staged diff and the configured model,
// and prints the JSON breakdown on stdout. It mirrors the diff fetch and
// cap that the real pipeline applies, so callers see the same truncation
// behaviour they would hit on a live `ai generate`.
func runContext(args []string) int {
	fs := flagSet("ai context")
	strict := fs.Bool(
		"strict",
		false,
		"Exit with code 3 when the estimated payload exceeds the model's context window. Off by default so the JSON is always emitted.",
	)
	modelOverride := fs.String(
		"model",
		"",
		"Override the model ID used for context-window lookup. Must match an entry in the local groq_models_cache.",
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}

	boot, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer boot.db.Close()

	maxBytes := boot.cfg.Prompts.ChangeAnalyzerMaxDiffSize
	diff, err := validateAndStageDiff(maxBytes)
	if err != nil {
		printErrorJSON("no_staged_diff", err.Error())
		return 1
	}

	est := aiengine.EstimateChangeAnalyzer(boot.cfg, diff, nil)

	model := boot.cfg.Prompts.ChangeAnalyzerPromptModel
	if *modelOverride != "" {
		model = *modelOverride
	}
	contextWindow := lookupContextWindow(boot, model)

	out := contextJSON{
		Model:             model,
		ContextWindow:     contextWindow,
		SystemPromptChars: est.SystemPromptChars,
		UserInputChars:    est.UserInputChars,
		TotalChars:        est.TotalChars,
		EstTokens:         est.EstTokens,
		DiffTruncated:     maxBytes > 0 && len(diff) >= maxBytes,
		DiffMaxBytes:      maxBytes,
	}

	if contextWindow > 0 {
		pct := float64(est.EstTokens) / float64(contextWindow) * 100
		fits := est.EstTokens <= contextWindow
		out.UsagePct = &pct
		out.Fits = &fits
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
	if *strict && out.Fits != nil && !*out.Fits {
		return 3
	}
	return 0
}

// lookupContextWindow returns the cached context window for modelID, or 0
// when the model is not in the local groq_models_cache. We don't fetch
// fresh from Groq here — that's a separate, network-bound operation; this
// command stays offline so it's safe to run repeatedly from an agent loop.
func lookupContextWindow(boot *bootstrap, modelID string) int {
	if boot.db == nil || modelID == "" {
		return 0
	}
	models, _, err := boot.db.LoadModelsCache()
	if err != nil {
		return 0
	}
	for _, m := range models {
		if m.ID == modelID {
			return m.ContextWindow
		}
	}
	return 0
}
