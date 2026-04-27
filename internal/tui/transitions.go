package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"commit_craft_reborn/internal/changelog"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"
)

func (model *Model) cancelProcess(state appState) (tea.Model, tea.Cmd) {
	var statusBarMessage string
	statusBarLevel := statusbar.LevelInfo

	switch state {
	case stateChoosingCommit:
		statusBarMessage = fmt.Sprintf(
			"choose, create, or edit a commit ::: %s",
			model.Theme.AppStyles().
				Base.Foreground(model.Theme.Tertiary).
				SetString(model.mainList.Title),
		)
		model.commitMsg = ""
		model.commitType = ""
		model.commitTranslate = ""
		model.resetScopes()
		model.keyPoints = nil
		model.iaSummaryOutput = ""
		model.iaCommitRawOutput = ""
		model.iaTitleRawOutput = ""
		model.activeTab = 0
		model.activePipelineStage = 0
		model.RewordHash = ""
		model.commitAndReword = false
		model.useDbCommmit = false
		model.scopeDataStale = false
		model.syncScopeStaleIndicator()
		if gitData, gErr := git.GetAllGitStatusData(); gErr == nil {
			model.gitStatusData = gitData
			model.currentUpdateFileListFn(model.pwd, &model.fileList, model.gitStatusData)
		}
		model.keys = mainListKeys()
	case stateChoosingType:
		statusBarMessage = "Select a prefix for the commit"
		model.commitType = ""
		model.resetScopes()
		model.keys = listKeys()
	case stateChoosingScope:
		statusBarMessage = "choose a file or folder for your commit"
		model.resetScopes()
		model.keys = fileListKeys()
		model.commitsKeysInput.Blur()
	case stateReleaseMainMenu:
		model.keys = releaseMainListKeys()
		statusBarMessage = fmt.Sprintf(
			"choose, create, or edit a release ::: %s",
			model.Theme.AppStyles().
				Base.Foreground(model.Theme.Tertiary).
				SetString(model.mainList.Title),
		)
	}
	model.state = state
	model.WritingStatusBar.Content = statusBarMessage
	model.WritingStatusBar.Level = statusBarLevel
	return model, nil
}

func createCommit(model *Model) (tea.Model, tea.Cmd) {
	if v := model.commitsKeysInput.Value(); v != "" {
		model.keyPoints = append(model.keyPoints, v)
		model.commitsKeysInput.SetValue("")
	}
	model.currentCommit.Type = model.commitType
	model.currentCommit.Scope = model.commitScope
	model.currentCommit.KeyPoints = model.keyPoints
	model.currentCommit.MessageEN = model.commitTranslate
	model.currentCommit.Workspace = model.pwd
	model.currentCommit.Diff_code = model.diffCode
	model.currentCommit.IaSummary = model.iaSummaryOutput
	model.currentCommit.IaCommitRaw = model.iaCommitRawOutput
	model.currentCommit.IaTitle = model.iaTitleRawOutput
	model.currentCommit.CreatedAt = time.Now()

	// Reword flows rewrite an existing commit's message; staging a new
	// CHANGELOG file in that path either changes the historical commit's
	// scope unexpectedly (HEAD amend) or breaks the interactive rebase used
	// for older commits. The "commit and reword" mode is the exception —
	// it produces a brand-new commit, so the changelog write is welcome.
	skipChangelogWrite := model.RewordHash != "" && !model.commitAndReword

	if model.iaChangelogEntry != "" && model.iaChangelogTargetPath != "" && !skipChangelogWrite {
		if err := changelog.Prepend(model.iaChangelogTargetPath, model.iaChangelogEntry); err != nil {
			model.log.Error("Failed to update CHANGELOG", "error", err)
			return model, model.WritingStatusBar.ShowMessageForDuration(
				fmt.Sprintf("CHANGELOG update failed: %s", err),
				statusbar.LevelError,
				3*time.Second,
			)
		}
		if err := git.StageFile(model.iaChangelogTargetPath); err != nil {
			model.log.Error("Failed to stage CHANGELOG", "error", err)
			return model, model.WritingStatusBar.ShowMessageForDuration(
				fmt.Sprintf("CHANGELOG staged update failed: %s", err),
				statusbar.LevelError,
				3*time.Second,
			)
		}
	}

	var err error
	if model.currentCommit.ID != 0 {
		err = model.db.FinalizeCommit(model.currentCommit)
	} else {
		err = model.db.CreateCommit(model.currentCommit)
	}

	if err != nil {
		model.log.Error("Error saving commit", "error", err)
		model.err = err
		return model, tea.Quit
	}

	UpdateCommitList(model.pwd, model.db, model.log, &model.mainList, commitDb)
	// The CHANGELOG file likely changed (we just prepended an entry) — refresh
	// the indicator so the pill flips between auto/passive on the next render.
	model.refreshChangelogState()
	model.state = stateChoosingCommit
	model.keys = mainListKeys()
	model.WritingStatusBar.Content = fmt.Sprintf(
		"choose, create, or edit a commit ::: %s",
		model.Theme.AppStyles().
			Base.Foreground(model.Theme.Tertiary).
			SetString(model.mainList.Title),
	)
	cmd := model.WritingStatusBar.ShowMessageForDuration(
		"Record created in the db successfully",
		statusbar.LevelSuccess,
		2*time.Second,
	)
	return model, cmd
}

