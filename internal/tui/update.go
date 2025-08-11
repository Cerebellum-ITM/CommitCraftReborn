package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (model *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
func updateChoosingType(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle logic specific to this state first.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Quit):
			return model, tea.Quit

		case key.Matches(msg, model.keys.Enter):
			model.FinalMessage = model.list.SelectedItem().(Item).title
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
