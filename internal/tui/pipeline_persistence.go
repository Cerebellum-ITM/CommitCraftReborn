package tui

import (
	"time"

	"commit_craft_reborn/internal/storage"
)

// stageDBLabel maps a stageID into the short tag stored in the
// ai_calls.stage column. Kept narrow on purpose: only the stages we
// actually call through the pipeline pipeline produce telemetry.
var stageDBLabel = map[stageID]string{
	stageSummary:   "summary",
	stageBody:      "body",
	stageTitle:     "title",
	stageChangelog: "changelog",
}

// stageIDFromDBLabel inverts stageDBLabel for the reload path.
func stageIDFromDBLabel(label string) (stageID, bool) {
	for id, lbl := range stageDBLabel {
		if lbl == label {
			return id, true
		}
	}
	return 0, false
}

// persistPipelineAICalls flushes the per-stage telemetry currently held in
// model.pipeline.stages to the ai_calls table for commitID. Existing rows
// for the commit are removed first so a draft saved repeatedly never
// accumulates orphan stage records.
func persistPipelineAICalls(model *Model, commitID int) {
	if commitID <= 0 || model == nil || model.db == nil {
		return
	}
	if err := model.db.DeleteAICallsByCommitID(commitID); err != nil {
		model.log.Warn("ai_calls cleanup failed", "commit_id", commitID, "error", err)
		return
	}
	for i := range model.pipeline.stages {
		st := &model.pipeline.stages[i]
		if !st.HasStats {
			continue
		}
		label, ok := stageDBLabel[st.ID]
		if !ok {
			continue
		}
		modelName := st.StatsModel
		if modelName == "" {
			modelName = st.Model
		}
		call := storage.AICall{
			CommitID:         commitID,
			Stage:            label,
			Model:            modelName,
			PromptTokens:     st.PromptTokens,
			CompletionTokens: st.CompletionTokens,
			TotalTokens:      st.TotalTokens,
			QueueTimeMs:      durationToMs(st.QueueTime),
			PromptTimeMs:     durationToMs(st.PromptTime),
			CompletionTimeMs: durationToMs(st.CompletionTime),
			TotalTimeMs:      durationToMs(stageDisplayDuration(st)),
			RequestID:        st.RequestID,
		}
		if _, err := model.db.CreateAICall(call); err != nil {
			model.log.Warn(
				"ai_calls insert failed",
				"commit_id", commitID, "stage", label, "error", err,
			)
		}
	}
}

// loadPipelineAICalls rehydrates model.pipeline.stages with telemetry
// previously persisted for commitID. Stages without a stored row keep
// their zero values so the UI shows them as empty.
func loadPipelineAICalls(model *Model, commitID int) {
	if commitID <= 0 || model == nil || model.db == nil {
		return
	}
	calls, err := model.db.GetAICallsByCommitID(commitID)
	if err != nil {
		model.log.Warn("ai_calls load failed", "commit_id", commitID, "error", err)
		return
	}
	for _, c := range calls {
		id, ok := stageIDFromDBLabel(c.Stage)
		if !ok {
			continue
		}
		if int(id) < 0 || int(id) >= len(model.pipeline.stages) {
			continue
		}
		st := &model.pipeline.stages[id]
		st.HasStats = true
		st.StatsModel = c.Model
		st.PromptTokens = c.PromptTokens
		st.CompletionTokens = c.CompletionTokens
		st.TotalTokens = c.TotalTokens
		st.QueueTime = msToDuration(c.QueueTimeMs)
		st.PromptTime = msToDuration(c.PromptTimeMs)
		st.CompletionTime = msToDuration(c.CompletionTimeMs)
		st.APITotalTime = msToDuration(c.TotalTimeMs)
		st.RequestID = c.RequestID
		// The reloaded stage is conceptually "done" — it produced output
		// previously — so mirror the wall-clock latency from the stored
		// total_time_ms so renderStageStatsLine has a duration to print
		// and flip the lifecycle status so the card reads as completed.
		if st.Latency == 0 {
			st.Latency = st.APITotalTime
		}
		st.Status = statusDone
		st.Progress = 1
	}
}

// stageDisplayDuration picks the duration we want persisted as the canonical
// total time for a stage row. Wall-clock Latency wins when non-zero (it
// includes network overhead the user actually felt), with the API-side
// total_time as fallback.
func stageDisplayDuration(st *pipelineStage) time.Duration {
	if st.Latency > 0 {
		return st.Latency
	}
	return st.APITotalTime
}

func durationToMs(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(d / time.Millisecond)
}

func msToDuration(ms int) time.Duration {
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}
