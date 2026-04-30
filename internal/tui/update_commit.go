package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"
)

func updateChoosingScope(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if model.fileList.FilterState() == list.Filtering {
			switch {
			case key.Matches(msg, model.keys.Up):
				model.fileList.CursorUp()
				return model, nil
			case key.Matches(msg, model.keys.Down):
				model.fileList.CursorDown()
				return model, nil
			}
		}
		switch {
		case key.Matches(msg, model.keys.Esc):
			return model.cancelProcess(stateChoosingType)
		case key.Matches(msg, model.keys.Toggle):
			model.fileListFilter = !model.fileListFilter
			model.currentUpdateFileListFn = ChooseUpdateFileListFunction(model.fileListFilter)
			model.currentUpdateFileListFn(scopeFilePickerPwd, &model.fileList, model.gitStatusData)
			ResetAndActiveFilterOnList(&model.fileList)
		case key.Matches(msg, model.keys.Left):
			parentDir := filepath.Dir(scopeFilePickerPwd)
			absParentDir := filepath.Clean(parentDir)
			absGitRoot := filepath.Clean(model.gitStatusData.Root)
			if absParentDir == absGitRoot || strings.HasPrefix(absParentDir, absGitRoot+string(filepath.
				Separator)) {
				scopeFilePickerPwd = parentDir
				model.WritingStatusBar.Level = statusbar.LevelInfo
				model.WritingStatusBar.Content = fmt.Sprintf("Choose a file or folder for your commit ::: %s", model.Theme.AppStyles().Base.Foreground(model.Theme.Tertiary).SetString(TruncatePath(scopeFilePickerPwd, 2)).String())
				model.currentUpdateFileListFn(parentDir, &model.fileList, model.gitStatusData)
				ResetAndActiveFilterOnList(&model.fileList)
			} else {
				model.WritingStatusBar.Level = statusbar.LevelError
				model.WritingStatusBar.Content = "You cannot move outside the workspace"
			}

			return model, nil
		case key.Matches(msg, model.keys.Right):
			selected := model.fileList.SelectedItem()
			if item, ok := selected.(FileItem); ok {
				if item.IsDir() {
					scopeFilePickerPwd = filepath.Join(scopeFilePickerPwd, item.Title())
					model.currentUpdateFileListFn(scopeFilePickerPwd, &model.fileList, model.gitStatusData)
					ResetAndActiveFilterOnList(&model.fileList)
					model.WritingStatusBar.Level = statusbar.LevelInfo
					model.WritingStatusBar.Content = fmt.Sprintf("Choose a file or folder for your commit ::: %s", model.Theme.AppStyles().Base.Foreground(model.Theme.Tertiary).SetString(TruncatePath(scopeFilePickerPwd, 2)).String())

				} else {
					model.WritingStatusBar.Level = statusbar.LevelError
					model.WritingStatusBar.Content = "The selected item is not a directory"
				}
				return model, nil
			}
		case key.Matches(msg, model.keys.Enter):
			commitScopeSelected := model.fileList.SelectedItem()
			if item, ok := commitScopeSelected.(FileItem); ok {
				model.WritingStatusBar.Level = statusbar.LevelInfo
				model.WritingStatusBar.Content = "Craft your commit"
				model.addScope(item.Title())
				model.state = stateWritingMessage
				model.focusedElement = focusComposeSummary
				model.keys = writingMessageKeys()
				cmd = model.commitsKeysInput.Focus()
				return model, cmd
			}
			return model, nil
		}
	}

	model.fileList, cmd = model.fileList.Update(msg)
	return model, cmd
}

