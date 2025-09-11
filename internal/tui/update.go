package tui

import (
	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/components/statusbar"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	// "github.com/charmbracelet/lipgloss/v2"
)

type IaCommitBuilderResultMsg struct {
	Err error
}

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
		listHeight := model.height - 4
		model.apiKeyInput.SetWidth(model.width)
		model.mainList.SetSize(model.width, listHeight)
		model.commitTypeList.SetSize(model.width, listHeight)
		model.msgInput.SetHeight(listHeight - 2)
		model.msgInput.SetWidth(model.width / 2)
		model.iaViewport.SetHeight(listHeight)
		model.iaViewport.SetWidth(model.width / 2)
		model.msgEdit.SetHeight(listHeight - 2)
		model.msgEdit.SetWidth(model.width / 2)
		model.WritingStatusBar.AppWith = model.width

	case openPopupMsg:
		selectedItem := model.mainList.SelectedItem()
		if commitItem, ok := selectedItem.(HistoryCommitItem); ok {
			model.popup = NewPopup(model.width, model.height, commitItem.commit.ID, commitItem.commit.MessageES)
			return model, nil
		}

	case closePopupMsg:
		model.popup = nil
		return model, nil

	case deleteItemMsg:
		model.popup = nil

		err := model.db.DeleteCommit(msg.ID)
		if err != nil {
			model.err = err
			return model, nil
		}

		UpdateCommitList(model.pwd, model.db, model.log, &model.mainList)
		return model, nil
	case IaCommitBuilderResultMsg:
		cmds = append(cmds, model.WritingStatusBar.StopSpinner())

		if msg.Err != nil {
			model.err = msg.Err
			model.WritingStatusBar.Content = fmt.Sprintf("Error: %s", msg.Err.Error())
			model.WritingStatusBar.Level = statusbar.LevelError
		} else {
			model.WritingStatusBar.Content = "AI commit message ready!"
			model.WritingStatusBar.Level = statusbar.LevelInfo
		}
		model.state = stateWritingMessage
		return model, tea.Batch(cmds...)

	case tea.KeyMsg:
		if model.popup != nil {
			var popupCmd tea.Cmd
			model.popup, popupCmd = model.popup.Update(msg)
			return model, popupCmd
		}
		switch {
		case key.Matches(msg, model.keys.GlobalQuit):
			return model, tea.Quit
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
	case stateEditMessage:
		subModel, subCmd = updateEditingMessage(msg, model)
	}

	cmds = append(cmds, subCmd)
	return subModel, tea.Batch(cmds...)
}

// HELPERS

func saveAPIKeyToEnv(key string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, ".config", "CommitCraft")
	envPath := filepath.Join(configDir, ".env")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf("GROQ_API_KEY=%s\n", key)
	return os.WriteFile(envPath, []byte(content), 0600)
}

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
		model.commitScope = ""
		model.keys = mainListKeys()
	case stateChoosingType:
		statusBarMessage = "Select a prefix for the commit"
		model.commitType = ""
		model.commitScope = ""
		model.keys = listKeys()
	case stateChoosingScope:
		statusBarMessage = "choose a file or folder for your commit"
		model.commitScope = ""
		model.keys = fileListKeys()
		model.msgInput.Blur()
	}
	model.state = state
	model.WritingStatusBar.Content = statusBarMessage
	model.WritingStatusBar.Level = statusBarLevel
	return model, nil
}

func createCommit(model *Model) (tea.Model, tea.Cmd) {
	newCommit := storage.Commit{
		ID:        0,
		Type:      model.commitType,
		Scope:     model.commitScope,
		MessageES: model.commitMsg,
		MessageEN: model.commitTranslate,
		Workspace: model.pwd,
		CreatedAt: time.Now(),
	}

	err := model.db.CreateCommit(newCommit)
	if err != nil {
		model.log.Error("Error saving commit from stateChoosingType", "error", err)
		model.err = err
		return model, tea.Quit
	}

	UpdateCommitList(model.pwd, model.db, model.log, &model.mainList)
	listHeight := model.height - 4
	model.mainList.SetSize(model.width, listHeight)
	model.state = stateChoosingCommit
	model.keys = mainListKeys()
	model.WritingStatusBar.Level = statusbar.LevelSuccess
	model.WritingStatusBar.Content = "Record created in the db successfully"
	return model, nil
}

