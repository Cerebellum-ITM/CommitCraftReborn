package tui

import (
	_ "embed"
	"image/color"
	"strings"

	"charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	rewordPopupRatio         = 1
	pipelineCompactThreshold = 30
)

var (
	focusColor      color.Color
	focusColorText  color.Color
	blurColor       color.Color
	VerticalSpace   = lipgloss.NewStyle().Height(1).Render("")
	HorizontalSpace = lipgloss.NewStyle().Width(1).Render("")
	LineStyle       = lipgloss.NewStyle()
	HeaderStyle     = lipgloss.NewStyle().
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




// View renders the UI based on the current state of the model.
func (model *Model) View() tea.View {
	var mainContent string

	appStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(model.width).
		Height(model.height)

	if model.err != nil {
		return tea.NewView("Error: " + model.err.Error())
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
	case stateReleaseMainMenu:
		model.releaseMainList.SetSize(availableWidthForMainContent/2, availableHeightForMainContent)
		uiElements := model.releaseMainList.View()
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
	case stateReleaseChoosingCommits, stateReleaseBuildingText:
		mainContent = model.buildReleaseView(appStyle)
	case stateRewordSelectCommit:
		rewordContent := model.buildRewordSelectView()
		mainView := lipgloss.JoinVertical(lipgloss.Left,
			rewordContent,
			VerticalSpace,
			helpView,
		)
		finalView := tea.NewView(mainView)
		finalView.AltScreen = true
		return finalView
	}

	mainView := lipgloss.JoinVertical(lipgloss.Left,
		mainContent,
		VerticalSpace,
		helpView,
	)

	if model.logViewVisible {
		logsView := model.renderLogsPopup()
		startX, startY := calculatePopupPosition(model.width, model.height, logsView)
		mainLayer := lipgloss.NewLayer(mainView)
		logsLayer := lipgloss.NewLayer(logsView).X(startX).Y(startY).Z(2)
		comp := lipgloss.NewCompositor(mainLayer, logsLayer)
		finalView := tea.NewView(comp.Render())
		finalView.AltScreen = true
		return finalView
	}

	if model.popup == nil {
		finalView := tea.NewView(mainView)
		finalView.AltScreen = true
		return finalView
	}

	var ok bool
	var popupView tea.View

	switch popupModel := model.popup.(type) {
	case DeleteConfirmPopupModel:
		ok = true
		popupView = popupModel.View()
	case listPopupModel:
		ok = true
		popupView = popupModel.View()
	case mentionFilePopupModel:
		ok = true
		popupView = popupModel.View()
	case diffViewPopup:
		ok = true
		popupView = popupModel.View()
	case versionPopupModel:
		ok = true
		popupView = popupModel.View()
	default:
		ok = false
	}

	if !ok {
		return tea.NewView("Error: The popup is not of the expected type.")
	}

	startX, startY := calculatePopupPosition(model.width, model.height, popupView.Content)
	mainLayer := lipgloss.NewLayer(mainView)
	popupLayer := lipgloss.NewLayer(popupView.Content).X(startX).Y(startY).Z(1)
	comp := lipgloss.NewCompositor(mainLayer, popupLayer)
	finalRender := comp.Render()
	finalView := tea.NewView(finalRender)
	finalView.AltScreen = true
	return finalView
}
