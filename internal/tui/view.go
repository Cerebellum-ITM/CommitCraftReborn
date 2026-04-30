package tui

import (
	_ "embed"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
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
	// Skip the first frame before bubbletea has delivered the initial
	// tea.WindowSizeMsg. Without dimensions, lipgloss.Place can't center
	// content (the loading panel ends up flush top-left, then jumps to
	// the middle once size lands). An empty frame here is invisible and
	// the next render — already armed with width/height — paints
	// correctly the very first time the user sees anything.
	if model.width <= 0 || model.height <= 0 {
		empty := tea.NewView("")
		empty.AltScreen = true
		return empty
	}
	var mainContent string

	appStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(model.width).
		Height(model.height)

	if model.err != nil {
		return tea.NewView("Error: " + model.err.Error())
	}
	statusBarContent := model.WritingStatusBar.Render()
	var helpView string
	if model.state == stateWritingMessage {
		helpView = lipgloss.NewStyle().
			Padding(0, 2).
			SetString(model.renderComposeHelpLine()).
			String()
	} else {
		helpView = lipgloss.NewStyle().Padding(0, 2).SetString(model.renderStateHelpLine()).String()
	}

	// Persistent tab bar lives above everything; it hides on the API-key
	// bootstrap so it stays a focused single-step screen.
	var tabBarContent string
	tabBarH := 0
	if model.shouldShowTabBar() {
		tabBarContent = model.renderTabBar(model.width)
		tabBarH = lipgloss.Height(tabBarContent)
	}

	contentHeight := model.height
	helpViewH := lipgloss.Height(helpView)
	availableWidthForMainContent := max(0, model.width-appStyle.GetHorizontalFrameSize())
	if model.height > 10 {
		contentHeight = contentHeight - model.height/10*2
	}
	statusBarH := lipgloss.Height(statusBarContent)
	VerticalSpaceH := 2 * lipgloss.Height(VerticalSpace)
	availableHeightForMainContent := contentHeight - statusBarH - VerticalSpaceH - helpViewH - tabBarH

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
		centeredBox := lipgloss.Place(
			availableWidthForMainContent,
			availableHeightForMainContent,
			lipgloss.Center,
			lipgloss.Center,
			renderedBox,
		)
		mainContent = centeredBox
	case stateChoosingCommit:
		// Mirror the statePipeline pattern: the shared
		// availableWidth/Height calc shaves 20 % of the height and leaves
		// a horizontal margin for an appStyle that is never applied to
		// mainView. Use the real remaining surface so the History frame
		// reaches both edges of the terminal.
		histW := model.width
		histH := max(0, model.height-statusBarH-VerticalSpaceH-helpViewH-tabBarH)
		listW, listH := model.historyView.MasterListSize(histW, histH)
		model.mainList.SetSize(listW, listH)
		model.historyView.SetCounts(
			len(model.mainList.VisibleItems()),
			len(model.mainList.Items()),
		)
		masterListView := model.mainList.View()
		mainContent = model.historyView.View(masterListView, histW, histH)
	case stateReleaseMainMenu:
		histW := model.width
		histH := max(0, model.height-statusBarH-VerticalSpaceH-helpViewH-tabBarH)
		// While the async release-history sync is in flight, render a
		// centered loading panel instead of the half-painted chrome.
		// Without this, the user sees the master list flash in before
		// the dual panel hydrates — visually noisy on slow git lookups.
		if model.releaseLoading {
			mainContent = model.renderReleaseLoading(histW, histH)
			break
		}
		// Same chrome as stateChoosingCommit but bound to the release
		// history view (its own filter modes, dual panel, etc.).
		listW, listH := model.releaseHistoryView.MasterListSize(histW, histH)
		model.releaseMainList.SetSize(listW, listH)
		model.releaseHistoryView.SetCounts(
			len(model.releaseMainList.VisibleItems()),
			len(model.releaseMainList.Items()),
		)
		masterListView := model.releaseMainList.View()
		mainContent = model.releaseHistoryView.View(masterListView, histW, histH)
	case stateChoosingType:
		model.commitTypeList.SetSize(availableWidthForMainContent, availableHeightForMainContent)
		mainContent = model.commitTypeList.View()

	case stateChoosingScope:
		model.fileList.SetSize(availableWidthForMainContent, availableHeightForMainContent)
		mainContent = model.fileList.View()

	case stateWritingMessage:
		mainContent = model.buildWritingMessageView(appStyle)
	case stateReleaseChoosingCommits:
		mainContent = model.buildReleaseChooseCommitsView(appStyle)
	case stateReleaseBuildingText:
		// The release builder reuses the Pipeline tab cards: stage 1 =
		// Release Body, stage 2 = Release Title, stage 3 = Release
		// Refine. The pipeline preset is flipped to release in
		// updateReleaseChoosingCommits before the run kicks off.
		pipeW := model.width
		pipeH := max(0, model.height-statusBarH-VerticalSpaceH-helpViewH-tabBarH)
		mainContent = model.viewPipeline(pipeW, pipeH)
	case statePipeline:
		// The shared availableWidth/Height calc subtracts paddings that
		// aren't actually applied to mainContent (mainView is composed
		// without appStyle) and shaves 20% off the height for unclear
		// reasons. The Pipeline tab is densely packed; give it the real
		// remaining surface so its panels can fill the terminal.
		pipeW := model.width
		pipeH := max(0, model.height-statusBarH-VerticalSpaceH-helpViewH-tabBarH)
		mainContent = model.viewPipeline(pipeW, pipeH)
	case stateRewordSelectCommit:
		mainContent = model.buildRewordSelectView()
	case stateOutput:
		mainContent = model.buildOutputView(appStyle)
	}

	// Final layout: WritingStatusBar is the very first element, separated
	// from the tab bar by a blank line, so the level-coloured status surface
	// always stays at the top of the screen.
	var stack []string
	stack = append(stack, statusBarContent, VerticalSpace)
	if tabBarContent != "" {
		stack = append(stack, tabBarContent)
	}
	stack = append(stack, mainContent, VerticalSpace, helpView)
	mainView := lipgloss.JoinVertical(lipgloss.Left, stack...)

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
	case commitTypePopupModel:
		ok = true
		popupView = popupModel.View()
	case scopePopupModel:
		ok = true
		popupView = popupModel.View()
	case editMessagePopupModel:
		ok = true
		popupView = popupModel.View()
	case configPopupModel:
		ok = true
		popupView = popupModel.View()
	case modelPickerPopup:
		ok = true
		popupView = popupModel.View()
	case stageHistoryPopupModel:
		ok = true
		popupView = popupModel.View()
	case keybindingsPopupModel:
		ok = true
		popupView = popupModel.View()
	case commandPalettePopupModel:
		ok = true
		popupView = popupModel.View()
	case tagPalettePopupModel:
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