func createAndSendIaMessage(
	systemPrompt string,
	userInput string,
	iaModel string,
	model *Model,
) (string, error) {
	if iaModel == "" {
		iaModel = "llama-3.1-8b-instant"
	}
	apiKey := model.globalConfig.TUI.GroqAPIKey
	messages := []api.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userInput,
		},
	}
	response, err := api.GetGroqChatCompletion(apiKey, iaModel, messages)
	if err != nil {
		return "", fmt.Errorf(
			"An error occurred while making the following call:\n systemPrompt: %s\n userInput: %s\n Error: %s",
			systemPrompt,
			userInput,
			err,
		)
	}
	return response, nil
}

func ia_commit_builder(userInput string, model *Model) error {
	diffSummary, err := GetStagedDiffSummary(model.globalConfig.Prompts.SummaryPromptMaxDiffsize)
	promptConfig := model.globalConfig.Prompts
	// formattedCommitType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, model.commitType)
	preambleMessage := fmt.Sprintf("%s %s: ", model.commitType, model.commitScope)
	model.log.Debug("User Input", "preambleMessage", userInput)
	model.log.Debug("git diff summary", "diffSummary", diffSummary)
	if err != nil {
		return fmt.Errorf(
			"An error occurred while trying to generate the git diff summary.\n%s",
			err,
		)
	}

	iaSumarry, err := createAndSendIaMessage(
		promptConfig.SummaryPrompt,
		fmt.Sprintf("TITLE:\n%s\nCONTEXT:\n%s", userInput, diffSummary),
		promptConfig.SummaryPromptModel,
		model,
	)

	model.log.Debug("exit summary prompt", "iaSumarry", iaSumarry)
	iaCommitRawOutput, err := createAndSendIaMessage(
		promptConfig.CommitBuilderPrompt,
		iaSumarry,
		promptConfig.CommitBuilderPromptModel,
		model,
	)
	if err != nil {
		return fmt.Errorf(
			"An error occurred while trying to generate the row commit output.\n%s",
			err,
		)
	}

	iaFormattedInput := fmt.Sprintf("[PREAMBLE]: %s\n%s", preambleMessage, iaCommitRawOutput)
	model.log.Debug("output Input", "iaFormattedInput", iaFormattedInput)
	iaFormattedOutput, err := createAndSendIaMessage(
		promptConfig.OutputFormatPrompt,
		iaFormattedInput,
		promptConfig.OutputFormatPromptModel,
		model,
	)
	if err != nil {
		return fmt.Errorf(
			"An error occurred while trying to generate the formatted output.\n%s",
			err,
		)
	}

	model.log.Debug(
		"Final output of the construction process",
		"iaFormattedOutput",
		iaFormattedOutput,
	)
	model.commitMsg = userInput
	model.commitTranslate = iaFormattedOutput
	return nil
}

func callIaCommitBuilderCmd(userInput string, model *Model) tea.Cmd {
	return func() tea.Msg {
		err := ia_commit_builder(userInput, model)
		return IaCommitBuilderResultMsg{Err: err}
	}
}

func switchFocusElement(model *Model) {
	switch model.focusedElement {
	case focusMsgInput:
		model.focusedElement = focusAIResponse
	case focusAIResponse:
		model.focusedElement = focusMsgInput
	}
}

// UPDATE functions

func updateEditingMessage(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.NextField):
			switchFocusElement(model)
			return model, nil
		case key.Matches(msg, model.keys.PrevField):
			switchFocusElement(model)
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
		switch {
		case key.Matches(msg, model.keys.NextField):
			switchFocusElement(model)
			return model, nil
		case key.Matches(msg, model.keys.PrevField):
			switchFocusElement(model)
			return model, nil
		case key.Matches(msg, model.keys.Esc):
			return model.cancelProcess(stateChoosingScope)
		case key.Matches(msg, model.keys.Edit):
			model.WritingStatusBar.Content = "You are making modifications to the AI's response"
			model.WritingStatusBar.Level = statusbar.LevelWarning
			model.state = stateEditMessage
			model.keys = editingMessageKeys()
			model.msgEdit.SetValue(model.commitTranslate)
			return model, nil
		case key.Matches(msg, model.keys.Enter):
			if model.commitTranslate != "" {
				createCommit(model)
			} else {
				model.WritingStatusBar.Content = "You need to first make a request to the AI to continue!!"
				model.WritingStatusBar.Level = statusbar.LevelError
			}
			return model, nil

		case key.Matches(msg, model.keys.CreateIaCommit):
			model.WritingStatusBar.ResetContentStyle()
			model.WritingStatusBar.Level = statusbar.LevelWarning
			model.WritingStatusBar.Content = "Making a request to the AI. Please wait ..."
			spinnerCmd := model.WritingStatusBar.StartSpinner()
			userInput := model.msgInput.Value()
			iaBuilderCmd := callIaCommitBuilderCmd(userInput, model)
			return model, tea.Batch(spinnerCmd, iaBuilderCmd)
		}
	}
	switch model.focusedElement {
	case focusMsgInput:
		model.msgInput, cmd = model.msgInput.Update(msg)
	case focusAIResponse:
		model.iaViewport, cmd = model.iaViewport.Update(msg)
	}
	return model, cmd
}

