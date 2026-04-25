package tui

import (
	"sort"
	"strings"
	"time"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/tui/statusbar"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func updateEditingMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		model.log.Debug(msg.String())
		switch {
		case key.Matches(msg, model.keys.NextField):
			cmd = switchFocusElement(model)
			return model, cmd
		case key.Matches(msg, model.keys.PrevField):
			cmd = switchFocusElement(model)
			return model, cmd
		case key.Matches(msg, model.keys.delteLine):
			lineToDelete := model.msgEdit.Line()
			lines := strings.Split(model.msgEdit.Value(), "\n")
			lines = append(lines[:lineToDelete], lines[lineToDelete+1:]...)
			model.msgEdit.SetValue(strings.Join(lines, "\n"))

			for model.msgEdit.Line() > lineToDelete {
				model.msgEdit.CursorUp()
			}

			return model, nil
		case key.Matches(msg, model.keys.Edit):
			model.state = stateWritingMessage
			model.keys = writingMessageKeys()
			model.WritingStatusBar.Level = statusbar.LevelInfo
			model.WritingStatusBar.Content = "No change was applied"

		case key.Matches(msg, model.keys.Enter):
			model.commitTranslate = model.msgEdit.Value()
			model.WritingStatusBar.Level = statusbar.LevelSuccess
			model.WritingStatusBar.Content = "Changes applied"
			model.keys = writingMessageKeys()
			model.state = stateWritingMessage
			return model, nil
		}
	}
	switch model.focusedElement {
	case focusMsgInput:
		model.msgEdit, cmd = model.msgEdit.Update(msg)
	case focusAIResponse:
		model.iaViewport, cmd = model.iaViewport.Update(msg)
	}
	return model, cmd
}

func updateWritingMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "@" && model.focusedElement == focusMsgInput {
			files, err := git.GetGitDiffNameStatus()
			if err == nil && len(files) > 0 {
				fileList := make([]string, 0, len(files))
				for f := range files {
					fileList = append(fileList, f)
				}
				sort.Strings(fileList)
				model.mentionStart = len([]rune(model.commitsKeysInput.Value()))
				model.popup = newMentionFilePopup(fileList, model.width, model.height, model.Theme)
			}
		}
		switch {
		case key.Matches(msg, model.keys.SwitchTab):
			if model.activeTab == 0 {
				model.activeTab = 1
				model.focusedElement = focusPipelineDiffList
				model.commitsKeysInput.Blur()
				model.keys.RerunStage1.SetEnabled(true)
				model.keys.RerunStage2.SetEnabled(true)
				model.keys.RerunStage3.SetEnabled(true)
			} else {
				model.activeTab = 0
				model.activePipelineStage = 0
				model.focusedElement = focusMsgInput
				model.keys.RerunStage1.SetEnabled(false)
				model.keys.RerunStage2.SetEnabled(false)
				model.keys.RerunStage3.SetEnabled(false)
				return model, model.commitsKeysInput.Focus()
			}
			return model, nil
		case key.Matches(msg, model.keys.RerunStage1):
			if model.iaSummaryOutput != "" {
				model.WritingStatusBar.Level = statusbar.LevelWarning
				model.WritingStatusBar.Content = "Re-running from Stage 1..."
				spinnerCmd := model.WritingStatusBar.StartSpinner()
				return model, tea.Batch(spinnerCmd, callIaSummaryCmd(model))
			}
			return model, nil
		case key.Matches(msg, model.keys.RerunStage2):
			if model.iaSummaryOutput != "" {
				model.WritingStatusBar.Level = statusbar.LevelWarning
				model.WritingStatusBar.Content = "Re-running from Stage 2..."
				spinnerCmd := model.WritingStatusBar.StartSpinner()
				return model, tea.Batch(spinnerCmd, callIaCommitBuilderStage2Cmd(model))
			}
			return model, nil
		case key.Matches(msg, model.keys.RerunStage3):
			if model.iaCommitRawOutput != "" {
				model.WritingStatusBar.Level = statusbar.LevelWarning
				model.WritingStatusBar.Content = "Re-running Stage 3..."
				spinnerCmd := model.WritingStatusBar.StartSpinner()
				return model, tea.Batch(spinnerCmd, callIaOutputFormatCmd(model))
			}
			return model, nil
		case key.Matches(msg, model.keys.NextField):
			if model.activeTab == 1 {
				switch model.focusedElement {
				case focusPipelineDiffList:
					model.focusedElement = focusPipelineViewport
					model.activePipelineStage = 0
				case focusPipelineViewport:
					if model.activePipelineStage < 2 {
						model.activePipelineStage++
					} else {
						model.activePipelineStage = 0
						model.focusedElement = focusPipelineDiffList
					}
				}
				return model, nil
			}
			cmd = switchFocusElement(model)
			return model, cmd
		case key.Matches(msg, model.keys.PrevField):
			if model.activeTab == 1 {
				switch model.focusedElement {
				case focusPipelineDiffList:
					model.focusedElement = focusPipelineViewport
					model.activePipelineStage = 2
				case focusPipelineViewport:
					if model.activePipelineStage > 0 {
						model.activePipelineStage--
					} else {
						model.focusedElement = focusPipelineDiffList
					}
				}
				return model, nil
			}
			cmd = switchFocusElement(model)
			return model, cmd
		case key.Matches(msg, model.keys.SaveDraft):
			if v := model.commitsKeysInput.Value(); v != "" {
				model.keyPoints = append(model.keyPoints, v)
				model.commitsKeysInput.SetValue("")
			}
			model.currentCommit.KeyPoints = model.keyPoints
			model.currentCommit.MessageEN = model.commitTranslate
			model.currentCommit.Type = model.commitType
			model.currentCommit.Scope = model.commitScope
			model.currentCommit.Workspace = model.pwd
			model.currentCommit.Diff_code = model.diffCode
			model.currentCommit.IaSummary = model.iaSummaryOutput
			model.currentCommit.IaCommitRaw = model.iaCommitRawOutput
			model.currentCommit.IaTitle = model.iaTitleRawOutput
			if err := model.db.SaveDraft(&model.currentCommit); err != nil {
				model.err = err
				return model, nil
			}
			cmd := model.WritingStatusBar.ShowMessageForDuration("Draft saved!", statusbar.LevelSuccess, 2*time.Second)
			return model, cmd
		case key.Matches(msg, model.keys.AddCommitKey):
			model.keyPoints = append(model.keyPoints, model.commitsKeysInput.Value())
			model.commitsKeysInput.SetValue("")
			cmd = model.commitsKeysInput.Focus()
			return model, cmd
		case key.Matches(msg, model.keys.Edit):
			model.WritingStatusBar.Content = "You are making modifications to the AI's response"
			model.WritingStatusBar.Level = statusbar.LevelWarning
			model.state = stateEditMessage
			model.keys = editingMessageKeys()
			model.msgEdit.SetValue(model.commitTranslate)
			return model, nil
		case key.Matches(msg, model.keys.Esc):
			return model.cancelProcess(stateChoosingScope)
		case key.Matches(msg, model.keys.Enter):
			if model.focusedElement == focusPipelineDiffList && model.activeTab == 1 {
				if item, ok := model.pipelineDiffList.SelectedItem().(DiffFileItem); ok {
					return model, fetchDiffCmd(item.FilePath)
				}
				return model, nil
			}
			if model.commitTranslate != "" {
				_, cmd := createCommit(model)
				model.useDbCommmit = false
				if model.RewordHash != "" {
					model.FinalMessage = assembleOutputCommitMessage(model, model.currentCommit)
					return model, tea.Quit
				}
				return model, cmd
			} else {
				model.WritingStatusBar.Content = "You need to first make a request to the AI to continue!!"
				model.WritingStatusBar.Level = statusbar.LevelError
				return model, nil
			}

		case key.Matches(msg, model.keys.CreateIaCommit):
			if v := model.commitsKeysInput.Value(); v != "" {
				model.keyPoints = append(model.keyPoints, v)
				model.commitsKeysInput.SetValue("")
			}
			model.WritingStatusBar.Level = statusbar.LevelWarning
			model.WritingStatusBar.Content = "Making a request to the AI. Please wait ..."
			spinnerCmd := model.WritingStatusBar.StartSpinner()
			iaBuilderCmd := callIaCommitBuilderCmd(model)
			return model, tea.Batch(spinnerCmd, iaBuilderCmd)
		case key.Matches(msg, model.keys.PgUp):
			if model.focusedElement == focusMsgInput {
				model.commitsKeysViewport, cmd = model.commitsKeysViewport.Update(msg)
				return model, cmd
			}
		case key.Matches(msg, model.keys.PgDown):
			if model.focusedElement == focusMsgInput {
				model.commitsKeysViewport, cmd = model.commitsKeysViewport.Update(msg)
				return model, cmd
			}
		}
	}
	switch model.focusedElement {
	case focusMsgInput:
		model.commitsKeysInput, cmd = model.commitsKeysInput.Update(msg)
	case focusAIResponse:
		model.iaViewport, cmd = model.iaViewport.Update(msg)
	case focusPipelineDiffList:
		model.pipelineDiffList, cmd = model.pipelineDiffList.Update(msg)
	case focusPipelineViewport:
		switch model.activePipelineStage {
		case 0:
			model.pipelineViewport1, cmd = model.pipelineViewport1.Update(msg)
		case 1:
			model.pipelineViewport2, cmd = model.pipelineViewport2.Update(msg)
		case 2:
			model.pipelineViewport3, cmd = model.pipelineViewport3.Update(msg)
		}
	}
	return model, cmd
}

