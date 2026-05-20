package tui

import (
	tea "charm.land/bubbletea/v2"

	"commit_craft_reborn/internal/git"
)

func callIaCommitBuilderCmd(model *Model) tea.Cmd {
	return func() tea.Msg {
		err := ia_commit_builder(model)
		return IaCommitBuilderResultMsg{Err: err}
	}
}

func callIaSummaryCmd(model *Model) tea.Cmd {
	return func() tea.Msg {
		err := ia_commit_builder(model)
		return IaSummaryResultMsg{Err: err}
	}
}

func callIaCommitBuilderStage2Cmd(model *Model) tea.Cmd {
	return func() tea.Msg {
		commitBody, err := iaCallCommitBodyGenerator(model, model.iaSummaryOutput)
		if err != nil {
			return IaCommitRawResultMsg{Err: err}
		}
		model.iaCommitRawOutput = commitBody

		titleText, err := iaCallCommitTitleGenerator(model, commitBody)
		if err != nil {
			return IaCommitRawResultMsg{Err: err}
		}
		model.iaTitleRawOutput = titleText
		runChangelogRefiner(model)
		model.commitTranslate = composeFinalCommitMessage(model)
		return IaCommitRawResultMsg{Err: nil}
	}
}

func callIaOutputFormatCmd(model *Model) tea.Cmd {
	return func() tea.Msg {
		titleText, err := iaCallCommitTitleGenerator(model, model.iaCommitRawOutput)
		if err != nil {
			return IaOutputFormatResultMsg{Err: err}
		}
		model.iaTitleRawOutput = titleText
		runChangelogRefiner(model)
		model.commitTranslate = composeFinalCommitMessage(model)
		return IaOutputFormatResultMsg{Err: nil}
	}
}

// callIaChangelogOnlyCmd re-runs only the changelog refiner using the
// currently stored stage 2 / stage 3 outputs. Used by the stage 4 retry
// shortcut so the user can iterate on the entry without re-spending tokens
// on the upstream stages.
func callIaChangelogOnlyCmd(model *Model) tea.Cmd {
	return func() tea.Msg {
		runChangelogRefiner(model)
		model.commitTranslate = composeFinalCommitMessage(model)
		return IaChangelogResultMsg{Err: nil}
	}
}

func callIaReleaseBuilderCmd(model *Model) tea.Cmd {
	return func() tea.Msg {
		body, title, final, err := iaReleaseBuilder(model)
		return IaReleaseBuilderResultMsg{
			Err:   err,
			From:  stageSummary,
			Body:  body,
			Title: title,
			Final: final,
		}
	}
}

// callIaReleaseCascadeCmd kicks off a partial release pipeline run from
// `from` (stageSummary / stageBody / stageTitle). Used by the
// per-stage retry shortcuts in updateReleaseBuildingText.
func callIaReleaseCascadeCmd(model *Model, from stageID) tea.Cmd {
	return func() tea.Msg {
		body, title, final, err := iaReleaseCascade(model, from)
		return IaReleaseBuilderResultMsg{
			Err:   err,
			From:  from,
			Body:  body,
			Title: title,
			Final: final,
		}
	}
}

// openReleaseConfigPopup builds the multi-field release configuration
// popup, pre-filling the inputs with whatever is already in the live
// ReleaseConfig plus auto-detected defaults from the workspace. The
// `autoOpen` flag flows through to releaseConfigSavedMsg so the
// upload-chain handler can resume the upload after the save.
func openReleaseConfigPopup(model *Model, autoOpen bool) releaseConfigPopupModel {
	// Sized 20 % larger than the v0.53.0 release-config popup: the
	// auto_build / build_tool / build_target additions plus the
	// upcoming list-picker affordances need the extra breathing
	// room. Floors / ceilings shift accordingly.
	w := model.width * 3 / 5
	if w < 72 {
		w = 72
	}
	if w > 108 {
		w = 108
	}
	h := model.height * 9 / 10
	if h < 29 {
		h = 29
	}
	current := ReleaseConfigSnapshot{
		Repository:  model.globalConfig.ReleaseConfig.Repository,
		Branch:      model.globalConfig.ReleaseConfig.Branch,
		Version:     model.globalConfig.ReleaseConfig.Version,
		AssetsPath:  model.globalConfig.ReleaseConfig.BinaryAssetsPath,
		AutoBuild:   model.globalConfig.ReleaseConfig.AutoBuild,
		BuildTool:   model.globalConfig.ReleaseConfig.BuildTool,
		BuildTarget: model.globalConfig.ReleaseConfig.BuildTarget,
	}
	detected := DetectRelease(model.pwd)
	tools := ListBuildTools()
	targets := ListMakefileTargets(model.pwd)
	return newReleaseConfigPopup(
		w, h, model.Theme, current, detected, autoOpen, tools, targets,
	)
}

// openChangelogConfigPopup mirrors openReleaseConfigPopup for the
// ChangelogConfig. The popup pre-fills the inputs with whatever is
// already in the live config and runs DetectChangelog for the
// auto-detected defaults panel.
func openChangelogConfigPopup(model *Model) changelogConfigPopupModel {
	// Same +20 % sizing as openReleaseConfigPopup so the two popups
	// look balanced when the user pings back and forth via the
	// command palette.
	w := model.width * 3 / 5
	if w < 72 {
		w = 72
	}
	if w > 108 {
		w = 108
	}
	h := model.height * 9 / 10
	if h < 27 {
		h = 27
	}
	current := ChangelogConfigSnapshot{
		Enabled:      model.globalConfig.Changelog.Enabled,
		Path:         model.globalConfig.Changelog.Path,
		BumpStrategy: model.globalConfig.Changelog.BumpStrategy,
		PromptFile:   model.globalConfig.Changelog.PromptFile,
		PromptModel:  model.globalConfig.Changelog.PromptModel,
	}
	detected := DetectChangelog(model.pwd)
	return newChangelogConfigPopup(w, h, model.Theme, current, detected)
}

func openVersionEditor(model *Model) versionPopupModel {
	tag, err := git.GetLastGitTag()
	if err != nil {
		model.log.Error("Error reading last git tag", "error", err)
	}
	w := model.width / 2
	if w < 50 {
		w = 60
	}
	h := model.height / 2
	if h < 12 {
		h = 14
	}
	return newVersionPopup(
		w, h,
		model.globalConfig.ReleaseConfig.Version,
		tag,
		model.Theme,
	)
}

// setupCommitReword configures the model so the user can rewrite the message
// of model.pendingRewordHash through the regular commit AI pipeline (type →
// scope → AI generation). Triggered from the startup chooser popup.
