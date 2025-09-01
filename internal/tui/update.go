package tui

import (
	"commit_craft_reborn/internal/storage"
	"fmt"
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
		case key.Matches(msg, model.keys.Delete):
			return model, func() tea.Msg { return openPopupMsg{} }
		}
	}

	// Update logic depends on the current state.
	switch model.state {
	case stateChoosingCommit:
		return updateChoosingCommit(msg, model)
	case stateChoosingType:
		return updateChoosingType(msg, model)

	case stateWritingMessage:
		// To be implemented.
	}

	return model, nil
}

func cancelProcess(model *Model) (tea.Model, tea.Cmd) {
	model.state = stateChoosingCommit
	model.commitMsg = ""
	model.commitType = ""
	model.commitTranslate = ""
	return model, nil
}

func createCommit(model *Model) (tea.Model, tea.Cmd) {
	newCommit := storage.Commit{
		ID:        0,
		Type:      model.commitType,
		Scope:     "user-profile",
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
	statusMenssageStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightYellow)
	model.mainList.NewStatusMessage(
		statusMenssageStyle.Render("record created in the db successfully"),
	)
	return model, nil
}
func updateChoosingType(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Help):
			model.help.ShowAll = !model.help.ShowAll
		case key.Matches(msg, model.keys.Esc):
			cancelProcess(model)

		case key.Matches(msg, model.keys.Enter):
			commitTypeSelected := model.commitTypeList.SelectedItem()
			if item, ok := commitTypeSelected.(CommitTypeItem); ok {
				model.commitType = item.Tag
				model.state = stateChoosingScope
				return model, model.filePicker.Init()
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
		switch {
		case key.Matches(msg, model.keys.Quit):
			return model, tea.Quit

		case key.Matches(msg, model.keys.AddCommit):
			model.state = stateChoosingType
			return model, nil
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
		// case tea.WindowSizeMsg:
		// model.mainList.SetSize(msg.Width, msg.Height-4)
	}

	model.mainList, cmd = model.mainList.Update(msg)
	return model, cmd
}
