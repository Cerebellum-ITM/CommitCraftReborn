package tui

import (
	"github.com/charmbracelet/lipgloss/v2"
)

// View renders the UI based on the current state of the model.
func (model *Model) View() string {
	// Si el popup está activo, le delegamos el renderizado completo a él.
	// if model.popup != nil {
	// return model.popup.View()
	// }

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
	mainView := mainContent + helpView
	mainLayer := lipgloss.NewLayer(mainView)
	canvas := lipgloss.NewCanvas(mainLayer)

	if model.popup != nil {
		popupModel, ok := model.popup.(PopupModel)
		if !ok {
			return "Error: The popup is not of the expected type."
		}

		popupView := popupModel.View()
		popupWidth := lipgloss.Width(popupView)
		popupHeight := lipgloss.Height(popupView)

		// PASO C: Calcula las coordenadas para la esquina superior izquierda
		//
		//	del popup para que quede centrado.
		startX := (model.width - popupWidth) / 2
		startY := (model.height - popupHeight) / 2
		popupLayer := lipgloss.NewLayer(popupView).X(startX).Y(startY)
		canvas = lipgloss.NewCanvas(mainLayer, popupLayer)
		return canvas.Render()
	}
	return canvas.Render()
}
