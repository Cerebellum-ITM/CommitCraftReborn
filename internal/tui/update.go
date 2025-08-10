package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (model *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure the model is passed as a pointer.
	switch msg := msg.(type) {
	// Global key handling for quitting.
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return model, tea.Quit
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

// updateChoosingType handles the logic for the type-choosing state.
func updateChoosingType(msg tea.Msg, model *model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle logic specific to this state first.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return model, tea.Quit
		}
		if msg.String() == "enter" {
			// Save the selection and switch state.
			model.commitType = model.list.SelectedItem().(item).title
			model.state = stateChoosingScope
			return model, nil
		}
	case tea.WindowSizeMsg:
		// Ensure the list is resized.
		model.list.SetSize(msg.Width, msg.Height-4)
	}

	// Then, pass all messages to the list's update method for its internal handling.
	model.list, cmd = model.list.Update(msg)
	return model, cmd
}
