package aiengine

import (
	"fmt"
	"strings"

	"commit_craft_reborn/internal/config"
)

// ChangeAnalyzerEstimate is the pre-flight payload measurement for stage 1
// of the pipeline. It is computed without calling Groq and without touching
// the DB so the CLI and (future) TUI indicator can probe context-window
// usage cheaply before triggering CallChangeAnalyzer.
//
// EstTokens uses the standard chars/4 heuristic. It is intentionally an
// approximation — typically within ~15% of the real Groq tokenizer for
// English + code, more than enough to detect imminent context overflow.
type ChangeAnalyzerEstimate struct {
	SystemPromptChars int
	UserInputChars    int
	TotalChars        int
	EstTokens         int
}

// EstimateChangeAnalyzer reproduces the exact payload that CallChangeAnalyzer
// would send: the change-analyzer system prompt plus the
// "DEVELOPER_POINTS:\n…\nGIT_CHANGES:\n…" user input. Keypoints are joined
// with newlines and passed through stripMentions so the measurement matches
// what the real call would emit byte-for-byte.
func EstimateChangeAnalyzer(
	cfg config.Config,
	gitChanges string,
	keyPoints []string,
) ChangeAnalyzerEstimate {
	systemPrompt := cfg.Prompts.ChangeAnalyzerPrompt
	developerPoints := stripMentions(strings.Join(keyPoints, "\n"))
	userInput := fmt.Sprintf(
		"DEVELOPER_POINTS:\n%s\nGIT_CHANGES:\n%s",
		developerPoints,
		gitChanges,
	)

	sysChars := len(systemPrompt)
	usrChars := len(userInput)
	total := sysChars + usrChars
	estTokens := (total + 3) / 4 // ceil(total/4)

	return ChangeAnalyzerEstimate{
		SystemPromptChars: sysChars,
		UserInputChars:    usrChars,
		TotalChars:        total,
		EstTokens:         estTokens,
	}
}
