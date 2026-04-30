// Package aiengine runs the multi-stage AI commit-message pipeline
// (change analyzer → commit body → commit title → optional changelog
// refiner) against plain inputs/dependencies, with no Bubble Tea state.
//
// The TUI calls Run via a thin shim that maps *tui.Model fields onto
// Input/Deps and copies Output back. The headless `commitcraft ai …`
// subcommands call Run directly. Behavior must match the TUI exactly:
// same prompts, same model selection, same fallbacks.
package aiengine

import (
	"fmt"
	"strings"
	"time"

	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
)

// StageID labels the four pipeline stages so callers (TUI / headless)
// can route StageStats back to their own per-stage UI or telemetry.
type StageID int

const (
	StageSummary   StageID = iota // change analyzer
	StageBody                     // commit body generator
	StageTitle                    // commit title generator
	StageChangelog                // optional changelog refiner
)

// StageStats mirrors api.CallStats plus the model identifier we used,
// so callers can persist ai_calls rows or render the per-stage card.
type StageStats struct {
	ID               StageID
	Model            string
	HasStats         bool
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	QueueTime        time.Duration
	PromptTime       time.Duration
	CompletionTime   time.Duration
	APITotalTime     time.Duration
	RequestID        string
	StatsModel       string
	TPMLimitAtCall   int
}

// Input is the per-run user-supplied data: keypoints + tag + scope, the
// staged diff (left empty to let Run read it via git.GetStagedDiffSummary),
// and a flag controlling the optional changelog refiner stage.
type Input struct {
	KeyPoints       []string
	Type            string
	Scope           string
	Diff            string
	ChangelogActive bool
}

// Deps groups the long-lived services the pipeline depends on. DB is
// optional — only used to persist rate-limit snapshots; nil is OK and
// rate-limit persistence becomes a no-op.
type Deps struct {
	Cfg config.Config
	DB  *storage.DB
	Log *logger.Logger
	Pwd string
}

// Output bundles every text artifact the pipeline produced plus the
// per-stage telemetry. FinalMessage is title + body, with the optional
// changelog mention line appended to the body when present.
type Output struct {
	Summary                   string
	Body                      string
	Title                     string
	ChangelogEntry            string
	ChangelogMentionLine      string
	ChangelogTargetPath       string
	ChangelogSuggestedVersion string
	FinalMessage              string
	Diff                      string
	Stages                    []StageStats
}

// Run executes stages 1–3 and, when in.ChangelogActive is true and the
// project has a CHANGELOG, the refiner stage. Errors from stages 1–3
// abort the run; the refiner is best-effort and logs warnings instead.
func Run(deps Deps, in Input) (Output, error) {
	out := Output{Stages: make([]StageStats, 4)}
	for i := range out.Stages {
		out.Stages[i].ID = StageID(i)
	}

	diff := in.Diff
	if diff == "" {
		var err error
		diff, err = git.GetStagedDiffSummary(deps.Cfg.Prompts.ChangeAnalyzerMaxDiffSize)
		if err != nil {
			return out, fmt.Errorf("failed to get staged diff: %w", err)
		}
	}
	out.Diff = diff

	summary, sumStats, err := CallChangeAnalyzer(deps, in.KeyPoints, diff)
	if err != nil {
		return out, fmt.Errorf("stage 1 (change analyzer): %w", err)
	}
	RecordStage(&out, StageSummary, deps.Cfg.Prompts.ChangeAnalyzerPromptModel, sumStats)
	out.Summary = summary

	body, bodyStats, err := CallCommitBody(deps, in.Type, in.Scope, summary)
	if err != nil {
		return out, fmt.Errorf("stage 2 (commit body): %w", err)
	}
	RecordStage(&out, StageBody, deps.Cfg.Prompts.CommitBodyGeneratorPromptModel, bodyStats)
	out.Body = body

	title, titleStats, err := CallCommitTitle(deps, in.Type, in.Scope, body)
	if err != nil {
		return out, fmt.Errorf("stage 3 (commit title): %w", err)
	}
	RecordStage(&out, StageTitle, deps.Cfg.Prompts.CommitTitleGeneratorPromptModel, titleStats)
	out.Title = title

	if in.ChangelogActive && deps.Cfg.Changelog.Enabled {
		RunChangelogRefiner(deps, &out)
	}

	out.FinalMessage = ComposeFinalMessage(out.Title, out.Body, out.ChangelogMentionLine)
	if deps.Log != nil {
		deps.Log.Debug("Final commit message", "commitTranslate", out.FinalMessage)
	}
	return out, nil
}

// RecordStage copies a CallStats snapshot into out.Stages[id], also
// stamping the per-stage Model name so callers can render it without
// re-deriving from config. Safe with stats == nil (records only the
// model name and clears HasStats).
func RecordStage(out *Output, id StageID, modelName string, stats *api.CallStats) {
	if int(id) < 0 || int(id) >= len(out.Stages) {
		return
	}
	st := &out.Stages[id]
	st.ID = id
	st.Model = modelName
	if stats == nil {
		return
	}
	st.HasStats = true
	st.PromptTokens = stats.PromptTokens
	st.CompletionTokens = stats.CompletionTokens
	st.TotalTokens = stats.TotalTokens
	st.QueueTime = stats.QueueTime
	st.PromptTime = stats.PromptTime
	st.CompletionTime = stats.CompletionTime
	st.APITotalTime = stats.TotalTime
	st.RequestID = stats.RequestID
	st.StatsModel = stats.Model
	st.TPMLimitAtCall = stats.RateLimits.LimitTokens
}