func updateChoosingType(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if model.commitTypeList.FilterState() == list.Filtering {
			switch {
			case key.Matches(msg, model.keys.Up):
				model.commitTypeList.CursorUp()
				return model, nil
			case key.Matches(msg, model.keys.Down):
				model.commitTypeList.CursorDown()
				return model, nil
			}
		}
		switch {
		case key.Matches(msg, model.keys.Esc):
			model.keys = mainListKeys()
			return model.cancelProcess(stateChoosingCommit)
		case key.Matches(msg, model.keys.Enter):
			commitTypeSelected := model.commitTypeList.SelectedItem()
			if item, ok := commitTypeSelected.(CommitTypeItem); ok {
				model.commitType = item.Tag
				scopeFilePickerPwd = model.pwd
				gitStatusMap, err := git.GetGitDiffNameStatus()
				if err != nil {
					model.log.Error("Error getting git diff status", "error", err)
				}
				model.log.Debug("Git Diff Status Map", "map_content", fmt.Sprintf("%v", gitStatusMap))
				model.WritingStatusBar.Content = fmt.Sprintf("Choose a file or folder for your commit ::: %s", model.Theme.AppStyles().Base.Foreground(model.Theme.Tertiary).SetString(TruncatePath(scopeFilePickerPwd, 2)).String())
				model.state = stateChoosingScope
				model.keys = fileListKeys()
				ResetAndActiveFilterOnList(&model.fileList)
				return model, nil
			}
		}
	}

	model.commitTypeList, cmd = model.commitTypeList.Update(msg)
	return model, cmd
}

