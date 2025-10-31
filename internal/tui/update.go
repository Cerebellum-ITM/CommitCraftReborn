package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/atotto/clipboard"
)

type IaCommitBuilderResultMsg struct {
	Err error
}

type IaResleaseBuilderResultMsg struct {
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
		model.apiKeyInput.SetWidth(model.width)
		model.WritingStatusBar.AppWith = model.width
	case openPopupMsg:
		switch msg.Type {
		case Confirmation:
			switch msg.Db {
			case commitDb:
				selectedItem := model.mainList.SelectedItem()
				if commitItem, ok := selectedItem.(HistoryCommitItem); ok {
					model.popup = NewPopup(model.width, model.height, commitItem.commit.ID, commitItem.commit.MessageES, commitDb, WithColor(model.Theme.Warning), WithTheme(model.Theme))
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
	case releaseAction:
		model.popup = nil

		switch msg.action {
		case "Create":
			_, cmd := createRelease(model)
			return model, cmd
		case "Print in console":
			if selectedItem, ok := model.releaseMainList.SelectedItem().(HistoryReleaseItem); ok {
				formattedReleaseType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, selectedItem.release.Type)
				model.FinalMessage = fmt.Sprintf("%s %s: %s\n%s", formattedReleaseType, selectedItem.release.Branch, selectedItem.release.Title, selectedItem.release.Body)
			}
			return model, tea.Quit
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
		case "Release Commit":
			model.releaseType = "REL"
			branch, err := GetCurrentGitBranch()
			if err != nil {
				model.log.Error("Error getting the current branch", "error", err)
				model.err = err
				return model, tea.Quit
			}
			model.releaseBranch = branch
			return model, func() tea.Msg { return releaseAction{action: "Create"} }
		case "Merge Commit":
			model.releaseType = "MERGE"
			branches, err := GetGitBranches()
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
		} else {
			model.WritingStatusBar.Content = "AI commit message ready!"
			model.WritingStatusBar.Level = statusbar.LevelInfo
		}
		model.state = stateWritingMessage
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

	case tea.KeyMsg:
		if model.popup != nil {
			var popupCmd tea.Cmd
			model.popup, popupCmd = model.popup.Update(msg)
			return model, popupCmd
		}
		switch {
		case key.Matches(msg, model.keys.GlobalQuit):
			return model, tea.Quit
		case key.Matches(msg, model.keys.Help):
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
	case stateEditMessage:
		subModel, subCmd = updateEditingMessage(msg, model)
	case stateReleaseChoosingCommits:
		subModel, subCmd = updateReleaseChoosingCommits(msg, model)
	case stateReleaseBuildingText:
		subModel, subCmd = updateReleaseBuildingText(msg, model)
	case stateReleaseMainMenu:
		subModel, subCmd = updateReleaseMainMenu(msg, model)
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

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf("GROQ_API_KEY=%s\n", key)
	return os.WriteFile(envPath, []byte(content), 0o600)
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
	newCommit := storage.Commit{
		ID:        0,
		Type:      model.commitType,
		Scope:     model.commitScope,
		MessageES: model.commitMsg,
		MessageEN: model.commitTranslate,
		Workspace: model.pwd,
		Diff_code: model.diffCode,
		CreatedAt: time.Now(),
	}

	err := model.db.CreateCommit(newCommit)
	if err != nil {
		model.log.Error("Error saving commit from stateChoosingType", "error", err)
		model.err = err
		return model, tea.Quit
	}

	UpdateCommitList(model.pwd, model.db, model.log, &model.mainList, commitDb)
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
	var diffSummary string
	var err error

	if model.useDbCommmit {
		diffSummary = model.diffCode
	} else {
		diffSummary, err = GetStagedDiffSummary(model.globalConfig.Prompts.SummaryPromptMaxDiffsize)
		if err != nil {
			return fmt.Errorf(
				"An error occurred while trying to generate the git diff summary.\n%s",
				err,
			)
		}

	}
	promptConfig := model.globalConfig.Prompts
	// formattedCommitType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, model.commitType)
	preambleMessage := fmt.Sprintf("%s %s: ", model.commitType, model.commitScope)
	model.log.Debug("User Input", "preambleMessage", userInput)
	model.log.Debug("git diff summary", "diffSummary", diffSummary)

	iaSumarry, err := createAndSendIaMessage(
		promptConfig.SummaryPrompt,
		fmt.Sprintf("TITLE:\n%s\nCONTEXT:\n%s", userInput, diffSummary),
		promptConfig.SummaryPromptModel,
		model,
	)

	model.log.Debug("exit summary prompt", "iaSumarry", iaSumarry)
	iaCommitRawOutput, err := createAndSendIaMessage(
		promptConfig.CommitBuilderPrompt,
		fmt.Sprintf("COMMIT_TYPE:\n%s\nSUMMARY:\n%s", model.commitType, iaSumarry),
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
	model.diffCode = diffSummary
	return nil
}

func callIaCommitBuilderCmd(userInput string, model *Model) tea.Cmd {
	return func() tea.Msg {
		err := ia_commit_builder(userInput, model)
		return IaCommitBuilderResultMsg{Err: err}
	}
}

func callIaReleaseBuilderCmd(model *Model) tea.Cmd {
	return func() tea.Msg {
		err := iaReleaseBuilder(model)
		return IaResleaseBuilderResultMsg{Err: err}
	}
}

func iaReleaseBuilder(model *Model) error {
	var input strings.Builder
	delimiter := "--- COMMIT SEPARATOR ---"
	for _, item := range model.selectedCommitList {
		commitContent := fmt.Sprintf(
			"%s\nCommit.Date:%s\nCommit.Title:%s\ncommit.body:%s\n%s\n",
			delimiter,
			item.Date,
			item.Subject,
			item.Body,
			delimiter,
		)
		input.WriteString(commitContent)
	}
	promptConfig := model.globalConfig.Prompts
	model.log.Debug("release ia Input", "input", input)

	iaResponse, err := createAndSendIaMessage(
		promptConfig.ReleasePrompt,
		input.String(),
		promptConfig.ReleasePromptModel,
		model,
	)
	if err != nil {
		return fmt.Errorf(
			"An error occurred while trying to generate the release output.\n%s",
			err,
		)
	}
	model.commitLivePreview = iaResponse
	model.releaseText = iaResponse
	return nil
}

func switchFocusElement(model *Model) {
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
	case focusAIResponse:
		model.focusedElement = focusMsgInput
	}
}

// UPDATE functions
func updateReleaseMainMenu(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
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
		case key.Matches(msg, model.keys.Enter):
			var menuOptions []itemsOptions
			menu := []string{"Print in console", "Copy to clipboard"}
			menuOptions = append(menuOptions, itemsOptions{index: 0, color: model.Theme.Success, icon: model.Theme.AppSymbols().Console})
			menuOptions = append(menuOptions, itemsOptions{index: 1, color: model.ToolsInfo.xclip.textColor, icon: model.ToolsInfo.xclip.icon})
			return model, func() tea.Msg {
				return openListPopup{items: menu, itemsOptions: menuOptions, width: model.width / 2, height: model.height / 2, color: model.Theme.Success}
			}
		case key.Matches(msg, model.keys.Delete):
			return model, func() tea.Msg { return openPopupMsg{Type: Confirmation, Db: releaseDb} }
		case key.Matches(msg, model.keys.SwitchMode):
			model.AppMode = CommitMode
			model.state = stateChoosingCommit
			model.keys = mainListKeys()
			model.WritingStatusBar.Content = fmt.Sprintf(
				"choose, create, or edit a commit ::: %s",
				model.Theme.AppStyles().
					Base.Foreground(model.Theme.Tertiary).
					SetString(model.mainList.Title),
			)
			cmd = model.WritingStatusBar.ShowMessageForDuration(
				"Change app mode: Commit",
				statusbar.LevelWarning,
				2*time.Second,
			)
			return model, cmd

		}
	}

	model.releaseMainList, cmd = model.releaseMainList.Update(msg)
	return model, cmd
}

func updateReleaseBuildingText(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch model.focusedElement {
	case focusViewportElement:
		model.releaseViewport, cmd = model.releaseViewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Enter):
			menu := []string{"Create item in CommitCraft", "Create release in Github"}
			return model, func() tea.Msg { return openListPopup{items: menu, width: model.width / 2, height: model.height / 2} }
		case key.Matches(msg, model.keys.NextField):
			switchFocusElement(model)
			model.state = stateReleaseChoosingCommits
			return model, nil
		case key.Matches(msg, model.keys.PrevField):
			switchFocusElement(model)
			model.state = stateReleaseChoosingCommits
			return model, nil
		}
	}

	return model, cmd
}