func updateSettingApiKey(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	model.apiKeyInput, cmd = model.apiKeyInput.Update(msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Enter):
			apiKey := model.apiKeyInput.Value()
			if apiKey != "" {
				err := saveAPIKeyToEnv(apiKey)
				if err != nil {
					model.err = err
					return model, nil
				}
				model.globalConfig.TUI.GroqAPIKey = apiKey
				model.globalConfig.TUI.IsAPIKeySet = true

				return model.cancelProcess(stateChoosingCommit)
			}
		case key.Matches(msg, model.keys.GlobalQuit):
			return model, tea.Quit
		}
	}
	return model, cmd
}

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
		case key.Matches(msg, model.keys.Help):
			model.help.ShowAll = !model.help.ShowAll
		case key.Matches(msg, model.keys.Left):
			parentDir := filepath.Dir(scopeFilePickerPwd)
			scopeFilePickerPwd = parentDir
			model.WritingStatusBar.Level = statusbar.LevelInfo
			model.WritingStatusBar.Content = fmt.Sprintf("Choose a file or folder for your commit ::: %s", model.Theme.AppStyles().Base.Foreground(model.Theme.Tertiary).SetString(TruncatePath(scopeFilePickerPwd, 2)).String())

			UpdateFileList(parentDir, &model.fileList, model.gitStatusData)
			ResetAndActiveFilterOnList(&model.fileList)
			return model, nil
		case key.Matches(msg, model.keys.Right):
			selected := model.fileList.SelectedItem()
			if item, ok := selected.(FileItem); ok {
				if item.IsDir() {
					scopeFilePickerPwd = filepath.Join(scopeFilePickerPwd, item.Title())
					UpdateFileList(scopeFilePickerPwd, &model.fileList, model.gitStatusData)
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
				model.WritingStatusBar.Content = "Craft your commit"
				model.commitScope = item.Title()
				model.state = stateWritingMessage
				model.keys = writingMessageKeys()
				model.msgInput.Focus()
				return model, nil
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
		case key.Matches(msg, model.keys.Help):
			model.help.ShowAll = !model.help.ShowAll
		case key.Matches(msg, model.keys.Esc):
			model.keys = mainListKeys()
			return model.cancelProcess(stateChoosingCommit)
		case key.Matches(msg, model.keys.Enter):
			commitTypeSelected := model.commitTypeList.SelectedItem()
			if item, ok := commitTypeSelected.(CommitTypeItem); ok {
				model.commitType = item.Tag
				model.commitTypeColor = item.Color()
				scopeFilePickerPwd = model.pwd
				gitStatusMap, err := GetGitDiffNameStatus()
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
		isTypingInFilter := model.mainList.FilterState().String()
		if isTypingInFilter != "filtering" {
			switch {
			case key.Matches(msg, model.keys.Quit):
				return model, tea.Quit
			case key.Matches(msg, model.keys.AddCommit):
				model.WritingStatusBar.Content = "Select a prefix for the commit"
				model.state = stateChoosingType
				model.keys = listKeys()
				ResetAndActiveFilterOnList(&model.commitTypeList)
				return model, nil
			case key.Matches(msg, model.keys.Delete):
				return model, func() tea.Msg { return openPopupMsg{} }
			case key.Matches(msg, model.keys.Enter):
				selectedItem := model.mainList.SelectedItem()
				if commitItem, ok := selectedItem.(HistoryCommitItem); ok {
					commit := commitItem.commit
					// model.log.Info("selecting commit", "commitMessage", commit.Type, commit.Scope, commit.MessageEN)
					formattedCommitType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, commit.Type)
					model.FinalMessage = fmt.Sprintf("%s %s: %s", formattedCommitType, commit.Scope, commit.MessageEN)
				}
				return model, tea.Quit

			case key.Matches(msg, model.keys.Help):
				model.help.ShowAll = !model.help.ShowAll
			}
		}
	}

	model.mainList, cmd = model.mainList.Update(msg)
	return model, cmd
}
