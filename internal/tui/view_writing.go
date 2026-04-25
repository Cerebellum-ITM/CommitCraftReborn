package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
)

func (model *Model) buildTabBar(totalWidth int) string {
	base := model.Theme.AppStyles().Base
	activeStyle := base.
		Background(model.Theme.Purple).
		Foreground(model.Theme.FgBase).
		Bold(true).
		Padding(0, 2)
	inactiveStyle := base.
		Background(model.Theme.Blur).
		Foreground(model.Theme.FocusableElement).
		Padding(0, 2)

	gap := base.Render(" ")

	var composeTab, pipelineTab string
	if model.activeTab == 0 {
		composeTab = activeStyle.Render("● Compose")
		pipelineTab = inactiveStyle.Render("○ Pipeline")
	} else {
		composeTab = inactiveStyle.Render("○ Compose")
		pipelineTab = activeStyle.Render("● Pipeline")
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Left, composeTab, gap, pipelineTab)
	whiteSpaces := HorizontalSpace + HorizontalSpace
	tabWidth := lipgloss.Width(tabs) + lipgloss.Width(whiteSpaces)
	line := base.Foreground(model.Theme.Blur).
		Render(strings.Repeat("─", max(0, totalWidth-tabWidth)))
	return lipgloss.JoinHorizontal(lipgloss.Left, tabs, whiteSpaces, line)
}

func (model *Model) buildPipelineDiffListView(width, height int) string {
	isFocused := model.focusedElement == focusPipelineDiffList
	state := "blur"
	if isFocused {
		state = "focus"
	}

	model.pipelineDiffList.SetWidth(width)
	model.pipelineDiffList.SetHeight(height - 4)

	header := model.buildStyledBorder(state, "Changed Files  [Enter] view diff  [Tab] switch panel", HeaderStyle, width, AlignHeader)
	count := len(model.pipelineDiffList.Items())
	footer := model.buildStyledBorder(state, fmt.Sprintf("%d file(s) modified", count), FooterStyle, width, AlignFooter)

	return lipgloss.JoinVertical(lipgloss.Left, header, model.pipelineDiffList.View(), footer)
}

