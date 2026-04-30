// AI commit-message pipeline (Bubble Tea side).
//
// The actual prompt assembly + Groq calls + changelog refiner now live
// in internal/aiengine so the headless `commitcraft ai …` subcommands
// can run the same pipeline without spinning up a TUI. This file is a
// thin shim that builds aiengine.Deps from the *Model, dispatches the
// per-stage helpers, and copies their results back onto the model so
// the existing per-stage retry commands and pipeline cards keep
// working unchanged.
package tui

import (
	"fmt"
	"strings"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/storage"
)

// engineDeps is a small adapter that converts a TUI *Model into the
// dependency bundle that aiengine functions expect. The pipeline only
// reads from cfg/db/log/pwd, so this stays a pure projection.
func engineDeps(model *Model) aiengine.Deps {
	return aiengine.Deps{
		Cfg: model.globalConfig,
		DB:  model.db,
		Log: model.log,
		Pwd: model.pwd,
	}
}

// recordStageStats copies a CallStats into the per-stage record on the
// pipeline model so the card and the persistence layer can read the
// same numbers. Safe with stats == nil (no-op) so error paths can
// still call it without an extra branch.
func recordStageStats(model *Model, id stageID, stats *api.CallStats) {
	if model == nil || stats == nil {
		return
	}
	if int(id) < 0 || int(id) >= len(model.pipeline.stages) {
		return
	}
	st := &model.pipeline.stages[id]
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

func iaCallCommitBodyGenerator(model *Model, summaryParagraphs string) (string, error) {
	result, stats, err := aiengine.CallCommitBody(
		engineDeps(model), model.commitType, model.commitScope, summaryParagraphs,
	)
	if err != nil {
		return "", fmt.Errorf("stage 2 (commit body): %w", err)
	}
	recordStageStats(model, stageBody, stats)
	return result, nil
}

func iaCallCommitTitleGenerator(model *Model, commitBody string) (string, error) {
	result, stats, err := aiengine.CallCommitTitle(
		engineDeps(model), model.commitType, model.commitScope, commitBody,
	)
	if err != nil {
		return "", fmt.Errorf("stage 3 (commit title): %w", err)
	}
	recordStageStats(model, stageTitle, stats)
	return result, nil
}

// runChangelogRefiner is the optional 4th step. Delegates to aiengine;
// the changelogActive flag is the single source of truth (set by
// pipelineStartFullRun, which already evaluated cfg.Enabled plus the
// dirty-file safeguard) so we re-check it here to preserve that gate.
func runChangelogRefiner(model *Model) {
	model.iaChangelogEntry = ""
	model.iaChangelogMentionLine = ""
	if !model.changelogActive {
		return
	}
	partial := aiengine.Output{
		Body:   model.iaCommitRawOutput,
		Title:  model.iaTitleRawOutput,
		Stages: make([]aiengine.StageStats, 4),
	}
	aiengine.RunChangelogRefiner(engineDeps(model), &partial)
	if partial.Stages[aiengine.StageChangelog].HasStats {
		recordStageStats(
			model,
			stageChangelog,
			stageStatsToCallStats(partial.Stages[aiengine.StageChangelog]),
		)
	}
	model.iaChangelogEntry = partial.ChangelogEntry
	model.iaChangelogMentionLine = partial.ChangelogMentionLine
	model.iaChangelogTargetPath = partial.ChangelogTargetPath
	model.iaChangelogSuggestedVersion = partial.ChangelogSuggestedVersion
}

func assembleCommitMessage(titleText, commitBody string) string {
	return fmt.Sprintf("%s\n\n%s", titleText, commitBody)
}

func assembleOutputCommitMessage(model *Model, c storage.Commit) (string, error) {
	return commit.FormatFinalMessage(
		model.globalConfig.CommitFormat.TypeFormat,
		c.Type,
		c.Scope,
		c.MessageEN,
	)
}

// outputCommitMessageOrFallback wraps assembleOutputCommitMessage for
// TUI call sites that need a non-empty string no matter what. If the
// invariant (tag/scope/message all populated) is violated, log the
// error and degrade to the raw MessageEN so the user still sees
// something instead of a blank screen.
func outputCommitMessageOrFallback(model *Model, c storage.Commit) string {
	msg, err := assembleOutputCommitMessage(model, c)
	if err != nil {
		model.log.Error("assemble output commit message", "error", err, "commit_id", c.ID)
		return c.MessageEN
	}
	return msg
}

// composeFinalCommitMessage builds the user-visible commit message:
// stage 3 title + stage 2 body verbatim, plus the refiner's mention
// line appended at the end when present. Stage 2's stored output is
// never modified — the appended line only lives in the final string.
func composeFinalCommitMessage(model *Model) string {
	return aiengine.ComposeFinalMessage(
		model.iaTitleRawOutput,
		model.iaCommitRawOutput,
		model.iaChangelogMentionLine,
	)
}

// ia_commit_builder runs the full 3-stage pipeline (plus optional
// stage 4) end-to-end. Used by Ctrl+W on the writing-message screen
// and by the full-pipeline retry command.
func ia_commit_builder(model *Model) error {
	deps := engineDeps(model)
	in := aiengine.Input{
		KeyPoints:       model.keyPoints,
		Type:            model.commitType,
		Scope:           model.commitScope,
		ChangelogActive: model.changelogActive,
	}
	if model.usePreloadedDiff {
		in.Diff = model.diffCode
	}

	out, err := aiengine.Run(deps, in)
	if err != nil {
		return err
	}

	model.diffCode = out.Diff
	model.iaSummaryOutput = out.Summary
	model.iaCommitRawOutput = out.Body
	model.iaTitleRawOutput = out.Title
	model.iaChangelogEntry = out.ChangelogEntry
	model.iaChangelogMentionLine = out.ChangelogMentionLine
	model.iaChangelogTargetPath = out.ChangelogTargetPath
	model.iaChangelogSuggestedVersion = out.ChangelogSuggestedVersion
	model.commitTranslate = out.FinalMessage

	for i := range out.Stages {
		if !out.Stages[i].HasStats {
			continue
		}
		recordStageStats(model, stageID(i), stageStatsToCallStats(out.Stages[i]))
	}
	model.log.Debug("Final commit message", "commitTranslate", model.commitTranslate)
	return nil
}

// stageStatsToCallStats projects an aiengine.StageStats back into an
// api.CallStats so the existing recordStageStats helper can ingest it
// without forking the per-field copy. RateLimits.LimitTokens is the
// only field the recorder reads from there — the rest stays zero.
func stageStatsToCallStats(s aiengine.StageStats) *api.CallStats {
	return &api.CallStats{
		PromptTokens:     s.PromptTokens,
		CompletionTokens: s.CompletionTokens,
		TotalTokens:      s.TotalTokens,
		QueueTime:        s.QueueTime,
		PromptTime:       s.PromptTime,
		CompletionTime:   s.CompletionTime,
		TotalTime:        s.APITotalTime,
		RequestID:        s.RequestID,
		Model:            s.StatsModel,
		RateLimits:       api.RateLimits{LimitTokens: s.TPMLimitAtCall},
	}
}

// iaReleaseBuilder remains here because the release flow has its own
// prompt + storage path that doesn't share any logic with the commit
// pipeline. Lives next to its TUI callers for locality.
func iaReleaseBuilder(model *Model) error {
	var input strings.Builder
	delimiter := "--- COMMIT SEPARATOR ---"
	for _, item := range model.selectedCommitList {
		commitContent := fmt.Sprintf(
			"%s\nCommit.Date:%s\nCommit.Title:%s\ncommit.body:%s\n%s\n",
			delimiter,
			item.Date,
			item.Subject,
			item.Body,
			delimiter,
		)
		input.WriteString(commitContent)
	}
	pc := model.globalConfig.Prompts
	model.log.Debug("release ia Input", "input", input)

	iaResponse, _, err := aiengine.SendIaMessage(
		engineDeps(model),
		pc.ReleasePrompt,
		input.String(),
		pc.ReleasePromptModel,
	)
	if err != nil {
		model.log.Error(
			fmt.Sprintf("An error occurred while trying to generate the release output.\n%s", err),
		)
		return fmt.Errorf(
			"An error occurred while trying to generate the release output.\n%s",
			ExtractJSONError(err.Error()),
		)
	}
	model.commitLivePreview = iaResponse
	model.releaseText = iaResponse
	return nil
}