// updateChoosingType handles the logic for the type-choosing state.
func updateChoosingCommit(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ctrl+f cycles the filter mode (title → id → type → scope) at
		// any time on the workspace view. When the filter bar is empty
		// it just swaps the mode pill; when there is an active query we
		// re-apply it so DefaultFilter re-runs against the new
		// FilterValue strings.
		if msg.String() == "ctrl+f" {
			model.historyView.CycleFilterMode()
			val := model.historyView.FilterValue()
			if val == "" {
				model.mainList.SetFilterText("")
				model.mainList.SetFilterState(list.Unfiltered)
			} else {
				// Reset+set forces the bubbles list to recompute the
				// filter results with the just-changed FilterValue
				// implementation.
				model.mainList.SetFilterText("")
				model.mainList.SetFilterText(val)
				model.mainList.SetFilterState(list.Filtering)
			}
			syncHistoryViewSelection(model)
			return model, nil
		}
		// FilterBar focus: route keys to the textinput. Esc clears + blurs;
		// Enter blurs without clearing (so the user can navigate the filtered
		// list); every other key is forwarded and the master list's filter is
		// kept in sync after each value change.
		if model.historyView.IsFilterFocused() {
			switch msg.String() {
			case "esc":
				model.historyView.ResetFilter()
				model.historyView.BlurFilter()
				model.mainList.SetFilterText("")
				model.mainList.SetFilterState(list.Unfiltered)
				return model, nil
			case "enter":
				model.historyView.BlurFilter()
				return model, nil
			}
			cmd, changed := model.historyView.UpdateFilter(msg)
			if changed {
				val := model.historyView.FilterValue()
				model.mainList.SetFilterText(val)
				if val == "" {
					model.mainList.SetFilterState(list.Unfiltered)
				} else {
					model.mainList.SetFilterState(list.Filtering)
				}
				syncHistoryViewSelection(model)
			}
			return model, cmd
		}
		switch msg.String() {
		case "pgup", "pgdown", "ctrl+up", "ctrl+down":
			panelCmd := model.historyView.UpdatePanel(msg)
			return model, panelCmd
		}
		switch {
		case key.Matches(msg, model.keys.SwapMode):
			model.historyView.ToggleMode()
			return model, nil
		case key.Matches(msg, model.keys.CycleNext):
			model.historyView.CycleLeftCursor(+1)
			return model, nil
		case key.Matches(msg, model.keys.CyclePrev):
			model.historyView.CycleLeftCursor(-1)
			return model, nil
		case key.Matches(msg, model.keys.Filter):
			return model, model.historyView.FocusFilter()
		case key.Matches(msg, model.keys.Quit):
			return quitWithAutodraft(model)
		case key.Matches(msg, model.keys.AddCommit):
			model.currentCommit = storage.Commit{}
			model.keyPoints = nil
			model.resetScopes()
			if len(model.finalCommitTypes) > 0 {
				model.commitType = model.finalCommitTypes[0].Tag
			}
			model.commitsKeysInput.SetValue("")
			model.commitTranslate = ""
			model.iaSummaryOutput = ""
			model.iaCommitRawOutput = ""
			model.iaTitleRawOutput = ""
			model.WritingStatusBar.Content = "Craft your commit"
			model.WritingStatusBar.Level = statusbar.LevelInfo
			model.state = stateWritingMessage
			model.keys = writingMessageKeys()
			model.focusedElement = focusComposeSummary
			cmd = model.commitsKeysInput.Focus()
			return model, cmd
		case key.Matches(msg, model.keys.ReleaseCommit):
			model.WritingStatusBar.Content = "Select the commits to create a release"
			model.state = stateReleaseChoosingCommits
			model.releaseCommitList = NewReleaseCommitList(model.pwd, model.Theme)
			model.releaseCommitList.Select(0)
			model.focusedElement = focusListElement
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
			model.keys = releaseKeys()
			return model, nil
		case key.Matches(msg, model.keys.EditIaCommit):
			selectedItem := model.mainList.SelectedItem()
			if commitItem, ok := selectedItem.(HistoryCommitItem); ok {
				commit := commitItem.commit
				model.currentCommit = commit
				model.loadScopesFromString(commit.Scope)
				model.commitType = commit.Type
				model.diffCode = commit.Diff_code
				model.commitMsg = strings.Join(commit.KeyPoints, "\n")
				model.commitTranslate = commit.MessageEN
				model.iaSummaryOutput = commit.IaSummary
				model.iaCommitRawOutput = commit.IaCommitRaw
				model.iaTitleRawOutput = commit.IaTitle
				model.iaChangelogEntry = commit.IaChangelog
				model.usePreloadedDiff = true
				model.scopeDataStale = true
				model.syncScopeStaleIndicator()
				model.keyPoints = commit.KeyPoints
				loadPipelineFilesFromDb(model, commit.Diff_code)
				loadPipelineAICalls(model, commit.ID)
				model.state = stateWritingMessage
				model.focusedElement = focusComposeSummary
				model.iaViewport.SetContent(commit.MessageEN)
				model.keys = writingMessageKeys()
				if cmd := model.commitsKeysInput.Focus(); cmd != nil {
					return model, cmd
				}
			}
			return model, nil

		case key.Matches(msg, model.keys.Delete):
			return model, func() tea.Msg { return openPopupMsg{Type: Confirmation, Db: commitDb} }
		case key.Matches(msg, model.keys.CreateLocalTomlConfig):
			CreateLocalConfigTomlTmpl()
			cmd := model.WritingStatusBar.ShowMessageForDuration("Configuration file created!", statusbar.LevelSuccess, 2*time.Second)
			return model, cmd
		case key.Matches(msg, model.keys.Enter):
			selectedItem, ok := model.mainList.SelectedItem().(HistoryCommitItem)
			if !ok {
				return model, nil // Should not happen
			}

			commit := selectedItem.commit
			if commit.Status == "draft" {
				// Load draft into editor
				model.currentCommit = commit
				model.commitMsg = strings.Join(commit.KeyPoints, "\n")
				model.keyPoints = commit.KeyPoints
				model.commitTranslate = commit.MessageEN
				model.commitType = commit.Type
				model.loadScopesFromString(commit.Scope)
				model.diffCode = commit.Diff_code
				model.iaSummaryOutput = commit.IaSummary
				model.iaCommitRawOutput = commit.IaCommitRaw
				model.iaTitleRawOutput = commit.IaTitle
				model.iaChangelogEntry = commit.IaChangelog
				model.usePreloadedDiff = true
				model.scopeDataStale = true
				model.syncScopeStaleIndicator()
				loadPipelineFilesFromDb(model, commit.Diff_code)
				loadPipelineAICalls(model, commit.ID)
				model.state = stateWritingMessage
				model.focusedElement = focusComposeSummary
				model.iaViewport.SetContent(commit.MessageEN)
				model.keys = writingMessageKeys()
				model.WritingStatusBar.Content = "Continuing with draft..."
				if cmd := model.commitsKeysInput.Focus(); cmd != nil {
					return model, cmd
				}
				return model, nil
			} else {
				model.currentCommit = commit
				if model.OutputDirect {
					model.FinalMessage = assembleOutputCommitMessage(model, commit)
					return model, tea.Quit
				}
				var menuOptions []itemsOptions
				menu := []string{"Output message", "Open output view", "Reword with this message", "Reword with new AI run"}
				menuOptions = append(menuOptions, itemsOptions{index: 0, color: model.Theme.Red, icon: model.Theme.AppSymbols().Console})
				menuOptions = append(menuOptions, itemsOptions{index: 1, color: model.Theme.Secondary, icon: model.Theme.AppSymbols().Console})
				menuOptions = append(menuOptions, itemsOptions{index: 2, color: model.Theme.Yellow, icon: model.Theme.AppSymbols().ReuseMessage})
				menuOptions = append(menuOptions, itemsOptions{index: 3, color: model.Theme.Success, icon: model.Theme.AppSymbols().NewDbRecord})
				return model, func() tea.Msg {
					return openListPopup{items: menu, itemsOptions: menuOptions, width: model.width / 2, height: model.height / 2}
				}
			}

		case key.Matches(msg, model.keys.SwitchMode):
			model.AppMode = ReleaseMode
			model.state = stateReleaseMainMenu
			model.keys = releaseMainListKeys()
			loadCmd := enterReleaseHistoryLoading(model)
			model.WritingStatusBar.Content = fmt.Sprintf(
				"choose, create, or edit a release ::: %s",
				model.Theme.AppStyles().
					Base.Foreground(model.Theme.Tertiary).
					SetString(model.mainList.Title),
			)
			cmd = model.WritingStatusBar.ShowMessageForDuration(
				"Change app mode: Release",
				statusbar.LevelWarning,
				2*time.Second,
			)
			return model, tea.Batch(loadCmd, cmd)

		case key.Matches(msg, model.keys.ToggleDrafts):
			model.draftMode = !model.draftMode
			status := "completed"
			msg := "Showing completed commits"
			if model.draftMode {
				status = "draft"
				msg = "Showing drafts"
			}
			commits, err := model.db.GetCommits(model.pwd, status)
			if err != nil {
				model.err = err
				return model, nil
			}
			items := make([]list.Item, len(commits))
			for i, c := range commits {
				items[i] = HistoryCommitItem{commit: c}
			}
			model.mainList.SetItems(items)
			// Ensure the viewport is updated
			if len(items) > 0 {
				model.mainList.Select(0)
			}
			model.mainList.Title = msg
			cmd := model.WritingStatusBar.ShowMessageForDuration(msg, statusbar.LevelSuccess, 2*time.Second)
			return model, cmd
		}
	}

	model.mainList, cmd = model.mainList.Update(msg)
	syncHistoryViewSelection(model)
	return model, cmd
}

// syncHistoryViewSelection mirrors the master list's current selection into
// the HistoryView's DualPanel so the inspection split always reflects the
// commit under the cursor. Safe to call after every keystroke that may have
// moved the cursor or rebuilt the items.
func syncHistoryViewSelection(model *Model) {
	if item, ok := model.mainList.SelectedItem().(HistoryCommitItem); ok {
		model.historyView.SetCommit(item.commit, loadCommitAICalls(model, item.commit.ID))
	} else {
		model.historyView.ClearCommit()
	}
}

// loadCommitAICalls returns the ai_calls slice for a commit, or nil when
// the lookup fails. The HistoryDualPanel uses the slice both to decide
// whether stage 4 belongs in the inspect list and to render the per-stage
// telemetry strip in Stages/Response mode — a single SQLite read replaces
// what used to be a "does the changelog stage exist?" yes/no probe.
func loadCommitAICalls(model *Model, commitID int) []storage.AICall {
	if model == nil || model.db == nil || commitID <= 0 {
		return nil
	}
	calls, err := model.db.GetAICallsByCommitID(commitID)
	if err != nil {
		model.log.Warn("ai_calls lookup failed", "commit_id", commitID, "error", err)
		return nil
	}
	return calls
}
