package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"
)

func updateRewordSelectCommit(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	model.releaseCommitList, cmd = model.releaseCommitList.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Up, model.keys.Down):
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
		case key.Matches(msg, model.keys.Enter):
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				if model.commitAndReword {
					model.RewordHash = item.Hash
					model.syncRewordIndicator()
					model.commitAndReword = false
					model.usePreloadedDiff = true
					diffCode, err := git.GetCommitDiffSummary(item.Hash, model.globalConfig.Prompts.ChangeAnalyzerMaxDiffSize)
					if err != nil {
						model.log.Error("Error getting commit diff", "error", err)
					}
					model.diffCode = diffCode
					if commitGitData, gErr := git.GetCommitGitStatusData(item.Hash); gErr == nil {
						model.gitStatusData = commitGitData
						model.currentUpdateFileListFn(model.pwd, &model.fileList, model.gitStatusData)
						model.scopeDataStale = false
						model.syncScopeStaleIndicator()
					} else {
						model.log.Error("Error getting commit git status data", "error", gErr)
					}
					loadPipelineFilesFromDb(model, diffCode)
					model.currentCommit = storage.Commit{}
					model.keyPoints = nil
					model.commitTranslate = ""
					model.iaSummaryOutput = ""
					model.iaCommitRawOutput = ""
					model.iaTitleRawOutput = ""
					model.activeTab = 0
					model.activePipelineStage = 0
					model.state = stateWritingMessage
					model.focusedElement = focusComposeSummary
					model.keys = writingMessageKeys()
					model.WritingStatusBar.Content = fmt.Sprintf("Reword %s · compose key points and run AI", item.Hash[:7])
					model.WritingStatusBar.Level = statusbar.LevelInfo
					if cmd := model.commitsKeysInput.Focus(); cmd != nil {
						return model, cmd
					}
					return model, nil
				}
				model.RewordHash = item.Hash
				model.FinalMessage = assembleOutputCommitMessage(model, model.currentCommit)
				return quitWithAutodraft(model)
			}
		case key.Matches(msg, model.keys.Esc):
			model.state = stateChoosingCommit
			model.keys = mainListKeys()
			model.WritingStatusBar.Content = fmt.Sprintf(
				"choose, create, or edit a commit ::: %s",
				model.Theme.AppStyles().
					Base.Foreground(model.Theme.Tertiary).
					SetString(model.mainList.Title),
			)
			model.WritingStatusBar.Level = statusbar.LevelInfo
			return model, nil
		}
	}

	return model, cmd
}

// openVersionEditor builds the release-version editor popup with the latest
// git tag as the bump base. Used both by the Ctrl+V global shortcut and as a
// pre-flight step before publishing a GitHub release, so the user always
// reviews the tag the upload is about to use.

func setupCommitReword(model *Model) (tea.Model, tea.Cmd) {
	hash := model.pendingRewordHash
	model.pendingRewordHash = ""
	model.RewordHash = hash
	model.syncRewordIndicator()
	model.usePreloadedDiff = true

	diff, dErr := git.GetCommitDiffSummary(
		hash,
		model.globalConfig.Prompts.ChangeAnalyzerMaxDiffSize,
	)
	if dErr != nil {
		model.log.Error("Error getting commit diff for reword", "error", dErr)
	}
	model.diffCode = diff

	if commitGitData, gErr := git.GetCommitGitStatusData(hash); gErr == nil {
		model.gitStatusData = commitGitData
		model.currentUpdateFileListFn(model.pwd, &model.fileList, model.gitStatusData)
		model.scopeDataStale = false
		model.syncScopeStaleIndicator()
	} else {
		model.log.Error("Error getting commit git status data for reword", "error", gErr)
	}

	loadPipelineFilesFromDb(model, diff)
	model.currentCommit = storage.Commit{}
	model.keyPoints = nil
	model.commitTranslate = ""
	model.iaSummaryOutput = ""
	model.iaCommitRawOutput = ""
	model.iaTitleRawOutput = ""
	model.activeTab = 0
	model.activePipelineStage = 0

	model.state = stateWritingMessage
	model.focusedElement = focusComposeSummary
	model.keys = writingMessageKeys()
	model.WritingStatusBar.Content = fmt.Sprintf(
		"Reword %s · compose key points and run AI",
		hash[:7],
	)
	model.WritingStatusBar.Level = statusbar.LevelInfo
	if cmd := model.commitsKeysInput.Focus(); cmd != nil {
		return model, cmd
	}
	return model, nil
}

// setupReleaseReword routes the user from the -w startup popup into the
// regular release creation flow (-r). The pending hash is discarded because
// release creation is multi-commit: the user picks the commits to include in
// the release notes from stateReleaseMainMenu / stateReleaseChoosingCommits.
func setupReleaseReword(model *Model) (tea.Model, tea.Cmd) {
	model.pendingRewordHash = ""
	model.RewordHash = ""
	model.syncRewordIndicator()

	model.AppMode = ReleaseMode
	model.state = stateReleaseMainMenu
	model.keys = releaseMainListKeys()
	loadCmd := enterReleaseHistoryLoading(model)
	model.WritingStatusBar.Content = fmt.Sprintf(
		"choose, create, or edit a release ::: %s",
		model.Theme.AppStyles().
			Base.Foreground(model.Theme.Tertiary).
			SetString(model.releaseMainList.Title),
	)
	model.WritingStatusBar.Level = statusbar.LevelInfo
	return model, loadCmd
}
