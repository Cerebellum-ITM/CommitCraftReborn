package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/tui/statusbar"
	"commit_craft_reborn/internal/tui/styles"
)

type releaseUpdloadResultMsg struct {
	Err error
}

type IaCommitBuilderResultMsg struct {
	Err error
}

type IaResleaseBuilderResultMsg struct {
	Err error
}

type (
	IaSummaryResultMsg      struct{ Err error }
	IaCommitRawResultMsg    struct{ Err error }
	IaOutputFormatResultMsg struct{ Err error }
	IaChangelogResultMsg    struct{ Err error }
)

// Main Update Function
func (model *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure the model is passed as a pointer.
	var cmds []tea.Cmd
	var cmd tea.Cmd
	model.WritingStatusBar, cmd = model.WritingStatusBar.Update(msg)
	cmds = append(cmds, cmd)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model.width = msg.Width
		model.height = msg.Height
		model.apiKeyInput.SetWidth(model.width)
		model.WritingStatusBar.AppWith = model.width
	case list.FilterMatchesMsg:
		// Filter results are produced asynchronously by bubbles/list
		// when the user types in a popup like the scope picker. The
		// runtime delivers the msg here; route it back to the popup so
		// its inner list can apply the filtered items.
		if model.popup != nil {
			var popupCmd tea.Cmd
			model.popup, popupCmd = model.popup.Update(msg)
			return model, popupCmd
		}
	case openPopupMsg:
		switch msg.Type {
		case Confirmation:
			switch msg.Db {
			case commitDb:
				selectedItem := model.mainList.SelectedItem()
				if commitItem, ok := selectedItem.(HistoryCommitItem); ok {
					model.popup = NewPopup(model.width, model.height, commitItem.commit.ID, strings.Join(commitItem.commit.KeyPoints, " · "), commitDb, WithColor(model.Theme.Warning), WithTheme(model.Theme))
				}
			case releaseDb:
				if seletedItem, ok := model.releaseMainList.SelectedItem().(HistoryReleaseItem); ok {
					model.popup = NewPopup(model.width, model.height, seletedItem.release.ID, seletedItem.release.Title, releaseDb, WithColor(model.Theme.Warning), WithTheme(model.Theme))
				}
			}
		}
		return model, nil
	case openListPopup:
		var opts []PopupListOption
		if msg.title != "" {
			opts = append(opts, ListWithTitle(msg.title))
		}
		if msg.color != nil {
			opts = append(opts, ListWithColor(msg.color))
		}
		model.log.Debug(fmt.Sprintf("%v", msg.itemsOptions))
		model.popup = NewListPopup(msg.items, msg.itemsOptions, msg.width, msg.height, listKeys(), model.Theme, opts...)
		return model, nil
	case closePopupMsg, closeListPopup:
		model.popup = nil
		return model, nil
	case closeDiffViewPopupMsg:
		model.popup = nil
		return model, nil
	case closeTypePopupMsg:
		model.popup = nil
		return model, nil
	case setCommitTypeMsg:
		model.popup = nil
		model.commitType = msg.tag
		model.commitTypeColor = msg.color
		return model, nil
	case closeScopePopupMsg:
		model.popup = nil
		return model, nil
	case setScopeMsg:
		model.popup = nil
		model.addScope(msg.scope)
		return model, nil
	case closeEditMessagePopupMsg:
		model.popup = nil
		return model, nil
	case themePreviewMsg:
		model.Theme = styles.GetTheme(msg.name, model.globalConfig.TUI.UseNerdFonts)
		model.WritingStatusBar.SetTheme(model.Theme)
		return model, nil
	case themeAppliedMsg:
		model.Theme = styles.GetTheme(msg.name, model.globalConfig.TUI.UseNerdFonts)
		model.WritingStatusBar.SetTheme(model.Theme)
		model.themeName = msg.name
		model.globalConfig.TUI.Theme = msg.name
		model.popup = nil
		if err := UpdateConfigTheme(msg.name); err != nil {
			model.log.Error("Failed to persist theme", "error", err)
			return model, model.WritingStatusBar.ShowMessageForDuration(
				fmt.Sprintf("Theme applied (not saved: %s)", err),
				statusbar.LevelWarning,
				3*time.Second,
			)
		}
		return model, model.WritingStatusBar.ShowMessageForDuration(
			fmt.Sprintf("Theme set to %s", msg.name),
			statusbar.LevelSuccess,
			2*time.Second,
		)
	case closeConfigPopupMsg:
		model.Theme = styles.GetTheme(model.themeName, model.globalConfig.TUI.UseNerdFonts)
		model.WritingStatusBar.SetTheme(model.Theme)
		model.popup = nil
		return model, nil
	case closeModelPickerMsg:
		model.popup = nil
		return model, nil
	case closeStageHistoryMsg:
		model.popup = nil
		return model, nil
	case closeKeybindingsPopupMsg:
		model.popup = nil
		return model, nil
	case releaseHistorySyncMsg:
		// Async result of startReleaseHistorySync — apply it to the
		// dual panel and clear the loading flag so the regular release
		// chrome takes over the next frame.
		if msg.cleared {
			model.releaseHistoryView.ClearRelease()
		} else {
			model.releaseHistoryView.SetRelease(msg.release, msg.messages, msg.calls)
		}
		model.releaseLoading = false
		return model, nil
	case spinner.TickMsg:
		// Drive the model-level spinner used by the release loading
		// screen. Other tabs own their own spinners and are unaffected
		// because spinner.Update silently ignores ticks for spinner ids
		// it didn't issue.
		if model.releaseLoading {
			var spCmd tea.Cmd
			model.spinner, spCmd = model.spinner.Update(msg)
			return model, spCmd
		}
	case stageHistoryApplyMsg:
		model.popup = nil
		entry, ok := model.pipeline.applyStageHistoryEntry(msg.stage, msg.index)
		if !ok {
			return model, nil
		}
		switch msg.stage {
		case stageSummary:
			model.iaSummaryOutput = entry.Text
		case stageBody:
			model.iaCommitRawOutput = entry.Text
		case stageTitle:
			model.iaTitleRawOutput = entry.Text
		case stageChangelog:
			model.iaChangelogEntry = entry.Text
		}
		model.commitTranslate = composeFinalCommitMessage(model)
		if vp := model.stageViewportModel(msg.stage); vp != nil {
			vp.GotoTop()
		}
		return model, model.WritingStatusBar.ShowMessageForDuration(
			fmt.Sprintf("Applied %s history v%d/%d",
				model.pipeline.stages[msg.stage].Title,
				msg.index+1,
				len(model.pipeline.stages[msg.stage].History),
			),
			statusbar.LevelSuccess,
			2*time.Second,
		)
	case modelPickerOpenedMsg:
		if msg.err != nil && len(msg.models) == 0 {
			model.popup = nil
			model.log.Error("model picker fetch failed", "error", msg.err)
			cmd := model.WritingStatusBar.ShowMessageForDuration(
				fmt.Sprintf("Could not load Groq models: %s", msg.err),
				statusbar.LevelError,
				3*time.Second,
			)
			return model, cmd
		}
		w := max(60, model.width*2/3)
		h := max(20, model.height*2/3)
		if w > model.width-4 {
			w = model.width - 4
		}
		if h > model.height-4 {
			h = model.height - 4
		}
		model.popup = newModelPickerPopup(
			msg.stage, msg.label, msg.current, msg.models, msg.cachedAt,
			w, h, model.Theme,
		)
		var statusCmd tea.Cmd
		if msg.err != nil {
			statusCmd = model.WritingStatusBar.ShowMessageForDuration(
				"Showing cached models (refresh failed)",
				statusbar.LevelWarning,
				3*time.Second,
			)
		}
		return model, statusCmd
	case modelPickerRefreshMsg:
		stage := msg.stage
		label := stageLabelFor(model, stage)
		statusCmd := model.WritingStatusBar.ShowMessageForDuration(
			"Refreshing models from Groq…",
			statusbar.LevelInfo,
			2*time.Second,
		)
		return model, tea.Batch(statusCmd, refreshModelPickerCmd(model, stage, label))
	case modelPickerResultMsg:
		model.popup = nil
		if err := config.SaveModelForStage(msg.stage, msg.modelID, msg.scope); err != nil {
			model.log.Error("save model failed", "error", err)
			cmd := model.WritingStatusBar.ShowMessageForDuration(
				fmt.Sprintf("Could not save model: %s", err),
				statusbar.LevelError,
				3*time.Second,
			)
			return model, cmd
		}
		config.ApplyModelToConfig(&model.globalConfig, msg.stage, msg.modelID)
		applyPipelineModelsToStages(model)
		scopeLabel := "global"
		if msg.scope == config.ScopeLocal {
			scopeLabel = "local"
		}
		cmd := model.WritingStatusBar.ShowMessageForDuration(
			fmt.Sprintf("Set %s → %s (%s)", msg.stage, msg.modelID, scopeLabel),
			statusbar.LevelSuccess,
			3*time.Second,
		)
		return model, cmd
	case editMessageAppliedMsg:
		model.popup = nil
		model.commitTranslate = msg.value
		cmd := model.WritingStatusBar.ShowMessageForDuration(
			"Changes applied", statusbar.LevelSuccess, 2*time.Second,
		)
		return model, cmd
	case closeVersionPopupMsg:
		model.popup = nil
		if model.pendingReleaseUpload != nil {
			model.pendingReleaseUpload = nil
			cancelCmd := model.WritingStatusBar.ShowMessageForDuration(
				"GitHub release cancelled",
				statusbar.LevelWarning,
				2*time.Second,
			)
			return model, cancelCmd
		}
		return model, nil
	case versionUpdatedMsg:
		model.popup = nil
		if msg.err != nil {
			model.log.Error("Error updating release version", "error", msg.err)
			model.WritingStatusBar.Content = fmt.Sprintf("Error: %s", msg.err)
			model.WritingStatusBar.Level = statusbar.LevelError
			model.pendingReleaseUpload = nil
			return model, nil
		}
		model.globalConfig.ReleaseConfig.Version = msg.version

		// If the version editor was popped as a pre-upload step, chain
		// straight into the GitHub release with the freshly-saved tag.
		if pending := model.pendingReleaseUpload; pending != nil {
			model.releaseViewState.releaseCreated = true
			rc := model.globalConfig.ReleaseConfig
			if rc.AutoBuild {
				model.WritingStatusBar.Level = statusbar.LevelWarning
				model.WritingStatusBar.Content = fmt.Sprintf(
					"Building release assets (%s %s)…",
					rc.BuildTool,
					rc.BuildTarget,
				)
				spinnerCmd := model.WritingStatusBar.StartSpinner()
				return model, tea.Batch(spinnerCmd, execReleaseBuild(model))
			}
			model.pendingReleaseUpload = nil
			model.WritingStatusBar.Level = statusbar.LevelWarning
			model.WritingStatusBar.Content = fmt.Sprintf(
				"Creating GitHub release %s…",
				msg.version,
			)
			spinnerCmd := model.WritingStatusBar.StartSpinner()
			return model, tea.Batch(spinnerCmd, execUploadRelease(*pending, model))
		}

		statusCmd := model.WritingStatusBar.ShowMessageForDuration(
			fmt.Sprintf("Release version set to %s", msg.version),
			statusbar.LevelSuccess,
			2*time.Second,
		)
		return model, statusCmd
	case diffFetchedMsg:
		if msg.err != nil {
			model.WritingStatusBar.Content = fmt.Sprintf("Error loading diff: %s", msg.err)
			model.WritingStatusBar.Level = statusbar.LevelError
			return model, nil
		}
		model.popup = newDiffViewPopup(msg.filePath, msg.content, model.width, model.height, model.Theme)
		return model, nil
	case mentionFileSelectedMsg:
		model.popup = nil
		currentVal := model.commitsKeysInput.Value()
		runes := []rune(currentVal)
		// Keep the leading `@` the user typed: replace from mentionStart+1
		// (the char right after `@`) onwards with the picked filename. The
		// `@` itself stays in the buffer so the mention is recognisable as
		// a coloured chip when we render the value ourselves; the marker
		// is stripped only just before the AI prompt is built.
		head := string(runes[:model.mentionStart])
		if model.mentionStart < len(runes) && runes[model.mentionStart] == '@' {
			head = string(runes[:model.mentionStart+1])
		} else {
			head += "@"
		}
		newVal := head + msg.filename + " "
		model.commitsKeysInput.SetValue(newVal)
		cmd = model.commitsKeysInput.Focus()
		return model, cmd
	case closeMentionPopupMsg:
		// Cancel: leave whatever the user already typed in place, including
		// the `@`. The user explicitly opted to keep mention markers in the
		// editable buffer, so the cancel path no longer rewrites the value.
		model.popup = nil
		cmd = model.commitsKeysInput.Focus()
		return model, cmd
	case releaseAction:
		model.popup = nil

		switch msg.action {
		case rewordChooseAsCommit:
			return setupCommitReword(model)
		case rewordChooseAsRelease:
			return setupReleaseReword(model)
		case "Create":
			_, cmd := createRelease(model)
			return model, cmd
		case "Print in console":
			if selectedItem, ok := model.releaseMainList.SelectedItem().(HistoryReleaseItem); ok {
				formattedReleaseType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, selectedItem.release.Type)
				model.FinalMessage = fmt.Sprintf("%s %s: %s\n\n%s", formattedReleaseType, selectedItem.release.Branch, selectedItem.release.Title, selectedItem.release.Body)
			}
			return model, tea.Quit
		case "Output message":
			model.FinalMessage = assembleOutputCommitMessage(model, model.currentCommit)
			return model, tea.Quit
		case "Reword commit":
			model.releaseCommitList = NewReleaseCommitList(model.pwd, model.Theme)
			model.releaseCommitList.Select(0)
			model.state = stateRewordSelectCommit
			model.focusedElement = focusListElement
			model.keys = rewordSelectKeys()
			model.WritingStatusBar.Content = "Select a commit to reword"
			model.WritingStatusBar.Level = statusbar.LevelInfo
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
			return model, nil
		case "Commit and reword":
			model.releaseCommitList = NewReleaseCommitList(model.pwd, model.Theme)
			model.releaseCommitList.Select(0)
			model.state = stateRewordSelectCommit
			model.focusedElement = focusListElement
			model.keys = rewordSelectKeys()
			model.WritingStatusBar.Content = "Select the git commit to reword with a new AI message"
			model.WritingStatusBar.Level = statusbar.LevelInfo
			model.commitAndReword = true
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
			return model, nil
		case "Copy to clipboard":
			var finalMessage string
			if selectedItem, ok := model.releaseMainList.SelectedItem().(HistoryReleaseItem); ok {
				formattedReleaseType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, selectedItem.release.Type)
				finalMessage = fmt.Sprintf("%s %s: %s\n%s", formattedReleaseType, selectedItem.release.Branch, selectedItem.release.Title, selectedItem.release.Body)
			}
			if model.ToolsInfo.xclip.available {
				return model, tea.Sequence(
					tea.SetClipboard(finalMessage),
					func() tea.Msg {
						_ = clipboard.WriteAll(finalMessage)
						return nil
					},
					tea.Quit,
				)
			} else {
				err := fmt.Errorf("%s is not available in the system", model.ToolsInfo.xclip.name)
				model.log.Error("%s is not available in the system!!")
				model.err = err
				return model, tea.Quit
			}
		case "Create release in repository":
			if selectedItem, ok := model.releaseMainList.SelectedItem().(HistoryReleaseItem); ok {
				item := selectedItem
				model.pendingReleaseUpload = &item
				model.popup = openVersionEditor(model)
			}
			return model, nil
		case "Release Commit":
			model.releaseType = "REL"
			branch, err := git.GetCurrentGitBranch()
			if err != nil {
				model.log.Error("Error getting the current branch", "error", err)
				model.err = err
				return model, tea.Quit
			}
			model.releaseBranch = branch
			return model, func() tea.Msg { return releaseAction{action: "Create"} }
		case "Merge Commit":
			model.releaseType = "MERGE"
			branches, err := git.GetGitBranches()
			if err != nil {
				model.log.Error("Error getting the current branch", "error", err)
				model.err = err
				return model, tea.Quit
			}
			return model, func() tea.Msg {
				return openListPopup{items: branches, width: model.width / 2, height: model.height / 2, title: "Select a branch"}
			}
		case "Create item in CommitCraft":
			menu := []string{"Merge Commit", "Release Commit"}
			return model, func() tea.Msg { return openListPopup{items: menu, width: model.width / 2, height: model.height / 2} }
		case "Create release in Github":
			model.releaseType = "REL"
			branch, err := git.GetCurrentGitBranch()
			if err != nil {
				model.log.Error("Error getting the current branch", "error", err)
				model.err = err
				return model, tea.Quit
			}
			model.releaseBranch = branch
			createRelease(model)
			release, err := model.db.GetLatestRelease(model.pwd)
			if err != nil {
				model.err = err
				return model, tea.Quit
			}
			item := HistoryReleaseItem{release: release}
			model.pendingReleaseUpload = &item
			model.popup = openVersionEditor(model)
			return model, nil
		default:
			// NOTE: Any selected branch leads to this action
			model.releaseBranch = msg.action
			return model, func() tea.Msg { return releaseAction{action: "Create"} }
		}

	case deleteItemMsg:
		var list *list.Model
		switch msg.Db {
		case commitDb:
			err := model.db.DeleteCommit(msg.ID)
			list = &model.mainList
			if err != nil {
				model.err = err
				return model, nil
			}

		case releaseDb:
			err := model.db.DeleteRelease(msg.ID)
			list = &model.releaseMainList
			if err != nil {
				model.err = err
				return model, nil
			}
		}

		model.popup = nil
		UpdateCommitList(model.pwd, model.db, model.log, list, msg.Db)
		cmd := model.WritingStatusBar.ShowMessageForDuration("Record deleted from the db", statusbar.LevelSuccess, 2*time.Second)
		return model, cmd

	case IaCommitBuilderResultMsg:
		cmds = append(cmds, model.WritingStatusBar.StopSpinner())

		if msg.Err != nil {
			model.err = msg.Err
			model.WritingStatusBar.Content = fmt.Sprintf("Error: %s", msg.Err.Error())
			model.WritingStatusBar.Level = statusbar.LevelError
		} else if model.iaChangelogEntry != "" {
			model.WritingStatusBar.Content = fmt.Sprintf(
				"AI commit + CHANGELOG entry %s ready!",
				model.iaChangelogSuggestedVersion,
			)
			model.WritingStatusBar.Level = statusbar.LevelChangelog
		} else {
			model.WritingStatusBar.Content = "AI commit message ready!"
			model.WritingStatusBar.Level = statusbar.LevelInfo
		}
		touched := []stageID{stageSummary, stageBody, stageTitle}
		if model.changelogActive {
			touched = append(touched, stageChangelog)
		}
		if msg.Err == nil {
			model.pipeline.pushStageHistory(stageSummary, model.iaSummaryOutput)
			model.pipeline.pushStageHistory(stageBody, model.iaCommitRawOutput)
			model.pipeline.pushStageHistory(stageTitle, model.iaTitleRawOutput)
			if model.changelogActive {
				model.pipeline.pushStageHistory(stageChangelog, model.iaChangelogEntry)
			}
		}
		cmds = append(cmds, model.applyPipelineResult(touched, msg.Err))
		if model.state != statePipeline {
			model.state = stateWritingMessage
		}
		return model, tea.Batch(cmds...)

	case IaSummaryResultMsg:
		cmds = append(cmds, model.WritingStatusBar.StopSpinner())
		if msg.Err != nil {
			model.WritingStatusBar.Content = fmt.Sprintf("Error (Stage 1): %s", msg.Err.Error())
			model.WritingStatusBar.Level = statusbar.LevelError
		} else if model.iaChangelogEntry != "" {
			model.WritingStatusBar.Content = fmt.Sprintf(
				"Pipeline re-run complete · CHANGELOG %s ready!",
				model.iaChangelogSuggestedVersion,
			)
			model.WritingStatusBar.Level = statusbar.LevelChangelog
		} else {
			model.WritingStatusBar.Content = "Pipeline re-run complete!"
			model.WritingStatusBar.Level = statusbar.LevelInfo
		}
		touched1 := []stageID{stageSummary, stageBody, stageTitle}
		if model.changelogActive {
			touched1 = append(touched1, stageChangelog)
		}
		if msg.Err == nil {
			model.pipeline.pushStageHistory(stageSummary, model.iaSummaryOutput)
			model.pipeline.pushStageHistory(stageBody, model.iaCommitRawOutput)
			model.pipeline.pushStageHistory(stageTitle, model.iaTitleRawOutput)
			if model.changelogActive {
				model.pipeline.pushStageHistory(stageChangelog, model.iaChangelogEntry)
			}
		}
		cmds = append(cmds, model.applyPipelineResult(touched1, msg.Err))
		if model.state != statePipeline {
			model.state = stateWritingMessage
		}
		return model, tea.Batch(cmds...)

	case IaCommitRawResultMsg:
		cmds = append(cmds, model.WritingStatusBar.StopSpinner())
		if msg.Err != nil {
			model.WritingStatusBar.Content = fmt.Sprintf("Error (Stage 2): %s", msg.Err.Error())
			model.WritingStatusBar.Level = statusbar.LevelError
		} else if model.iaChangelogEntry != "" {
			model.WritingStatusBar.Content = fmt.Sprintf(
				"Stages 2+3+CHANGELOG re-run complete (%s)!",
				model.iaChangelogSuggestedVersion,
			)
			model.WritingStatusBar.Level = statusbar.LevelChangelog
		} else {
			model.WritingStatusBar.Content = "Stages 2+3 re-run complete!"
			model.WritingStatusBar.Level = statusbar.LevelInfo
		}
		touched2 := []stageID{stageBody, stageTitle}
		if model.changelogActive {
			touched2 = append(touched2, stageChangelog)
		}
		if msg.Err == nil {
			model.pipeline.pushStageHistory(stageBody, model.iaCommitRawOutput)
			model.pipeline.pushStageHistory(stageTitle, model.iaTitleRawOutput)
			if model.changelogActive {
				model.pipeline.pushStageHistory(stageChangelog, model.iaChangelogEntry)
			}
		}
		cmds = append(cmds, model.applyPipelineResult(touched2, msg.Err))
		if model.state != statePipeline {
			model.state = stateWritingMessage
		}
		return model, tea.Batch(cmds...)

	case IaOutputFormatResultMsg:
		cmds = append(cmds, model.WritingStatusBar.StopSpinner())
		if msg.Err != nil {
			model.WritingStatusBar.Content = fmt.Sprintf("Error (Stage 3): %s", msg.Err.Error())
			model.WritingStatusBar.Level = statusbar.LevelError
		} else if model.iaChangelogEntry != "" {
			model.WritingStatusBar.Content = fmt.Sprintf(
				"Stage 3+CHANGELOG re-run complete (%s)!",
				model.iaChangelogSuggestedVersion,
			)
			model.WritingStatusBar.Level = statusbar.LevelChangelog
		} else {
			model.WritingStatusBar.Content = "Stage 3 re-run complete!"
			model.WritingStatusBar.Level = statusbar.LevelInfo
		}
		touched3 := []stageID{stageTitle}
		if model.changelogActive {
			touched3 = append(touched3, stageChangelog)
		}
		if msg.Err == nil {
			model.pipeline.pushStageHistory(stageTitle, model.iaTitleRawOutput)
			if model.changelogActive {
				model.pipeline.pushStageHistory(stageChangelog, model.iaChangelogEntry)
			}
		}
		cmds = append(cmds, model.applyPipelineResult(touched3, msg.Err))
		if model.state != statePipeline {
			model.state = stateWritingMessage
		}
		return model, tea.Batch(cmds...)

	case IaChangelogResultMsg:
		cmds = append(cmds, model.WritingStatusBar.StopSpinner())
		if msg.Err != nil {
			model.WritingStatusBar.Content = fmt.Sprintf("Error (Stage 4): %s", msg.Err.Error())
			model.WritingStatusBar.Level = statusbar.LevelError
		} else if model.iaChangelogEntry != "" {
			model.WritingStatusBar.Content = fmt.Sprintf(
				"CHANGELOG entry %s refreshed!",
				model.iaChangelogSuggestedVersion,
			)
			model.WritingStatusBar.Level = statusbar.LevelChangelog
		} else {
			model.WritingStatusBar.Content = "Changelog refiner produced no entry"
			model.WritingStatusBar.Level = statusbar.LevelWarning
		}
		if msg.Err == nil {
			model.pipeline.pushStageHistory(stageChangelog, model.iaChangelogEntry)
		}
		cmds = append(cmds, model.applyPipelineResult(
			[]stageID{stageChangelog}, msg.Err,
		))
		if model.state != statePipeline {
			model.state = stateWritingMessage
		}
		return model, tea.Batch(cmds...)

	case IaResleaseBuilderResultMsg:
		cmds = append(cmds, model.WritingStatusBar.StopSpinner())

		if msg.Err != nil {
			model.err = msg.Err
			model.WritingStatusBar.Content = fmt.Sprintf("Error: %s", msg.Err.Error())
			model.WritingStatusBar.Level = statusbar.LevelError
		} else {
			model.WritingStatusBar.Content = "AI release message ready!"
			model.WritingStatusBar.Level = statusbar.LevelInfo
		}
		return model, tea.Batch(cmds...)
	case releaseBuildResultMsg:
		if msg.Err != nil {
			model.log.Debug("release build failed", "output", msg.Output, "err", msg.Err)
			model.pendingReleaseUpload = nil
			cmds = append(cmds, model.WritingStatusBar.StopSpinner())
			model.err = msg.Err
			model.WritingStatusBar.Content = "Build failed — see logs"
			model.WritingStatusBar.Level = statusbar.LevelError
			return model, tea.Batch(cmds...)
		}
		model.log.Debug("release build ok", "output", msg.Output)
		pending := model.pendingReleaseUpload
		model.pendingReleaseUpload = nil
		if pending == nil {
			cmds = append(cmds, model.WritingStatusBar.StopSpinner())
			return model, tea.Batch(cmds...)
		}
		model.WritingStatusBar.Level = statusbar.LevelWarning
		model.WritingStatusBar.Content = fmt.Sprintf(
			"Creating GitHub release %s…",
			model.globalConfig.ReleaseConfig.Version,
		)
		cmds = append(cmds, execUploadRelease(*pending, model))
		return model, tea.Batch(cmds...)
	case releaseUpdloadResultMsg:
		cmds = append(cmds, model.WritingStatusBar.StopSpinner())
		if msg.Err != nil {
			model.err = msg.Err
			model.WritingStatusBar.Content = fmt.Sprintf("Error: %s", msg.Err.Error())
			model.WritingStatusBar.Level = statusbar.LevelError
		} else {
			model.WritingStatusBar.Content = "The release was successfully uploaded to Github"
			model.WritingStatusBar.Level = statusbar.LevelInfo
		}
		return model, tea.Batch(cmds...)
	case logLineMsg:
		if model.logViewVisible {
			model.refreshLogsViewport()
		}
		cmds = append(cmds, waitForLogLineCmd(model.logsCh))
		return model, tea.Batch(cmds...)
	case logsChannelClosedMsg:
		return model, tea.Batch(cmds...)
	case tea.KeyMsg:
		// Global hard-quit. Runs ahead of every other handler — popups,
		// logs viewer, per-state code — so Ctrl+X always exits the TUI
		// regardless of whatever overlay is open or which input is
		// focused. Without this guard the popup-routing block below
		// would swallow the key whenever a popup was active.
		if msg.String() == "ctrl+x" {
			// The version popup uses ctrl+x to decrement the version
			// component under the cursor. Let the popup consume it
			// instead of quitting the TUI.
			if _, ok := model.popup.(versionPopupModel); !ok {
				return model, tea.Quit
			}
		}
		// Global logs popup toggle — works on top of any state, even with
		// another popup open, as long as we're not typing in a filter.
		if msg.String() == "ctrl+l" {
			model.logViewVisible = !model.logViewVisible
			if model.logViewVisible {
				w, h := model.logsPopupSize()
				model.logViewport.SetWidth(w - 4)
				model.logViewport.SetHeight(h - 4)
				model.refreshLogsViewport()
			}
			return model, nil
		}
		if model.logViewVisible {
			if msg.String() == "esc" {
				model.logViewVisible = false
				return model, nil
			}
			var vpCmd tea.Cmd
			model.logViewport, vpCmd = model.logViewport.Update(msg)
			cmds = append(cmds, vpCmd)
			return model, tea.Batch(cmds...)
		}
		if model.popup != nil {
			var popupCmd tea.Cmd
			model.popup, popupCmd = model.popup.Update(msg)
			return model, popupCmd
		}
		if key.Matches(msg, versionPopupKey) {
			model.popup = openVersionEditor(model)
			return model, nil
		}
		if key.Matches(msg, openConfigPopupKey) {
			w := model.width / 3
			if w < 40 {
				w = 40
			}
			h := model.height / 2
			if h < 12 {
				h = 12
			}
			popup := newConfigPopup(w, h, model.Theme, model.themeName)
			model.popup = popup
			return model, nil
		}
		if model.shouldShowTabBar() {
			switch msg.String() {
			case "ctrl+1":
				_, _, tabCmd := model.switchToTab(tabOrder[0])
				return model, tabCmd
			case "ctrl+2":
				_, _, tabCmd := model.switchToTab(tabOrder[1])
				return model, tabCmd
			case "ctrl+3":
				_, _, tabCmd := model.switchToTab(tabOrder[2])
				return model, tabCmd
			}
		}
		switch {
		case key.Matches(msg, model.keys.GlobalQuit):
			return model, tea.Quit
		case key.Matches(msg, model.keys.Help):
			// Skip when an inline text input is focused so the user can
			// still type "?" into a filter / textarea without triggering
			// the popup toggle.
			if model.state == stateChoosingCommit && model.historyView.IsFilterFocused() {
				break
			}
			if model.state == stateReleaseMainMenu && model.releaseHistoryView.IsFilterFocused() {
				break
			}
			if entries, ok := keybindingsForState(model.state); ok {
				w := max(50, model.width*2/3)
				h := max(18, model.height*2/3)
				model.popup = newKeybindingsPopup(w, h, model.Theme, entries)
				return model, nil
			}
			model.help.ShowAll = !model.help.ShowAll
			return model, func() tea.Msg {
				return tea.WindowSizeMsg{Width: model.width, Height: model.height}
			}
		}
	}

	// Update logic depends on the current state.
	var subCmd tea.Cmd
	var subModel tea.Model

	switch model.state {
	case stateSettingAPIKey:
		subModel, subCmd = updateSettingApiKey(msg, model)
	case stateChoosingCommit:
		subModel, subCmd = updateChoosingCommit(msg, model)
	case stateChoosingType:
		subModel, subCmd = updateChoosingType(msg, model)
	case stateChoosingScope:
		subModel, subCmd = updateChoosingScope(msg, model)
	case stateWritingMessage:
		subModel, subCmd = updateWritingMessage(msg, model)
	case stateReleaseChoosingCommits:
		subModel, subCmd = updateReleaseChoosingCommits(msg, model)
	case stateReleaseBuildingText:
		subModel, subCmd = updateReleaseBuildingText(msg, model)
	case stateReleaseMainMenu:
		subModel, subCmd = updateReleaseMainMenu(msg, model)
	case statePipeline:
		subModel, subCmd = updatePipeline(msg, model)
	case stateRewordSelectCommit:
		subModel, subCmd = updateRewordSelectCommit(msg, model)
	}

	cmds = append(cmds, subCmd)
	// Keep the persistent tab indicator in sync with state transitions
	// triggered via the regular flow (Esc/Enter), so the user never sees
	// a "Compose" highlight while looking at the history list.
	model.topTab = tabForState(model.state)
	return subModel, tea.Batch(cmds...)
}
