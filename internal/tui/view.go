package tui

import (
	"fmt"

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
		currentIaViewportStyle := model.iaViewport.Style
		currentMsgInputPromptSyle := lipgloss.NewStyle()
		switch model.focusedElement {
		case focusMsgInput:
			currentMsgInputPromptSyle = currentMsgInputPromptSyle.Foreground(lipgloss.BrightGreen)
			currentIaViewportStyle = currentIaViewportStyle.BorderForeground(lipgloss.Black)
		case focusAIResponse:
			model.msgInput.Styles.Focused.Base = model.msgInput.Styles.Focused.Base.BorderForeground(
				lipgloss.Black,
			)
			currentMsgInputPromptSyle = currentMsgInputPromptSyle.Foreground(lipgloss.BrightWhite)
			currentIaViewportStyle = currentIaViewportStyle.BorderForeground(lipgloss.BrightGreen)
		}

		model.iaViewport.Style = currentIaViewportStyle
		model.msgInput.Prompt = currentMsgInputPromptSyle.Render("┃ ")

		selectedCommitTypeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(model.commitTypeColor))
		selectScopeStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightYellow)
		selectedDetails := fmt.Sprintf(
			"Type: %s, Scope: %s",
			selectedCommitTypeStyle.Render(model.commitType),
			selectScopeStyle.Render(model.commitScope),
		)
		userInputView := lipgloss.JoinVertical(lipgloss.Left,
			model.WritingStatusBar.Render(),
			model.msgInput.View(),
		)
		const (
			glamourGutter = 3
		)

		glamourRenderWidth := model.iaViewport.Width() - model.iaViewport.Style.GetHorizontalFrameSize() - glamourGutter
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

		glamourContentStr, _ := renderer.Render(
			fmt.Sprintf("# Final response of AI models\n%s", glamourContent),
		)
		translatedView := glamourContentStr
		model.iaViewport.SetContent(translatedView)
		leftTranslatedContent := lipgloss.JoinVertical(lipgloss.Left,
			selectedDetails,
			lipgloss.NewStyle().Height(1).Render(""),
			userInputView,
			lipgloss.NewStyle().Height(1).Render(""),
		)
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftTranslatedContent,
			model.iaViewport.View(),
		)
	case stateConfirming:
		mainContent = "Confirm (WIP)"
	case stateDone:
		mainContent = "Done! (WIP)"
	}

	helpView := fmt.Sprintf("  %s", model.help.View(model.keys))
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