func (model *Model) buildPipelineView(contentWidth, contentHeight int) string {
	glamourStyle := styles.TokyoNightStyleConfig
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(glamourStyle),
		glamour.WithWordWrap(contentWidth),
	)

	renderContent := func(raw string) string {
		if raw == "" {
			return "(empty — run the AI to populate this stage)"
		}
		s, _ := renderer.Render(raw)
		return s
	}

	if contentHeight < pipelineCompactThreshold {
		return model.buildPipelineViewCompact(contentWidth, contentHeight, renderContent)
	}

	stageH := contentHeight / 3

	model.pipelineViewport1.SetWidth(contentWidth)
	model.pipelineViewport1.SetHeight(max(1, stageH-3))
	model.pipelineViewport2.SetWidth(contentWidth)
	model.pipelineViewport2.SetHeight(max(1, stageH-3))
	model.pipelineViewport3.SetWidth(contentWidth)
	model.pipelineViewport3.SetHeight(max(1, stageH-3))

	model.pipelineViewport1.SetContent(renderContent(model.iaSummaryOutput))
	model.pipelineViewport2.SetContent(renderContent(model.iaCommitRawOutput))
	if model.commitTranslate == "" {
		model.pipelineViewport3.SetContent(renderContent(""))
	} else {
		model.pipelineViewport3.SetContent(renderContent(model.iaTitleRawOutput))
	}

	focusColor := model.Theme.BorderFocus
	blurColor := model.Theme.FocusableElement

	vp1Style := model.pipelineViewport1.Style
	vp2Style := model.pipelineViewport2.Style
	vp3Style := model.pipelineViewport3.Style

	switch model.activePipelineStage {
	case 0:
		vp1Style = vp1Style.BorderForeground(focusColor)
		vp2Style = vp2Style.BorderForeground(blurColor)
		vp3Style = vp3Style.BorderForeground(blurColor)
	case 1:
		vp1Style = vp1Style.BorderForeground(blurColor)
		vp2Style = vp2Style.BorderForeground(focusColor)
		vp3Style = vp3Style.BorderForeground(blurColor)
	case 2:
		vp1Style = vp1Style.BorderForeground(blurColor)
		vp2Style = vp2Style.BorderForeground(blurColor)
		vp3Style = vp3Style.BorderForeground(focusColor)
	}
	model.pipelineViewport1.Style = vp1Style
	model.pipelineViewport2.Style = vp2Style
	model.pipelineViewport3.Style = vp3Style

	stateOf := func(i int) string {
		if i == model.activePipelineStage {
			return "focus"
		}
		return "blur"
	}

	header1 := model.buildStyledBorder(stateOf(0), "Stage 1 — Summary  [1] re-run", HeaderStyle, contentWidth, AlignHeader)
	header2 := model.buildStyledBorder(stateOf(1), "Stage 2 — Raw Commit  [2] re-run", HeaderStyle, contentWidth, AlignHeader)
	header3 := model.buildStyledBorder(stateOf(2), "Stage 3 — Formatted  [3] re-run", HeaderStyle, contentWidth, AlignHeader)

	footer1 := model.buildStyledBorder(stateOf(0), fmt.Sprintf("%3.f%%", model.pipelineViewport1.ScrollPercent()*100), FooterStyle, contentWidth, AlignFooter)
	footer2 := model.buildStyledBorder(stateOf(1), fmt.Sprintf("%3.f%%", model.pipelineViewport2.ScrollPercent()*100), FooterStyle, contentWidth, AlignFooter)
	footer3 := model.buildStyledBorder(stateOf(2), fmt.Sprintf("%3.f%%", model.pipelineViewport3.ScrollPercent()*100), FooterStyle, contentWidth, AlignFooter)

	stage1 := lipgloss.JoinVertical(lipgloss.Left, header1, model.pipelineViewport1.View(), footer1)
	stage2 := lipgloss.JoinVertical(lipgloss.Left, header2, model.pipelineViewport2.View(), footer2)
	stage3 := lipgloss.JoinVertical(lipgloss.Left, header3, model.pipelineViewport3.View(), footer3)

	return lipgloss.JoinVertical(lipgloss.Left, stage1, stage2, stage3)
}

var pipelineStageLabels = [3]string{
	"Stage 1 — Summary  [1] re-run",
	"Stage 2 — Raw Commit  [2] re-run",
	"Stage 3 — Formatted  [3] re-run",
}

func (model *Model) buildPipelineViewCompact(
	contentWidth, contentHeight int,
	renderContent func(string) string,
) string {
	vps := [3]*viewport.Model{
		&model.pipelineViewport1,
		&model.pipelineViewport2,
		&model.pipelineViewport3,
	}
	rawContents := [3]string{model.iaSummaryOutput, model.iaCommitRawOutput, ""}
	if model.commitTranslate != "" {
		rawContents[2] = model.iaTitleRawOutput
	}

	active := model.activePipelineStage
	vp := vps[active]
	vp.SetWidth(contentWidth)
	vp.SetHeight(max(1, contentHeight-3))
	vp.SetContent(renderContent(rawContents[active]))
	vp.Style = vp.Style.BorderForeground(model.Theme.BorderFocus)

	label := fmt.Sprintf("%s  ← → switch", pipelineStageLabels[active])
	header := model.buildStyledBorder("focus", label, HeaderStyle, contentWidth, AlignHeader)
	footer := model.buildStyledBorder("focus", fmt.Sprintf("%3.f%%", vp.ScrollPercent()*100), FooterStyle, contentWidth, AlignFooter)

	return lipgloss.JoinVertical(lipgloss.Left, header, vp.View(), footer)
}

