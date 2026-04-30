package ai

import (
	"fmt"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/storage"
)

// runStagePartial re-runs a subset of the pipeline against an existing
// commit/draft. The cascade matches the TUI's per-stage retry shortcuts:
//
//   - body      → body + title + (refiner if active)
//   - title     → title + (refiner if active)
//   - changelog → only the refiner
//
// The previous run's outputs (Summary, plus whatever upstream stages we
// don't re-run) are reused so the final message stays coherent. The
// returned Output preserves prior ai_calls telemetry for the stages
// that didn't run, by pre-loading from the DB and only overwriting the
// stages we actually re-execute.
func runStagePartial(
	deps aiengine.Deps,
	c storage.Commit,
	stage string,
	changelogActive bool,
) (aiengine.Output, error) {
	out := aiengine.Output{
		Summary:        c.IaSummary,
		Body:           c.IaCommitRaw,
		Title:          c.IaTitle,
		ChangelogEntry: c.IaChangelog,
		Diff:           c.Diff_code,
		Stages:         loadStagesForCommit(deps.DB, c.ID),
	}
	if len(out.Stages) < 4 {
		// Defensive: ensure the four-slot layout the engine expects.
		grown := make([]aiengine.StageStats, 4)
		copy(grown, out.Stages)
		for i := range grown {
			grown[i].ID = aiengine.StageID(i)
		}
		out.Stages = grown
	}

	switch stage {
	case "body":
		body, stats, err := aiengine.CallCommitBody(deps, c.Type, c.Scope, c.IaSummary)
		if err != nil {
			return out, fmt.Errorf("stage body: %w", err)
		}
		aiengine.RecordStage(
			&out,
			aiengine.StageBody,
			deps.Cfg.Prompts.CommitBodyGeneratorPromptModel,
			stats,
		)
		out.Body = body
		fallthrough
	case "title":
		title, stats, err := aiengine.CallCommitTitle(deps, c.Type, c.Scope, out.Body)
		if err != nil {
			return out, fmt.Errorf("stage title: %w", err)
		}
		aiengine.RecordStage(
			&out,
			aiengine.StageTitle,
			deps.Cfg.Prompts.CommitTitleGeneratorPromptModel,
			stats,
		)
		out.Title = title
		fallthrough
	case "changelog":
		if changelogActive && deps.Cfg.Changelog.Enabled {
			aiengine.RunChangelogRefiner(deps, &out)
		} else {
			// Caller asked to skip the refiner — clear any stale fields
			// so the final message doesn't carry an old mention line.
			out.ChangelogEntry = ""
			out.ChangelogMentionLine = ""
		}
	default:
		return out, fmt.Errorf("unsupported stage %q", stage)
	}

	out.FinalMessage = aiengine.ComposeFinalMessage(out.Title, out.Body, out.ChangelogMentionLine)
	return out, nil
}
