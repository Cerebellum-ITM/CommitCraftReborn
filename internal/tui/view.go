package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss/v2"
)

// View renders the UI based on the current state of the model.
func (model *Model) View() string {
	var mainContent string
	if model.err != nil {
		return "Error: " + model.err.Error()
	}

	switch model.state {
	case stateSettingAPIKey:
		mainContent = fmt.Sprintf(
			"  Enter your Groq API Key:\n\n  %s\n\n  (Press Enter to save)",
			model.apiKeyInput.View(),
		)
	case stateChoosingCommit:
		mainContent = model.mainList.View()
	case stateChoosingType:
		mainContent = model.commitTypeList.View()
	case stateChoosingScope:
		mainContent = model.fileList.View()
	case stateWritingMessage:
		mainContent = model.msgInput.View()
	case stateConfirming:
		mainContent = "Confirm (WIP)"
	case stateDone:
		mainContent = "Done! (WIP)"
	}

	helpView := model.help.View(model.keys)
	mainView := mainContent + "\n  " + helpView
	mainLayer := lipgloss.NewLayer(mainView)
	canvas := lipgloss.NewCanvas(mainLayer)

	if model.popup != nil {
		popupModel, ok := model.popup.(DeleteConfirmPopupModel)
		if !ok {
			return "Error: The popup is not of the expected type."
		}

		popupView := popupModel.View()
		startX, startY := calculatePopupPosition(model.width, model.height, popupView)
		popupLayer := lipgloss.NewLayer(popupView).X(startX).Y(startY)
		canvas = lipgloss.NewCanvas(mainLayer, popupLayer)
		return canvas.Render()
	}
	return canvas.Render()
}