func (model *Model) buildWritingMessageView(appStyle lipgloss.Style) string {
	var (
		glamourContent             string
		iaViewHeaderContent        string
		userInputViewHeaderContent string
		iaViewFooterContent        string
		userInputiewFooterContent  string
		formattedLines             []string
	)

	const glamourGutter = 3
	statusBarContent := model.WritingStatusBar.Render()
	tabBar := model.buildTabBar(model.width)
	tabBarHeight := lipgloss.Height(tabBar + VerticalSpace)

	currentIaViewportStyle := model.iaViewport.Style
	switch model.focusedElement {
	case focusMsgInput:
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(
			model.Theme.FocusableElement,
		)
		model.commitsKeysViewport.Style = model.commitsKeysViewport.Style.BorderForeground(
			model.Theme.BorderFocus,
		)

		iaViewHeaderContent = model.iaHeaderView("blur")
		iaViewFooterContent = model.iaFooterView("blur")

		userInputViewHeaderContent = model.userInputHeaderView("focus")
		userInputiewFooterContent = model.userInputFooterView("focus")
	case focusAIResponse:
		currentIaViewportStyle = currentIaViewportStyle.BorderForeground(model.Theme.BorderFocus)
		model.commitsKeysViewport.Style = model.commitsKeysViewport.Style.BorderForeground(
			model.Theme.FocusableElement,
		)

		iaViewHeaderContent = model.iaHeaderView("focus")
		iaViewFooterContent = model.iaFooterView("focus")

		userInputViewHeaderContent = model.userInputHeaderView("blur")
		userInputiewFooterContent = model.userInputFooterView("blur")
	case focusPipelineViewport:
		userInputViewHeaderContent = model.userInputHeaderView("blur")
		userInputiewFooterContent = model.userInputFooterView("blur")
		iaViewHeaderContent = model.iaHeaderView("blur")
		iaViewFooterContent = model.iaFooterView("blur")
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
	totalAvailableContentHeight := model.height - appStyle.GetVerticalPadding() - helpViewHeight - statusBarHeight - verticalSpaceHeight - tabBarHeight - 2

	iaViewportContentHeight := totalAvailableContentHeight - iaVerticalMarginHeight
	userInputVContenHeight := totalAvailableContentHeight - userInputViewVerticalMarginHeight
	model.commitsKeysInput.SetWidth(model.width / 2)
	model.iaViewport.SetWidth(model.width / 2)
	model.commitsKeysViewport.SetWidth(model.width / 2)
	inputH := lipgloss.Height(model.commitsKeysInput.View())
	model.commitsKeysViewport.SetHeight(userInputVContenHeight - inputH - 2)
	model.iaViewport.SetHeight(iaViewportContentHeight)

	model.iaViewport.Style = currentIaViewportStyle
	glamourRenderWidth := model.iaViewport.Width() - model.iaViewport.Style.GetHorizontalFrameSize() - glamourGutter
	glamourStyle := styles.TokyoNightStyleConfig
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

	for i, point := range model.keyPoints {
		formattedLine := fmt.Sprintf("%d. %s", i+1, point)
		formattedLines = append(formattedLines, formattedLine)
	}
	keyPointsOutput := strings.Join(formattedLines, "\n")
	glamourContentStrCommitsKeys, _ := renderer.Render(keyPointsOutput)
	model.commitsKeysViewport.SetContent(glamourContentStrCommitsKeys)

	leftTranslatedContent := lipgloss.JoinVertical(lipgloss.Left,
		userInputViewHeaderContent,
		VerticalSpace,
		model.commitsKeysInput.View(),
		model.commitsKeysViewport.View(),
		VerticalSpace,
		userInputiewFooterContent,
	)

	var leftContent string
	var rightContent string
	if model.activeTab == 0 {
		leftContent = leftTranslatedContent
		rightContent = lipgloss.JoinVertical(lipgloss.Left,
			iaViewHeaderContent,
			model.iaViewport.View(),
			iaViewFooterContent,
		)
	} else {
		leftContent = model.buildPipelineDiffListView(model.width/2, totalAvailableContentHeight)
		rightContent = model.buildPipelineView(model.width/2, totalAvailableContentHeight)
	}

	uiElements := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftContent,
		rightContent,
	)
	return lipgloss.JoinVertical(lipgloss.Left,
		statusBarContent,
		VerticalSpace,
		tabBar,
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
	glamourStyle := styles.TokyoNightStyleConfig
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
