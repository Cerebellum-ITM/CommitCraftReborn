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

type BorderAlignment int

const (
	AlignHeader BorderAlignment = iota
	AlignFooter
)

func (m *Model) buildStyledBorder(
	state string,
	content string,
	baseStyle lipgloss.Style,
	componentWidth int,
	alignment BorderAlignment,
) string {
	textColor, lineColor := m.setColorVariables(state)

	styledContent := baseStyle.Foreground(textColor).Render(content)

	contentWidth := lipgloss.Width(styledContent)
	line := LineStyle.Foreground(lineColor).
		Render(strings.Repeat("─", max(0, componentWidth-contentWidth)))

	switch alignment {
	case AlignHeader:
		return lipgloss.JoinHorizontal(lipgloss.Left, styledContent, line)
	case AlignFooter:
		return lipgloss.JoinHorizontal(lipgloss.Left, line, styledContent)
	default:
		return ""
	}
}

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
	title := "Final response of AI models"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) iaFooterView(state string) string {
	info := fmt.Sprintf("%3.f%%", model.iaViewport.ScrollPercent()*100)
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
}

func (model *Model) userInputHeaderView(state string) string {
	title := "Enter the text with your summary of the changes"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) userInputFooterView(state string) string {
	info := fmt.Sprintf("Number of characters %d", lipgloss.Width(model.msgInput.Value()))
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
}

func (model *Model) msgEditHeaderView(state string) string {
	title := "Write the modifications"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) msgEditFooterView(state string) string {
	info := fmt.Sprintf("Number of characters %d", lipgloss.Width(model.msgEdit.Value()))
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
}

func (model *Model) releaseHeaderView(state string) string {
	title := "Commit list | Select at least one commit"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) releaseFooterView(state string) string {
	commitSymbol := model.Theme.AppSymbols().Commit
	info := fmt.Sprintf("%s %d %s",
		commitSymbol,
		len(model.selectedCommitList),
		"Selected Commits",
	)
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
}

func (model *Model) releaseLivePreviewHeaderView(state string) string {
	title := "Commit content"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) releaseLivePreviewFooterView(state string) string {
	info := fmt.Sprintf("%3.f%%", model.releaseViewport.ScrollPercent()*100)
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
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
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(
			model.Theme.FocusableElement,
		)

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
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(
			model.Theme.FocusableElement,
		)

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
	model.iaViewport.SetWidth(model.width / 2)
	model.msgEdit.SetWidth(model.width / 2)
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

func (model *Model) buildReleaseView(appStyle lipgloss.Style) string {
	const glamourGutter = 3
	var (
		releaseCommitListHeader string
		releaseCommitListFooter string
		headerViewport          string
		footerViewport          string
		viewportStyle           lipgloss.Style
	)

	statusBarContent := model.WritingStatusBar.Render()
	statusBarHeight := lipgloss.Height(model.WritingStatusBar.Render())
	verticalSpaceHeight := lipgloss.Height(VerticalSpace)
	helpViewHeight := lipgloss.Height(model.help.View(model.keys))
	viewportStyle = model.releaseViewport.Style

	switch model.focusedElement {
	case focusViewportElement:
		viewportStyle = viewportStyle.BorderForeground(model.Theme.BorderFocus)
		releaseCommitListHeader = model.releaseHeaderView("blur")
		releaseCommitListFooter = model.releaseFooterView("blur")
		headerViewport = model.releaseLivePreviewHeaderView("focus")
		footerViewport = model.releaseLivePreviewFooterView("focus")

	case focusListElement:
		viewportStyle = viewportStyle.BorderForeground(model.Theme.FocusableElement)
		releaseCommitListHeader = model.releaseHeaderView("focus")
		releaseCommitListFooter = model.releaseFooterView("focus")
		headerViewport = model.releaseLivePreviewHeaderView("blur")
		footerViewport = model.releaseLivePreviewFooterView("blur")
	}

	HeaderH := lipgloss.Height(releaseCommitListHeader)
	FooterH := lipgloss.Height(releaseCommitListFooter)

	model.releaseViewport.Style = viewportStyle
	totalAvailableContentHeight := model.height - appStyle.GetVerticalPadding() - helpViewHeight - statusBarHeight - verticalSpaceHeight - FooterH - HeaderH - 2
	model.releaseCommitList.SetWidth(model.width / 2)
	model.releaseCommitList.SetHeight(totalAvailableContentHeight)
	model.releaseViewport.SetWidth(model.width / 2)
	model.releaseViewport.SetHeight(totalAvailableContentHeight)

	glamourRenderWidth := model.releaseViewport.Width() - model.releaseViewport.Style.GetHorizontalFrameSize() - glamourGutter
	glamourStyle := styles.DarkStyleConfig
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(glamourStyle),
		glamour.WithWordWrap(glamourRenderWidth),
	)

	glamourContentStr, _ := renderer.Render(model.commitLivePreview)
	model.releaseViewport.SetContent(glamourContentStr)
	listFocusLine := lipgloss.NewStyle().Height(totalAvailableContentHeight).Render("┃")

	listCompositeView := lipgloss.JoinHorizontal(
		lipgloss.Center,
		listFocusLine,
		model.releaseCommitList.View(),
	)

	leftTranslatedContent := lipgloss.JoinVertical(lipgloss.Left,
		releaseCommitListHeader,
		VerticalSpace,
		listCompositeView,
		VerticalSpace,
		releaseCommitListFooter,
	)

	rightTranslatedContent := lipgloss.JoinVertical(lipgloss.Left,
		headerViewport,
		VerticalSpace,
		model.releaseViewport.View(),
		VerticalSpace,
		footerViewport,
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
		Width(model.width).
		Height(model.height)

	if model.err != nil {
		return "Error: " + model.err.Error()
	}
	statusBarContent := model.WritingStatusBar.Render()
	helpView := lipgloss.NewStyle().Padding(0, 2).SetString(model.help.View(model.keys)).String()
	contentHeight := model.height
	helpViewH := lipgloss.Height(helpView)
	availableWidthForMainContent := max(0, model.width-appStyle.GetHorizontalFrameSize()-appStyle.
		GetHorizontalPadding())
	if model.height > 10 {
		contentHeight = contentHeight - model.height/10*2
	}
	statusBarH := lipgloss.Height(statusBarContent)
	VerticalSpaceH := 2 * lipgloss.Height(VerticalSpace)
	availableHeightForMainContent := contentHeight - statusBarH - VerticalSpaceH - helpViewH

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
		model.mainList.SetSize(availableWidthForMainContent, availableHeightForMainContent)
		uiElements := model.mainList.View()
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			uiElements,
		)
	case stateChoosingType:
		model.commitTypeList.SetSize(availableWidthForMainContent, availableHeightForMainContent)
		uiElements := model.commitTypeList.View()
		mainContent = lipgloss.JoinVertical(lipgloss.Left,
			statusBarContent,
			VerticalSpace,
			uiElements,
		)

	case stateChoosingScope:
		model.fileList.SetSize(availableWidthForMainContent, availableHeightForMainContent)
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
	case stateReleaseChoosingCommits:
		mainContent = model.buildReleaseView(appStyle)
	}

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
