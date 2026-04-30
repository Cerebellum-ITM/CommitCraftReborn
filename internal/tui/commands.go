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
		err := iaReleaseBuilder(model)
		return IaReleaseBuilderResultMsg{Err: err}
	}
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
