package tui

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss/v2"
)

var (
	VerticalSpace = lipgloss.NewStyle().Height(1).Render("")
	iaHeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBottom(false).
			Padding(0, 1).
			Foreground(lipgloss.Color("12"))

	iaFooterStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderTop(false).
			Padding(0, 1).
			Foreground(lipgloss.Color("8"))
)

//go:embed prompts/writing_message_instructions.md
var defaultTranslatedContentPrompt string

func (model *Model) iaHeaderView() string {
	title := iaHeaderStyle.Render("Final response of AI models")
	line := strings.Repeat("─", model.iaViewport.Width()-lipgloss.Width(title))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (model *Model) iaFooterView() string {
	info := iaFooterStyle.Render(fmt.Sprintf("%3.f%%", model.iaViewport.ScrollPercent()*100))
	line := strings.Repeat("─", model.iaViewport.Width()-lipgloss.Width(info))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

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
		const glamourGutter = 3
		statusBarContent := model.WritingStatusBar.Render()
		currentIaViewportStyle := model.iaViewport.Style
		currentMsgInputPromptSyle := lipgloss.NewStyle()

		statusBarHeight := lipgloss.Height(model.WritingStatusBar.Render())
		verticalSpaceHeight := lipgloss.Height(VerticalSpace)
		helpViewHeight := lipgloss.Height(model.help.View(model.keys))
		iaHeaderH := lipgloss.Height(model.iaHeaderView())
		iaFooterH := lipgloss.Height(model.iaFooterView())
		iaVerticalMarginHeight := iaHeaderH + iaFooterH
		totalAvailableContentHeight := model.height - appStyle.GetVerticalPadding() - helpViewHeight - statusBarHeight - verticalSpaceHeight - 2

		iaViewportContentHeight := totalAvailableContentHeight - iaVerticalMarginHeight
		model.iaViewport.SetHeight(iaViewportContentHeight)

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
		userInputView := lipgloss.JoinVertical(lipgloss.Left,
			model.msgInput.View(),
		)
		glamourRenderWidth := model.iaViewport.Width() - model.iaViewport.Style.GetHorizontalFrameSize() - glamourGutter
		glamourStyle := styles.DarkStyleConfig
		renderer, _ := glamour.NewTermRenderer(
			glamour.WithStyles(glamourStyle),
			glamour.WithWordWrap(glamourRenderWidth),
		)
		if model.commitTranslate == "" {
			glamourContent = defaultTranslatedContentPrompt
		} else {
			glamourContent = model.commitTranslate
		}

		glamourContentStr, _ := renderer.Render(glamourContent)
		translatedView := glamourContentStr
		model.iaViewport.SetContent(translatedView)
		leftTranslatedContent := lipgloss.JoinVertical(lipgloss.Left,
			userInputView,
		)
		rightTranslatedContent := lipgloss.JoinVertical(lipgloss.Left,
			model.iaHeaderView(),
			model.iaViewport.View(),
			model.iaFooterView(),
		)
		uiElements := lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftTranslatedContent,
			rightTranslatedContent,
		)
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			uiElements,
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
		VerticalSpace,
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
