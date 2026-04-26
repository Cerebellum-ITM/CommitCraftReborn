package tui

import (
	"sort"
	"strings"
	"time"

	"commit_craft_reborn/internal/commit"
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
		// Per-section key handling. These run before the global shortcut
		// handlers below so the meaning of arrow / x / e / enter changes
		// based on the currently focused section.
		switch model.focusedElement {
		case focusComposeType:
			if handled, m, c := handleTypeSectionKey(model, msg); handled {
				return m, c
			}
		case focusComposeScope:
			if handled, m, c := handleScopeSectionKey(model, msg); handled {
				return m, c
			}
		case focusComposeKeypoints:
			if handled, m, c := handleKeypointsSectionKey(model, msg); handled {
				return m, c
			}
		}

		if msg.String() == "@" && model.focusedElement == focusComposeSummary {
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
		case key.Matches(msg, keyTypePopup):
			w := max(40, model.width/2)
			h := max(12, model.height*2/3)
			model.popup = newCommitTypePopup(
				model.finalCommitTypes,
				model.globalConfig.CommitFormat.TypeFormat,
				w, h, model.Theme,
			)
			return model, nil
		case key.Matches(msg, keyScopePopup):
			startPwd := model.gitStatusData.Root
			if startPwd == "" {
				startPwd = model.pwd
			}
			w := max(50, model.width*2/3)
			h := max(18, model.height*2/3)
			model.popup = newScopePopup(
				startPwd,
				model.gitStatusData,
				model.globalConfig.TUI.UseNerdFonts,
				w, h, model.Theme,
			)
			return model, nil
		case key.Matches(msg, model.keys.NextField):
			cmd = switchFocusElement(model)
			return model, cmd
		case key.Matches(msg, model.keys.PrevField):
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
			return model.cancelProcess(stateChoosingCommit)
		case key.Matches(msg, model.keys.Enter):
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
			if len(model.commitScopes) == 0 {
				model.WritingStatusBar.Level = statusbar.LevelError
				model.WritingStatusBar.Content = "Scope is required before requesting the AI. Add at least one scope."
				return model, nil
			}
			if v := model.commitsKeysInput.Value(); v != "" {
				model.keyPoints = append(model.keyPoints, v)
				model.commitsKeysInput.SetValue("")
			}
			if len(model.keyPoints) == 0 {
				model.WritingStatusBar.Level = statusbar.LevelError
				model.WritingStatusBar.Content = "At least one key point is required before requesting the AI."
				return model, nil
			}
			model.WritingStatusBar.Level = statusbar.LevelWarning
			model.WritingStatusBar.Content = "Making a request to the AI. Please wait ..."
			spinnerCmd := model.WritingStatusBar.StartSpinner()
			iaBuilderCmd := callIaCommitBuilderCmd(model)
			return model, tea.Batch(spinnerCmd, iaBuilderCmd)
		case key.Matches(msg, model.keys.PgUp):
			if model.focusedElement == focusComposeSummary {
				model.commitsKeysViewport, cmd = model.commitsKeysViewport.Update(msg)
				return model, cmd
			}
		case key.Matches(msg, model.keys.PgDown):
			if model.focusedElement == focusComposeSummary {
				model.commitsKeysViewport, cmd = model.commitsKeysViewport.Update(msg)
				return model, cmd
			}
		}
	}
	switch model.focusedElement {
	case focusComposeSummary, focusMsgInput:
		model.commitsKeysInput, cmd = model.commitsKeysInput.Update(msg)
	case focusComposeAISuggestion, focusAIResponse:
		model.iaViewport, cmd = model.iaViewport.Update(msg)
	}
	return model, cmd
}

// handleTypeSectionKey applies the keys that are only meaningful while
// the commit-type section has focus: ←/→ to cycle through types and
// digits 1-9 as a quick jump. Returns handled=true if the key was used.
func handleTypeSectionKey(model *Model, msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	if len(model.finalCommitTypes) == 0 {
		return false, model, nil
	}
	switch msg.String() {
	case "left", "h":
		i := indexOfCommitType(model.finalCommitTypes, model.commitType)
		i = (i - 1 + len(model.finalCommitTypes)) % len(model.finalCommitTypes)
		ct := model.finalCommitTypes[i]
		model.commitType = ct.Tag
		model.commitTypeColor = ct.Color
		return true, model, nil
	case "right", "l":
		i := indexOfCommitType(model.finalCommitTypes, model.commitType)
		i = (i + 1) % len(model.finalCommitTypes)
		ct := model.finalCommitTypes[i]
		model.commitType = ct.Tag
		model.commitTypeColor = ct.Color
		return true, model, nil
	}
	return false, model, nil
}

// handleScopeSectionKey applies the keys that are only meaningful while
// the scope section has focus. Scope is single-value, so navigation keys
// are no-ops; e/Enter opens the picker and x clears the current scope.
func handleScopeSectionKey(model *Model, msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "x", "backspace", "delete":
		model.resetScopes()
		return true, model, nil
	case "e", "enter":
		startPwd := model.gitStatusData.Root
		if startPwd == "" {
			startPwd = model.pwd
		}
		w := max(50, model.width*2/3)
		h := max(18, model.height*2/3)
		model.popup = newScopePopup(
			startPwd,
			model.gitStatusData,
			model.globalConfig.TUI.UseNerdFonts,
			w, h, model.Theme,
		)
		return true, model, nil
	}
	return false, model, nil
}

// handleKeypointsSectionKey applies inline navigation/removal of saved
// key points without leaving the compose view. ↑/↓ and ←/→ both move the
// cursor; x/delete/backspace remove the highlighted point.
func handleKeypointsSectionKey(model *Model, msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	if len(model.keyPoints) == 0 {
		return false, model, nil
	}
	switch msg.String() {
	case "up", "k", "left", "h":
		model.keypointIndex = (model.keypointIndex - 1 + len(model.keyPoints)) %
			len(model.keyPoints)
		return true, model, nil
	case "down", "j", "right", "l":
		model.keypointIndex = (model.keypointIndex + 1) % len(model.keyPoints)
		return true, model, nil
	case "x", "backspace", "delete":
		i := model.keypointIndex
		if i < 0 || i >= len(model.keyPoints) {
			return true, model, nil
		}
		model.keyPoints = append(model.keyPoints[:i], model.keyPoints[i+1:]...)
		if len(model.keyPoints) == 0 {
			model.keypointIndex = 0
		} else if model.keypointIndex >= len(model.keyPoints) {
			model.keypointIndex = len(model.keyPoints) - 1
		}
		return true, model, nil
	}
	return false, model, nil
}

// indexOfCommitType finds the index of the type matching tag in the
// configured list, or 0 when the tag is missing/empty so wrapping always
// returns a valid item.
func indexOfCommitType(types []commit.CommitType, tag string) int {
	for i, ct := range types {
		if ct.Tag == tag {
			return i
		}
	}
	return 0
}