func updateReleaseChoosingCommits(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch model.focusedElement {
	case focusListElement:
		model.releaseCommitList, cmd = model.releaseCommitList.Update(msg)
	case focusViewportElement:
		model.releaseViewport, cmd = model.releaseViewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.NextViewPort):
			if model.releaseViewState.releaseCreated {
				model.state = stateReleaseBuildingText
				model.focusedElement = focusViewportElement
				model.WritingStatusBar.Content = "Release creation"
				model.WritingStatusBar.Level = statusbar.LevelInfo
				model.commitLivePreview = model.releaseText
			}
			return model, nil
		case key.Matches(msg, model.keys.Enter):
			model.state = stateReleaseBuildingText
			model.focusedElement = focusViewportElement
			model.WritingStatusBar.Level = statusbar.LevelWarning
			model.WritingStatusBar.Content = "Making a request to the AI. Please wait ..."
			spinnerCmd := model.WritingStatusBar.StartSpinner()
			iaBuilderCmd := callIaReleaseBuilderCmd(model)
			model.releaseViewState.releaseCreated = true
			return model, tea.Batch(spinnerCmd, iaBuilderCmd)
		case key.Matches(msg, model.keys.AddCommit):
			item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem)
			if !ok {
				return model, nil
			}
			if item.Selected {
				item.Selected = false
				foundIndex := -1
				for i, r := range model.selectedCommitList {
					if r.Hash == item.Hash {
						foundIndex = i
						break
					}
				}
				model.selectedCommitList = append(model.selectedCommitList[:foundIndex], model.selectedCommitList[foundIndex+1:]...)
			} else {
				item.Selected = true
				model.selectedCommitList = append(model.selectedCommitList, item)
			}
			index := model.releaseCommitList.Index()
			cmd = model.releaseCommitList.SetItem(index, item)
			return model, cmd
		case key.Matches(msg, model.keys.Up, model.keys.Down):
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
		case key.Matches(msg, model.keys.NextField):
			switchFocusElement(model)
			return model, nil
		case key.Matches(msg, model.keys.PrevField):
			switchFocusElement(model)
			return model, nil
		case key.Matches(msg, model.keys.Esc):
			switch model.AppMode {
			case CommitMode:
				model.state = stateChoosingCommit
				model.keys = mainListKeys()
			case ReleaseMode:
				model.state = stateReleaseMainMenu
				model.keys = releaseMainListKeys()
			}
			return model, nil
		}
	}

	return model, cmd
}

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
				_, cmd := createCommit(model)
				model.useDbCommmit = false
				return model, cmd
			} else {
				model.WritingStatusBar.Content = "You need to first make a request to the AI to continue!!"
				model.WritingStatusBar.Level = statusbar.LevelError
				return model, nil
			}

		case key.Matches(msg, model.keys.CreateIaCommit):
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
	var nextState appState

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

				switch model.AppMode {
				case ReleaseMode:
					nextState = stateReleaseMainMenu
				case CommitMode:
					nextState = stateChoosingCommit
				}
				return model.cancelProcess(nextState)
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
				model.commitScope = item.Title()
				model.state = stateWritingMessage
				model.focusedElement = focusMsgInput
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
					model.commitScope = commit.Scope
					model.commitType = commit.Type
					model.diffCode = commit.Diff_code
					model.commitMsg = commit.MessageES
					model.commitTranslate = commit.MessageEN
					model.useDbCommmit = true
					model.msgInput.SetValue(commit.MessageES)
					model.state = stateWritingMessage
					model.keys = writingMessageKeys()
				}
				return model, nil

			case key.Matches(msg, model.keys.Delete):
				return model, func() tea.Msg { return openPopupMsg{Type: Confirmation, Db: commitDb} }
			case key.Matches(msg, model.keys.CreateLocalTomlConfig):
				CreateLocalConfigTomlTmpl()
				cmd := model.WritingStatusBar.ShowMessageForDuration("Configuration file created!", statusbar.LevelSuccess, 2*time.Second)
				return model, cmd
			case key.Matches(msg, model.keys.Enter):
				selectedItem := model.mainList.SelectedItem()
				if commitItem, ok := selectedItem.(HistoryCommitItem); ok {
					commit := commitItem.commit
					formattedCommitType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, commit.Type)
					model.FinalMessage = fmt.Sprintf("%s %s: %s", formattedCommitType, commit.Scope, commit.MessageEN)
				}
				return model, tea.Quit
			case key.Matches(msg, model.keys.SwitchMode):
				model.AppMode = ReleaseMode
				model.state = stateReleaseMainMenu
				model.keys = releaseMainListKeys()
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
				return model, cmd
			}
		}
	}

	model.mainList, cmd = model.mainList.Update(msg)
	return model, cmd
}
