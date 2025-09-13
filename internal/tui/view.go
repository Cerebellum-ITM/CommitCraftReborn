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
		Render(strings.Repeat("─", max(0, model.iaViewport.Width()-lipgloss.Width(title))))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (model *Model) userInputHeaderView(state string) string {
	textColor, lineColor := model.setColorVariables(state)
	title := HeaderStyle.Foreground(textColor).
		Render("Enter the text with your summary of the changes")
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", max(0, model.iaViewport.Width()-lipgloss.Width(title))))
	return lipgloss.JoinHorizontal(lipgloss.Right, title, line)
}

func (model *Model) userInputFooterView(state string) string {
	textColor, lineColor := model.setColorVariables(state)
	info := FooterStyle.Foreground(textColor).Render(
		fmt.Sprintf("Number of characters %d", lipgloss.Width(model.msgInput.Value())),
	)
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", max(0, model.iaViewport.Width()-lipgloss.Width(info))))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (model *Model) msgEditHeaderView(state string) string {
	textColor, lineColor := model.setColorVariables(state)
	title := HeaderStyle.Foreground(textColor).
		Render("Write the modifications")
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", max(0, model.iaViewport.Width()-lipgloss.Width(title))))
	return lipgloss.JoinHorizontal(lipgloss.Right, title, line)
}