func createRelease(model *Model) (tea.Model, tea.Cmd) {
	var commitList []string

	parts := strings.SplitN(model.releaseText, "\n", 2)

	for _, item := range model.selectedCommitList {
		commitList = append(commitList, item.Hash)
	}

	newRelease := storage.Release{
		ID:         0,
		Type:       model.releaseType,
		Title:      strings.TrimSpace(parts[0]),
		Body:       strings.TrimSpace(parts[1]),
		Branch:     model.releaseBranch,
		Version:    model.globalConfig.ReleaseConfig.Version,
		CommitList: strings.Join(commitList, ","),
		Workspace:  model.pwd,
		CreatedAt:  time.Now(),
	}

	err := model.db.CreateRelease(newRelease)
	if err != nil {
		model.log.Error("Error creating the release", "error", err)
		model.err = err
		return model, tea.Quit
	}

	UpdateCommitList(model.pwd, model.db, model.log, &model.releaseMainList, releaseDb)
	model.state = stateReleaseMainMenu
	model.keys = mainListKeys()
	cmd := model.WritingStatusBar.ShowMessageForDuration(
		"Record created in the db successfully",
		statusbar.LevelSuccess,
		2*time.Second,
	)
	return model, cmd
}

func switchFocusElement(model *Model) tea.Cmd {
	if model.state == stateWritingMessage {
		return cycleComposeFocus(model, true)
	}
	switch model.focusedElement {
	case focusListElement:
		model.keys = viewPortKeys()
		model.focusedElement = focusViewportElement
		if model.state == stateReleaseChoosingCommits {
			model.keys.NextViewPort.SetEnabled(true)
		}
	case focusViewportElement:
		model.keys = releaseKeys()
		model.focusedElement = focusListElement
		if model.state == stateReleaseChoosingCommits {
			model.keys.NextViewPort.SetEnabled(false)
		}
	case focusMsgInput:
		model.focusedElement = focusAIResponse
		model.commitsKeysInput.Blur()
	case focusAIResponse:
		model.focusedElement = focusMsgInput
		return model.commitsKeysInput.Focus()
	}
	return nil
}

// composeFocusOrder is the canonical Tab ordering for the compose view's
// 6 focusable sections.
var composeFocusOrder = []focusableElement{
	focusComposeType,
	focusComposeScope,
	focusComposeSummary,
	focusComposeKeypoints,
	focusComposePipelineModels,
	focusComposeAISuggestion,
}

// cycleComposeFocus advances focus to the next/previous compose section.
// It also (un)focuses the underlying textarea so typed input only reaches
// the summary section.
func cycleComposeFocus(model *Model, forward bool) tea.Cmd {
	cur := -1
	for i, f := range composeFocusOrder {
		if model.focusedElement == f {
			cur = i
			break
		}
	}
	if cur == -1 {
		if forward {
			model.focusedElement = composeFocusOrder[0]
		} else {
			model.focusedElement = composeFocusOrder[len(composeFocusOrder)-1]
		}
	} else {
		if forward {
			cur = (cur + 1) % len(composeFocusOrder)
		} else {
			cur = (cur - 1 + len(composeFocusOrder)) % len(composeFocusOrder)
		}
		model.focusedElement = composeFocusOrder[cur]
	}

	if model.focusedElement == focusComposeSummary {
		return model.commitsKeysInput.Focus()
	}
	model.commitsKeysInput.Blur()
	return nil
}

// UPDATE functions
