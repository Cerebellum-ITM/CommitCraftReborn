package tui

import (
	"commit_craft_reborn/internal/storage"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

func (model *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure the model is passed as a pointer.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model.width = msg.Width
		model.height = msg.Height
		listHeight := model.height - 4
		model.mainList.SetSize(model.width, listHeight)
		model.commitTypeList.SetSize(model.width, listHeight)
		model.fileList.SetSize(model.width, listHeight)

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
	switch model.state {
	case stateSettingAPIKey:
		return updateSettingApiKey(msg, model)
	case stateChoosingCommit:
		return updateChoosingCommit(msg, model)
	case stateChoosingType:
		return updateChoosingType(msg, model)
	case stateChoosingScope:
		return updateChoosingScope(msg, model)

	case stateWritingMessage:
		// To be implemented.
	}

	return model, nil
}

func saveAPIKeyToEnv(key string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, ".config", "commitcraft")
	envPath := filepath.Join(configDir, ".env")

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf("GROQ_API_KEY=%s\n", key)
	return os.WriteFile(envPath, []byte(content), 0600)
}

func cancelProcess(model *Model) (tea.Model, tea.Cmd) {
	model.state = stateChoosingCommit
	model.commitMsg = ""
	model.commitType = ""
	model.commitTranslate = ""
	model.commitScope = ""
	return model, nil
}

func createCommit(model *Model) (tea.Model, tea.Cmd) {
	newCommit := storage.Commit{
		ID:        0,
		Type:      model.commitType,
		Scope:     model.commitScope,
		MessageEN: "Add user profile update functionality.",
		MessageES: "Agrega funcionalidad de actualización de perfil de usuario.",
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
	model.keys = listKeys()
	statusMenssageStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightYellow)
	model.mainList.NewStatusMessage(
		statusMenssageStyle.Render("record created in the db successfully"),
	)
	return model, nil
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

				model.state = stateChoosingCommit
				return model, nil
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
		switch {
		case key.Matches(msg, model.keys.Esc):
			return cancelProcess(model)
		case key.Matches(msg, model.keys.Help):
			model.help.ShowAll = !model.help.ShowAll
		case key.Matches(msg, model.keys.Left):
			parentDir := filepath.Dir(scopeFilePickerPwd)
			scopeFilePickerPwd = parentDir
			UpdateFileList(parentDir, &model.fileList)
			model.fileList.ResetFilter()
			return model, nil
		case key.Matches(msg, model.keys.Right):
			selected := model.fileList.SelectedItem()
			if item, ok := selected.(FileItem); ok {
				if item.IsDir() {
					scopeFilePickerPwd = filepath.Join(scopeFilePickerPwd, item.Title())
					UpdateFileList(scopeFilePickerPwd, &model.fileList)
					model.fileList.ResetFilter()
				} else {
					statusMenssageStyle := lipgloss.NewStyle().Foreground(lipgloss.Red)
					model.fileList.NewStatusMessage(
						statusMenssageStyle.Render("The selected item is not a directory"))
				}
				return model, nil
			}
		case key.Matches(msg, model.keys.Enter):
			commitScopeSelected := model.fileList.SelectedItem()
			if item, ok := commitScopeSelected.(FileItem); ok {
				model.commitScope = item.Title()
				return createCommit(model)
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
		switch {
		case key.Matches(msg, model.keys.Help):
			model.help.ShowAll = !model.help.ShowAll
		case key.Matches(msg, model.keys.Esc):
			return cancelProcess(model)
		case key.Matches(msg, model.keys.Enter):
			commitTypeSelected := model.commitTypeList.SelectedItem()
			if item, ok := commitTypeSelected.(CommitTypeItem); ok {
				model.commitType = item.Tag
				scopeFilePickerPwd = model.pwd
				model.state = stateChoosingScope
				model.keys = fileListKeys()
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

	// Handle logic specific to this state first.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		isTypingInFilter := model.mainList.FilterState().String()
		if isTypingInFilter != "filtering" {
			switch {
			case key.Matches(msg, model.keys.Quit):
				return model, tea.Quit
			case key.Matches(msg, model.keys.AddCommit):
				model.state = stateChoosingType
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
		// case tea.WindowSizeMsg:
		// model.mainList.SetSize(msg.Width, msg.Height-4)
	}

	model.mainList, cmd = model.mainList.Update(msg)
	return model, cmd
}