func (model *Model) msgEditFooterView(state string) string {
	textColor, lineColor := model.setColorVariables(state)
	info := FooterStyle.Foreground(textColor).Render(
		fmt.Sprintf("Number of characters %d", lipgloss.Width(model.msgInput.Value())),
	)
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", max(0, model.iaViewport.Width()-lipgloss.Width(info))))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func (model *Model) iaFooterView(state string) string {
	textColor, lineColor := model.setColorVariables(state)
	info := FooterStyle.Foreground(textColor).
		Render(fmt.Sprintf("%3.f%%", model.iaViewport.ScrollPercent()*100))
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", max(0, model.iaViewport.Width()-lipgloss.Width(info))))
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
	switch model.focusedElement {
	case focusMsgInput:
		model.msgInput.Focus()
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(model.Theme.Blur)

		iaViewHeaderContent = model.iaHeaderView("blur")
		iaViewFooterContent = model.iaFooterView("blur")

		userInputViewHeaderContent = model.userInputHeaderView("focus")
		userInputiewFooterContent = model.userInputFooterView("focus")
	case focusAIResponse:
		model.msgInput.Blur()
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(model.Theme.BorderFocus)

		iaViewHeaderContent = model.iaHeaderView("focus")
		iaViewFooterContent = model.iaFooterView("focus")

		userInputViewHeaderContent = model.userInputHeaderView("blur")
		userInputiewFooterContent = model.userInputFooterView("blur")
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
	model.msgInput.SetWidth(model.width / 2)
	model.iaViewport.SetWidth(model.width / 2)
	model.msgInput.SetHeight(userInputVContenHeight - 2)
	model.iaViewport.SetHeight(iaViewportContentHeight)

	model.iaViewport.Style = currentIaViewportStyle
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
		model.msgInput.View(),
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

func (model *Model) buildEditingMessageView(appStyle lipgloss.Style) string {
	var (
		glamourContent           string
		iaViewHeaderContent      string
		msgEditViewHeaderContent string
		iaViewFooterContent      string
		msgEditiewFooterContent  string
	)

	const glamourGutter = 3
	statusBarContent := model.WritingStatusBar.Render()
	currentIaViewportStyle := model.iaViewport.Style
	switch model.focusedElement {
	case focusMsgInput:
		model.msgEdit.Focus()
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(model.Theme.Blur)

		iaViewHeaderContent = model.iaHeaderView("blur")
		iaViewFooterContent = model.iaFooterView("blur")

		msgEditViewHeaderContent = model.msgEditHeaderView("focus")
		msgEditiewFooterContent = model.msgEditFooterView("focus")
	case focusAIResponse:
		model.msgEdit.Blur()
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(model.Theme.BorderFocus)

		iaViewHeaderContent = model.iaHeaderView("focus")
		iaViewFooterContent = model.iaFooterView("focus")

		msgEditViewHeaderContent = model.msgEditHeaderView("blur")
		msgEditiewFooterContent = model.msgEditFooterView("blur")
	}

	statusBarHeight := lipgloss.Height(model.WritingStatusBar.Render())
	verticalSpaceHeight := lipgloss.Height(VerticalSpace)
	helpViewHeight := lipgloss.Height(model.help.View(model.keys))
	iaHeaderH := lipgloss.Height(iaViewHeaderContent)
	iaFooterH := lipgloss.Height(iaViewFooterContent)
	msgEditViewHeaderH := lipgloss.Height(msgEditViewHeaderContent)
	msgEditViewFooterH := lipgloss.Height(msgEditiewFooterContent)
	iaVerticalMarginHeight := iaHeaderH + iaFooterH
	msgEditViewVerticalMarginHeight := msgEditViewHeaderH + msgEditViewFooterH
	totalAvailableContentHeight := model.height - appStyle.GetVerticalPadding() - helpViewHeight - statusBarHeight - verticalSpaceHeight - 2

	iaViewportContentHeight := totalAvailableContentHeight - iaVerticalMarginHeight
	msgEditVContenHeight := totalAvailableContentHeight - msgEditViewVerticalMarginHeight
	model.iaViewport.SetHeight(iaViewportContentHeight)
	model.msgEdit.SetHeight(msgEditVContenHeight - 2)

	model.iaViewport.Style = currentIaViewportStyle
	msgEditView := lipgloss.JoinVertical(lipgloss.Left,
		model.msgEdit.View(),
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
		msgEditViewHeaderContent,
		VerticalSpace,
		msgEditView,
		VerticalSpace,
		msgEditiewFooterContent,
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

	statusBarContent := model.WritingStatusBar.Render()
	helpView := fmt.Sprintf("  %s", model.help.View(model.keys))
	contentHeight := model.height - lipgloss.Height(helpView) - appStyle.GetVerticalPadding() - 2
	availableWidthForMainContent := max(0, model.width-appStyle.GetHorizontalFrameSize()-appStyle.
		GetHorizontalPadding())

	switch model.state {
	case stateSettingAPIKey:
		boxStyle := model.Theme.AppStyles().Base.
			Border(lipgloss.RoundedBorder(), true).
			BorderForeground(model.Theme.BorderFocus).
			Padding(1, 2).
			Width(80).
			Height(model.height / 2).
			Align(lipgloss.Center)

		titleStyle := model.Theme.AppStyles().Base.Foreground(model.Theme.Secondary).Bold(true)

		mainInstructionStyle := model.Theme.AppStyles().Base.Foreground(model.Theme.White)
		secondaryInstructionStyle := model.Theme.AppStyles().Base.Foreground(model.Theme.Accent)
		contentLines := []string{
			titleStyle.Render("Groq API Key Configuration"),
			"",
			mainInstructionStyle.Render("Enter your Groq API Key:"),
			"",
			model.apiKeyInput.View(),
			"",
			secondaryInstructionStyle.Render(
				"(Press Enter to save, Esc to cancel)",
			),
		}
		boxContent := lipgloss.JoinVertical(lipgloss.Center, contentLines...)
		renderedBox := boxStyle.Render(boxContent)
		actualContentHeightForCenteringBox := max(
			0,
			contentHeight-lipgloss.Height(statusBarContent)-lipgloss.
				Height(VerticalSpace),
		)
		centeredBox := lipgloss.Place(
			availableWidthForMainContent,
			actualContentHeightForCenteringBox,
			lipgloss.Center,
			lipgloss.Center,
			renderedBox,
		)
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			centeredBox,
		)
	case stateChoosingCommit:
		statusBarH := lipgloss.Height(statusBarContent)
		VerticalSpaceH := lipgloss.Height(VerticalSpace)
		availableHeightList := contentHeight - statusBarH - VerticalSpaceH
		model.mainList.SetSize(model.width, availableHeightList)
		uiElements := model.mainList.View()
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			uiElements,
		)
	case stateChoosingType:
		uiElements := model.commitTypeList.View()
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			uiElements,
		)

	case stateChoosingScope:
		statusBarH := lipgloss.Height(statusBarContent)
		VerticalSpaceH := lipgloss.Height(VerticalSpace)
		availableHeightForFileList := contentHeight - statusBarH - VerticalSpaceH
		model.fileList.SetSize(model.width, availableHeightForFileList)
		uiElements := model.fileList.View()
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			uiElements,
		)

	case stateWritingMessage:
		mainContent = model.buildWritingMessageView(appStyle)
	case stateEditMessage:
		mainContent = model.buildEditingMessageView(appStyle)
	}

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
