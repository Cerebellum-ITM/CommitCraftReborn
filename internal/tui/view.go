package tui

import (
	_ "embed"
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

var (
	focusColor     color.Color
	focusColorText color.Color
	blurColor      color.Color
	VerticalSpace  = lipgloss.NewStyle().Height(1).Render("")
	LineStyle      = lipgloss.NewStyle()
	HeaderStyle    = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderBottom(false).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderTop(false).
			Padding(0, 1)
)

//go:embed prompts/writing_message_instructions.md
var defaultTranslatedContentPrompt string

func (model *Model) setColorVariables(state string) (textColor, lineColor ansi.Color) {
	focusColor = model.Theme.Primary
	focusColorText = model.Theme.Accent
	blurColor = model.Theme.Blur
	if state == "focus" {
		textColor = focusColorText
		lineColor = focusColor
	} else {
		textColor = blurColor
		lineColor = blurColor
	}
	return textColor, lineColor
}

func (model *Model) iaHeaderView(state string) string {
	textColor, lineColor := model.setColorVariables(state)
	title := HeaderStyle.Foreground(textColor).Render("Final response of AI models")
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", model.iaViewport.Width()-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (model *Model) userInputHeaderView(state string) string {
	textColor, lineColor := model.setColorVariables(state)

	title := HeaderStyle.Foreground(textColor).
		Render("Enter the text with your summary of the changes")
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", model.msgInput.Width()-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Right, title, line)
}

func (model *Model) userInputFooterView(state string) string {
	textColor, lineColor := model.setColorVariables(state)
	info := FooterStyle.Foreground(textColor).Render(
		fmt.Sprintf("Number of characters %d", lipgloss.Width(model.msgInput.Value())),
	)
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", model.msgInput.Width()-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (model *Model) iaFooterView(state string) string {
	textColor, lineColor := model.setColorVariables(state)
	info := FooterStyle.Foreground(textColor).
		Render(fmt.Sprintf("%3.f%%", model.iaViewport.ScrollPercent()*100))
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", model.iaViewport.Width()-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (model *Model) buildWritingMessageView(appStyle lipgloss.Style) string {
	var (
		glamourContent             string
		iaViewHeaderContent        string
		userInputViewHeaderContent string
		iaViewFooterContent        string
		userInputiewFooterContent  string
	)

	const glamourGutter = 3
	statusBarContent := model.WritingStatusBar.Render()
	currentIaViewportStyle := model.iaViewport.Style
	currentMsgInputPromptSyle := lipgloss.NewStyle()
	switch model.focusedElement {
	case focusMsgInput:
		currentMsgInputPromptSyle = currentMsgInputPromptSyle.Foreground(model.Theme.BorderFocus)
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(model.Theme.Blur)
		iaViewHeaderContent = model.iaHeaderView("blur")
		iaViewFooterContent = model.iaFooterView("blur")
		userInputViewHeaderContent = model.userInputHeaderView("focus")
		userInputiewFooterContent = model.userInputFooterView("focus")
	case focusAIResponse:
		iaViewHeaderContent = model.iaHeaderView("focus")
		iaViewFooterContent = model.iaFooterView("focus")
		userInputViewHeaderContent = model.userInputHeaderView("blur")
		userInputiewFooterContent = model.userInputFooterView("blur")
		currentMsgInputPromptSyle = currentMsgInputPromptSyle.Foreground(model.Theme.Blur)
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(model.Theme.BorderFocus)
	}

	statusBarHeight := lipgloss.Height(model.WritingStatusBar.Render())
	verticalSpaceHeight := lipgloss.Height(VerticalSpace)
	helpViewHeight := lipgloss.Height(model.help.View(model.keys))
	iaHeaderH := lipgloss.Height(iaViewHeaderContent)
	iaFooterH := lipgloss.Height(iaViewFooterContent)
	userInputViewHeaderH := lipgloss.Height(userInputViewHeaderContent)
	userInputViewFooterH := lipgloss.Height(userInputiewFooterContent)
	iaVerticalMarginHeight := iaHeaderH + iaFooterH
	userInputViewVerticalMarginHeight := userInputViewHeaderH + userInputViewFooterH
	totalAvailableContentHeight := model.height - appStyle.GetVerticalPadding() - helpViewHeight - statusBarHeight - verticalSpaceHeight - 2

	iaViewportContentHeight := totalAvailableContentHeight - iaVerticalMarginHeight
	userInputVContenHeight := totalAvailableContentHeight - userInputViewVerticalMarginHeight
	model.iaViewport.SetHeight(iaViewportContentHeight)
	model.msgInput.SetHeight(userInputVContenHeight - 2)

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
		userInputViewHeaderContent,
		VerticalSpace,
		userInputView,
		VerticalSpace,
		userInputiewFooterContent,
	)
	rightTranslatedContent := lipgloss.JoinVertical(lipgloss.Left,
		iaViewHeaderContent,
		model.iaViewport.View(),
		iaViewFooterContent,
	)
	uiElements := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftTranslatedContent,
		rightTranslatedContent,
	)
	return lipgloss.JoinVertical(lipgloss.Left,
		statusBarContent,
		VerticalSpace,
		uiElements,
	)
}

// View renders the UI based on the current state of the model.
func (model *Model) View() string {
	var mainContent string

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
		statusBarContent := model.WritingStatusBar.Render()
		uiElements := model.mainList.View()
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			uiElements,
		)
	case stateChoosingType:
		statusBarContent := model.WritingStatusBar.Render()
		uiElements := model.commitTypeList.View()
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			uiElements,
		)

	case stateChoosingScope:
		mainContent = model.fileList.View()
	case stateWritingMessage:
		mainContent = model.buildWritingMessageView(appStyle)
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
