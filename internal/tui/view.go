package tui



// View renders the UI based on the current state of the model.
func (model *Model) View() string {
	var mainContent string
	if model.err != nil {
		return "Error: " + model.err.Error()
	}

	switch model.state {
	case stateChoosingType:
		mainContent = model.list.View()
	case stateChoosingScope:
		mainContent = "You chose: " + model.commitType + "\n\nNow define the scope (WIP)"
	case stateWritingMessage:
		mainContent = "Message (WIP)"
	case stateTranslating:
		mainContent = "Translating (WIP)"
	case stateConfirming:
		mainContent = "Confirm (WIP)"
	case stateDone:
		mainContent = "Done! (WIP)"
	}

	helpView := model.help.View(model.keys)
	return mainContent + "\n" + helpView
}
