package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss/v2"
)

// View renders the UI based on the current state of the model.
func (model *Model) View() string {
	var mainContent string
	var glamourContent string

	appStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(model.width - 4).
		Height(model.height - 2)

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
		selectedDetails := lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Padding(0, 1).
			Render(fmt.Sprintf("Type: %s, Scope: %s", model.commitType, model.commitScope))
		inputLabel := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Render("Tu Mensaje de Commit:")
		userInputView := lipgloss.JoinVertical(lipgloss.Left,
			inputLabel,
			model.msgInput.View(),
		)

		const (
			glamourGutter = 3
			width         = 78
			height        = 20
		)

		vp := viewport.New()
		vp.SetWidth(width)
		vp.SetHeight(height)
		vp.Style = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			PaddingRight(2)

		translationLabel := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")). // Verde para la traducción
			Render("Mensaje Traducido por la IA:")

		glamourRenderWidth := width - vp.Style.GetHorizontalFrameSize() - glamourGutter
		glamourStyle := styles.DarkStyleConfig
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithStyles(glamourStyle),
			glamour.WithWordWrap(glamourRenderWidth),
		)
		if model.commitTranslate == "" {
			glamourContent = "There is no content to show yet"
		} else {
			glamourContent = model.commitTranslate
		}

		glamourContentStr, _ := renderer.Render(glamourContent)
		translatedView := lipgloss.JoinVertical(
			lipgloss.Left,
			translationLabel,
			lipgloss.NewStyle().Height(1).Render(""),
			glamourContentStr,
		)

		leftTranslatedContent := lipgloss.JoinVertical(lipgloss.Left,
			selectedDetails,
			lipgloss.NewStyle().Height(1).Render(""),
			userInputView,
			lipgloss.NewStyle().Height(1).Render(""),
			translatedView,
		)
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, leftTranslatedContent, translatedView)
	case stateConfirming:
		mainContent = "Confirm (WIP)"
	case stateDone:
		mainContent = "Done! (WIP)"
	}

	helpView := model.help.View(model.keys)
	contentHeight := model.height - lipgloss.Height(helpView) - appStyle.GetVerticalPadding() - 2
	mainContent = lipgloss.NewStyle().
		Height(contentHeight).
		MaxHeight(contentHeight).
		Render(mainContent)

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		mainContent,
		helpView,
	)
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
