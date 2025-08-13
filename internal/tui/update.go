package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
)

func (model *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure the model is passed as a pointer.
	switch msg := msg.(type) {
	// Global key handling for quitting.
	case tea.WindowSizeMsg:
		model.width = msg.Width
		model.height = msg.Height

	case openPopupMsg:
		model.popup = NewPopup(model.width, model.height)
		return model, nil
	case closePopupMsg:
		model.popup = nil
		return model, nil

	case tea.KeyMsg:
		if model.popup != nil {
			var popupCmd tea.Cmd
			model.popup, popupCmd = model.popup.Update(msg)
			return model, popupCmd
		}

		// if model.logViewVisible {
		// switch {
		// Si la tecla es 'ctrl+l' o 'esc', cerramos la ventana.
		// (Asumiendo que tienes 'Esc' definido en tu KeyMap, si no, puedes añadirlo)
		// case key.Matches(msg, model.keys.Logs), key.Matches(msg, model.keys.Esc):
		// model.toggleLogView()
		// return model, func() tea.Msg { return closePopupMsg{} }
		// }
		// Ignoramos cualquier otra tecla y evitamos que se procese más abajo.
		// return model, nil
		// }
		switch {
		case key.Matches(msg, model.keys.GlobalQuit):
			return model, tea.Quit
		case key.Matches(msg, model.keys.Logs):
			return model, func() tea.Msg { return openPopupMsg{} }
			// model.toggleLogView()
			// return model, nil
		}
	}

	// Update logic depends on the current state.
	switch model.state {
	case stateChoosingType:
		return updateChoosingType(msg, model)
	case stateChoosingScope:
		// To be implemented in the next step.
	case stateWritingMessage:
		// To be implemented.
	}

	return model, nil
}

func (m *Model) toggleLogView() {
	m.logViewVisible = !m.logViewVisible
	if m.logViewVisible {
		m.logViewport.GotoBottom()
	}
}

// updateChoosingType handles the logic for the type-choosing state.
func updateChoosingType(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle logic specific to this state first.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Quit):
			return model, tea.Quit

		case key.Matches(msg, model.keys.Enter):
			selectedItem := model.list.SelectedItem()
			if commitItem, ok := selectedItem.(CommitItem); ok {
				// commitID := commitItem.commit.ID
				commitMessage := commitItem.commit.MessageEN
				model.log.Info("selecting commit", "commitMessage", commitMessage)
				model.FinalMessage = fmt.Sprintf("%s", commitMessage)
			}
			return model, tea.Quit

		case key.Matches(msg, model.keys.Help):
			model.help.ShowAll = !model.help.ShowAll
		}
	case tea.WindowSizeMsg:
		model.list.SetSize(msg.Width, msg.Height-4)
	}

	// Then, pass all messages to the list's update method for its internal handling.
	model.list, cmd = model.list.Update(msg)
	return model, cmd
}