// SendIaMessage is the package-level analogue of the TUI's
// createAndSendIaMessage: identical Groq call, identical rate-limit
// recording. DB persistence of rate-limits is best-effort and silent
// when deps.DB is nil (headless callers without a DB still work).
func SendIaMessage(
	deps Deps,
	systemPrompt, userInput, iaModel string,
) (string, *api.CallStats, error) {
	if iaModel == "" {
		iaModel = "llama-3.1-8b-instant"
	}
	apiKey := deps.Cfg.TUI.GroqAPIKey
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userInput},
	}
	response, stats, err := api.GetGroqChatCompletion(apiKey, iaModel, messages)
	if err != nil {
		return "", stats, fmt.Errorf(
			"call failed (model=%s): %w", iaModel, err,
		)
	}
	if stats != nil {
		api.RecordRateLimits(iaModel, stats.RateLimits)
		persistRateLimits(deps, iaModel, stats.RateLimits)
	}
	return response, stats, nil
}

func persistRateLimits(deps Deps, modelID string, rl api.RateLimits) {
	if deps.DB == nil || modelID == "" {
		return
	}
	row := storage.ModelRateLimits{
		ModelID:           modelID,
		LimitRequests:     rl.LimitRequests,
		RemainingRequests: rl.RemainingRequests,
		ResetRequestsMs:   int(rl.ResetRequests / time.Millisecond),
		LimitTokens:       rl.LimitTokens,
		RemainingTokens:   rl.RemainingTokens,
		ResetTokensMs:     int(rl.ResetTokens / time.Millisecond),
		CapturedAt:        rl.CapturedAt,
		RequestsParsed:    rl.RequestsParsed,
		TokensParsed:      rl.TokensParsed,
		RequestsToday:     rl.RequestsToday,
		RequestsDay:       rl.RequestsDay,
	}
	if err := deps.DB.SaveModelRateLimits(row); err != nil && deps.Log != nil {
		deps.Log.Warn("rate-limit persistence failed", "model", modelID, "error", err)
	}
}

// CallChangeAnalyzer runs stage 1: feeds keypoints + staged diff to the
// change-analyzer prompt and returns the summary text + per-call stats.
func CallChangeAnalyzer(
	deps Deps,
	keyPoints []string,
	gitChanges string,
) (string, *api.CallStats, error) {
	pc := deps.Cfg.Prompts
	developerPoints := stripMentions(strings.Join(keyPoints, "\n"))
	if deps.Log != nil {
		deps.Log.Debug("Change Analyzer input",
			"developerPoints", developerPoints, "gitChanges", gitChanges)
	}
	result, stats, err := SendIaMessage(
		deps,
		pc.ChangeAnalyzerPrompt,
		fmt.Sprintf("DEVELOPER_POINTS:\n%s\nGIT_CHANGES:\n%s", developerPoints, gitChanges),
		pc.ChangeAnalyzerPromptModel,
	)
	if err != nil {
		return "", stats, err
	}
	if deps.Log != nil {
		deps.Log.Debug("Change Analyzer output", "result", result)
	}
	return result, stats, nil
}

// CallCommitBody runs stage 2: feeds tag/scope/summary to the
// commit-body-generator prompt and returns the body text + stats.
func CallCommitBody(
	deps Deps,
	commitType, commitScope, summaryParagraphs string,
) (string, *api.CallStats, error) {
	pc := deps.Cfg.Prompts
	result, stats, err := SendIaMessage(
		deps,
		pc.CommitBodyGeneratorPrompt,
		fmt.Sprintf("TAG:\n%s\nMODULE:\n%s\nSUMMARY_PARAGRAPHS:\n%s",
			commitType, commitScope, summaryParagraphs),
		pc.CommitBodyGeneratorPromptModel,
	)
	if err != nil {
		return "", stats, err
	}
	if deps.Log != nil {
		deps.Log.Debug("Commit Body Generator output", "result", result)
	}
	return result, stats, nil
}

// CallCommitTitle runs stage 3: feeds tag/scope/body to the
// commit-title-generator prompt and returns the trimmed title + stats.
func CallCommitTitle(
	deps Deps,
	commitType, commitScope, commitBody string,
) (string, *api.CallStats, error) {
	pc := deps.Cfg.Prompts
	result, stats, err := SendIaMessage(
		deps,
		pc.CommitTitleGeneratorPrompt,
		fmt.Sprintf("TAG:\n%s\nMODULE:\n%s\nCOMMIT_BODY:\n%s",
			commitType, commitScope, commitBody),
		pc.CommitTitleGeneratorPromptModel,
	)
	if err != nil {
		return "", stats, err
	}
	if deps.Log != nil {
		deps.Log.Debug("Commit Title Generator output", "result", result)
	}
	return strings.TrimSpace(result), stats, nil
}

// ComposeFinalMessage builds the user-visible commit message: title +
// blank line + body, with the optional changelog mention line appended
// to the body when present. Pure string formatting; no I/O.
func ComposeFinalMessage(title, body, mention string) string {
	if mention != "" {
		body = strings.TrimRight(body, " \n\t") + "\n\n" + mention
	}
	return fmt.Sprintf("%s\n\n%s", title, body)
}
