package tui

import "strings"

// View renders the UI based on the current state of the model.
func (model *model) View() string {
	if model.err != nil {
		return "Error: " + model.err.Error()
	}

	// The View method changes completely depending on the state.
	switch model.state {
	case stateChoosingScope:
		return "You chose: " + model.commitType + "\n\nNow define the scope (WIP)"
	case stateWritingMessage:
		return "Message (WIP)"
	case stateTranslating:
		return "Translating (WIP)"
	case stateConfirming:
		return "Confirm (WIP)"
	case stateDone:
		return "Done! (WIP)"
	default: // stateChoosingType
		var builder strings.Builder
		builder.WriteString("Choose the commit type:\n\n")
		builder.WriteString(model.list.View())
		builder.WriteString("\n\n(q or ctrl+c to quit)")
		return builder.String()
	}
}
