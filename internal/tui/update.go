package tui

import (
	"commit_craft_reborn/internal/storage"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
)

func (model *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure the model is passed as a pointer.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model.width = msg.Width
		model.height = msg.Height
		listHeight := model.height - 4
		model.list.SetSize(model.width, listHeight)
		model.commitTypeList.SetSize(model.width, listHeight)

	case openPopupMsg:
		selectedItem := model.list.SelectedItem()
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

		UpdateCommitList(model.db, model.log, &model.list)
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
		// To be implemented in the next step.
	case stateWritingMessage:
		// To be implemented.
	}

	return model, nil
}

func updateChoosingType(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Help):
			model.help.ShowAll = !model.help.ShowAll
		case key.Matches(msg, model.keys.Enter):
			commitTypeSelected := model.commitTypeList.SelectedItem()
			if item, ok := commitTypeSelected.(CommitTypeItem); ok {
				newCommit := storage.Commit{
					ID:        0,
					Type:      item.Tag,
					Scope:     "user-profile",
					MessageEN: "Add user profile update functionality.",
					MessageES: "Agrega funcionalidad de actualización de perfil de usuario.",
					Workspace: "default",
					CreatedAt: time.Now(),
				}
				err := model.db.CreateCommit(newCommit)
				if err != nil {
					model.log.Error("Error saving commit from stateChoosingType", "error", err)
					model.err = err
					return model, tea.Quit
				}

				UpdateCommitList(model.db, model.log, &model.list)
				listHeight := model.height - 4
				model.list.SetSize(model.width, listHeight)
				model.state = stateChoosingCommit
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
		switch {
		case key.Matches(msg, model.keys.Quit):
			return model, tea.Quit

		case key.Matches(msg, model.keys.AddCommit):
			model.state = stateChoosingType
			return model, nil
		case key.Matches(msg, model.keys.Enter):
			selectedItem := model.list.SelectedItem()
			if commitItem, ok := selectedItem.(HistoryCommitItem); ok {
				commitMessage := commitItem.commit.MessageEN
				model.log.Info("selecting commit", "commitMessage", commitMessage)
				model.FinalMessage = fmt.Sprintf("%s", commitMessage)
			}
			return model, tea.Quit

		case key.Matches(msg, model.keys.Help):
			model.help.ShowAll = !model.help.ShowAll
		}
		// case tea.WindowSizeMsg:
		// model.list.SetSize(msg.Width, msg.Height-4)
	}

	model.list, cmd = model.list.Update(msg)
	return model, cmd
}
