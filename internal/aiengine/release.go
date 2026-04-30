// Release pipeline: 3-stage AI flow that mirrors the commit pipeline
// for GitHub release notes. Stage 1 produces the body from the selected
// commits, stage 2 produces the title from body + commits, stage 3
// refines the assembled note. Same SendIaMessage call shape as the
// commit pipeline so telemetry plugs into the same per-stage cards.
package aiengine

import (
	"fmt"
	"strings"

	"commit_craft_reborn/internal/api"
)

// ReleaseStageID labels the three release pipeline stages so callers
// can route StageStats back to the per-stage UI / persistence.
type ReleaseStageID int

const (
	ReleaseStageBody   ReleaseStageID = iota // body assembler
	ReleaseStageTitle                        // title from body + commits
	ReleaseStageRefine                       // final polish
)

// ReleaseCommit is the per-commit projection the release pipeline reads.
// Kept independent of TUI types so headless callers and tests can build
// it directly.
type ReleaseCommit struct {
	Hash    string
	Date    string
	Subject string
	Body    string
}

// ReleaseInput is the per-run user-supplied data for the release pipeline.
// Mode is informational (release vs. merge) and only enters the log; the
// prompt content does not branch on it.
type ReleaseInput struct {
	Commits []ReleaseCommit
	Mode    string
}

// ReleaseOutput bundles the artifacts each stage produces plus per-stage
// telemetry. Final is the user-visible release note (title + blank line +
// body) emitted by the refine stage.
type ReleaseOutput struct {
	Body   string
	Title  string
	Final  string
	Stages [3]StageStats
}

// RunRelease executes the body → title → refine sequence. Errors abort
// the run and propagate to the caller; partial telemetry already
// recorded survives in out.Stages.
func RunRelease(deps Deps, in ReleaseInput) (ReleaseOutput, error) {
	out := ReleaseOutput{}
	for i := range out.Stages {
		out.Stages[i].ID = StageID(i)
	}

	pc := deps.Cfg.Prompts
	commitsBlob := formatReleaseCommits(in.Commits)

	bodyText, bodyStats, err := SendIaMessage(
		deps,
		pc.ReleaseBodyPrompt,
		commitsBlob,
		pc.ReleaseBodyPromptModel,
	)
	if err != nil {
		return out, fmt.Errorf("stage 1 (release body): %w", err)
	}
	recordReleaseStage(&out, ReleaseStageBody, pc.ReleaseBodyPromptModel, bodyStats)
	out.Body = strings.TrimSpace(bodyText)

	titleInput := fmt.Sprintf("BODY:\n%s\n\nCOMMITS:\n%s", out.Body, commitsBlob)
	titleText, titleStats, err := SendIaMessage(
		deps,
		pc.ReleaseTitlePrompt,
		titleInput,
		pc.ReleaseTitlePromptModel,
	)
	if err != nil {
		return out, fmt.Errorf("stage 2 (release title): %w", err)
	}
	recordReleaseStage(&out, ReleaseStageTitle, pc.ReleaseTitlePromptModel, titleStats)
	out.Title = strings.TrimSpace(titleText)

	refineInput := fmt.Sprintf("TITLE:\n%s\n\nBODY:\n%s", out.Title, out.Body)
	finalText, finalStats, err := SendIaMessage(
		deps,
		pc.ReleaseRefinePrompt,
		refineInput,
		pc.ReleaseRefinePromptModel,
	)
	if err != nil {
		return out, fmt.Errorf("stage 3 (release refine): %w", err)
	}
	recordReleaseStage(&out, ReleaseStageRefine, pc.ReleaseRefinePromptModel, finalStats)
	out.Final = strings.TrimSpace(finalText)

	if deps.Log != nil {
		deps.Log.Debug("Final release note", "mode", in.Mode, "release", out.Final)
	}
	return out, nil
}

// formatReleaseCommits builds the "--- COMMIT SEPARATOR ---" blob the
// release prompts expect. Matches the legacy single-stage layout so
// users with custom prompts can reuse them as the stage 1 body prompt.
func formatReleaseCommits(commits []ReleaseCommit) string {
	const sep = "--- COMMIT SEPARATOR ---"
	var b strings.Builder
	for _, c := range commits {
		fmt.Fprintf(&b, "%s\nCommit.Date:%s\nCommit.Title:%s\ncommit.body:%s\n%s\n",
			sep, c.Date, c.Subject, c.Body, sep)
	}
	return b.String()
}

func recordReleaseStage(
	out *ReleaseOutput,
	id ReleaseStageID,
	modelName string,
	stats *api.CallStats,
) {
	if int(id) < 0 || int(id) >= len(out.Stages) {
		return
	}
	st := &out.Stages[id]
	st.ID = StageID(id)
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
