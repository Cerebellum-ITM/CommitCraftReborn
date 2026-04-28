package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/storage"
)

// modelsCacheTTL is the maximum age of the cached catalogue before the
// picker triggers a fresh fetch on next open. 24h matches the cadence at
// which Groq tweaks the free-tier list in practice.
const modelsCacheTTL = 24 * time.Hour

// modelPickerOpenedMsg carries the list the parent will hand to the popup
// constructor. Always emitted from openModelPickerCmd, even when the fetch
// fails — `err` is set so the parent can warn the user.
type modelPickerOpenedMsg struct {
	stage    config.ModelStage
	label    string
	current  string
	models   []storage.CachedModel
	cachedAt time.Time
	err      error
}

// composeStageEntry pairs a human-readable label with the ModelStage id
// for the slots shown on the Compose tab. Order here drives the cursor
// navigation in renderComposePipelineModelsArea.
type composeStageEntry struct {
	label string
	stage config.ModelStage
}

// composePipelineStages returns the stages reachable from the Compose
// pipeline-models row. The CHANGELOG stage is appended when the feature
// is active so the user can swap that model in the same place.
func composePipelineStages(m *Model) []composeStageEntry {
	stages := []composeStageEntry{
		{label: "summary", stage: config.StageChangeAnalyzer},
		{label: "raw commit", stage: config.StageCommitBody},
		{label: "formatted", stage: config.StageCommitTitle},
	}
	if m.changelogActive {
		stages = append(stages, composeStageEntry{
			label: "changelog", stage: config.StageChangelog,
		})
	}
	return stages
}

// openModelPickerCmd loads the cached catalogue (refreshing it from the
// API when stale or empty) and emits modelPickerOpenedMsg so the Update
// loop can build the popup with fresh data.
func openModelPickerCmd(model *Model, stage config.ModelStage, label string) tea.Cmd {
	apiKey := model.globalConfig.TUI.GroqAPIKey
	current := config.CurrentModelForStage(model.globalConfig, stage)
	db := model.db
	return func() tea.Msg {
		cached, fetchedAt, err := db.LoadModelsCache()
		if err != nil {
			return modelPickerOpenedMsg{
				stage: stage, label: label, current: current, err: err,
			}
		}

		if storage.IsModelsCacheStale(fetchedAt, modelsCacheTTL) {
			fresh, ferr := fetchAndCacheModels(db, apiKey)
			if ferr != nil && len(cached) == 0 {
				return modelPickerOpenedMsg{
					stage: stage, label: label, current: current, err: ferr,
				}
			}
			if ferr == nil {
				cached = fresh
				fetchedAt = time.Now()
			}
		}

		return modelPickerOpenedMsg{
			stage:    stage,
			label:    label,
			current:  current,
			models:   cached,
			cachedAt: fetchedAt,
		}
	}
}

// refreshModelPickerCmd forces a fresh fetch (ignoring cache age) and
// emits modelPickerOpenedMsg so the popup can be rebuilt in place.
func refreshModelPickerCmd(model *Model, stage config.ModelStage, label string) tea.Cmd {
	apiKey := model.globalConfig.TUI.GroqAPIKey
	current := config.CurrentModelForStage(model.globalConfig, stage)
	db := model.db
	return func() tea.Msg {
		fresh, err := fetchAndCacheModels(db, apiKey)
		if err != nil {
			cached, fetchedAt, _ := db.LoadModelsCache()
			return modelPickerOpenedMsg{
				stage: stage, label: label, current: current,
				models: cached, cachedAt: fetchedAt, err: err,
			}
		}
		return modelPickerOpenedMsg{
			stage:    stage,
			label:    label,
			current:  current,
			models:   fresh,
			cachedAt: time.Now(),
		}
	}
}

// stageLabelFor returns the human-readable label of stage as it appears
// on the Compose tab; falls back to the bare ModelStage id when the
// stage is not part of the visible row (e.g. release/translate).
func stageLabelFor(m *Model, stage config.ModelStage) string {
	for _, s := range composePipelineStages(m) {
		if s.stage == stage {
			return s.label
		}
	}
	return string(stage)
}

// applyPipelineModelsToStages copies the model ids from globalConfig into
// the pipeline view's per-stage state so the Pipeline tab reflects the
// freshly chosen model without a restart.
func applyPipelineModelsToStages(m *Model) {
	m.pipeline.stages[stageSummary].Model = m.globalConfig.Prompts.ChangeAnalyzerPromptModel
	m.pipeline.stages[stageBody].Model = m.globalConfig.Prompts.CommitBodyGeneratorPromptModel
	m.pipeline.stages[stageTitle].Model = m.globalConfig.Prompts.CommitTitleGeneratorPromptModel
	m.pipeline.stages[stageChangelog].Model = m.globalConfig.Changelog.PromptModel
}

// fetchAndCacheModels hits the Groq /models endpoint, filters the result
// against the curated free-tier allowlist and writes the survivors into
// the SQLite cache.
func fetchAndCacheModels(db *storage.DB, apiKey string) ([]storage.CachedModel, error) {
	models, err := api.ListGroqModels(apiKey)
	if err != nil {
		return nil, err
	}
	out := make([]storage.CachedModel, 0, len(models))
	for _, m := range models {
		if !m.Active {
			continue
		}
		if !config.IsFreeTierChatModel(m.ID) {
			continue
		}
		out = append(out, storage.CachedModel{
			ID:            m.ID,
			OwnedBy:       m.OwnedBy,
			ContextWindow: m.ContextWindow,
		})
	}
	if err := db.SaveModelsCache(out); err != nil {
		return out, err
	}
	return out, nil
}
