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
	const glamourGutter = 3

	statusBarContent := model.WritingStatusBar.Render()

	statusBarHeight := lipgloss.Height(statusBarContent)
	verticalSpaceHeight := lipgloss.Height(VerticalSpace)
	helpViewHeight := lipgloss.Height(model.help.View(model.keys))
	totalAvailableContentHeight := model.height -
		appStyle.GetVerticalPadding() -
		helpViewHeight -
		statusBarHeight -
		verticalSpaceHeight -
		2

	// 45/55 split for the two titled panels — slightly favours the AI
	// suggestion side because that's where the rendered markdown lives.
	leftW := max(40, model.width*45/100)
	rightW := max(30, model.width-leftW-1)
	panelH := max(15, totalAvailableContentHeight)

	chromeCols, chromeRows := titledPanelChrome()
	innerLeftW := max(1, leftW-chromeCols-2)   // 2 = 1 char padding on each side
	innerLeftH := max(1, panelH-chromeRows-1)  // 1 row of internal top padding
	innerRightW := max(1, rightW-chromeCols-2)
	innerRightH := max(1, panelH-chromeRows-1)

	model.commitsKeysInput.SetWidth(innerLeftW)

	// Drive the right-pane viewport so the AI text wraps to the panel.
	model.iaViewport.SetWidth(innerRightW)
	model.iaViewport.SetHeight(innerRightH)
	glamourRenderWidth := max(10, innerRightW-glamourGutter)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.TokyoNightStyleConfig),
		glamour.WithWordWrap(glamourRenderWidth),
	)
	if model.commitTranslate != "" {
		rendered, _ := renderer.Render(model.commitTranslate)
		model.iaViewport.SetContent(rendered)
	} else {
		model.iaViewport.SetContent("")
	}

	leftFocus := isComposeFocus(model.focusedElement)
	leftBorder := model.Theme.Subtle
	if leftFocus {
		leftBorder = model.Theme.Primary
	}
	rightFocus := model.focusedElement == focusComposeAISuggestion ||
		model.focusedElement == focusAIResponse
	rightBorder := model.Theme.Subtle
	if rightFocus {
		rightBorder = model.Theme.Primary
	}

	// Left panel content: 5 stacked sections separated by blank lines.
	leftBody := model.assembleComposeLeftBody(innerLeftW, innerLeftH)

	leftPanel := renderTitledPanel(titledPanelOpts{
		icon:        "⚑",
		title:       "summary",
		hintRight:   "press Tab",
		content:     leftBody,
		width:       leftW,
		height:      panelH,
		borderColor: leftBorder,
		titleColor:  model.Theme.FG,
		hintColor:   model.Theme.Muted,
	})

	rightBody := model.renderAISuggestionContent(innerRightW, innerRightH)
	rightPanel := renderTitledPanel(titledPanelOpts{
		icon:        "✦",
		title:       "ai suggestion",
		hintRight:   "press ^W to generate",
		content:     rightBody,
		width:       rightW,
		height:      panelH,
		borderColor: rightBorder,
		titleColor:  model.Theme.FG,
		hintColor:   model.Theme.Muted,
	})

	uiElements := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
	bottomBar := model.renderComposeBottomBar(model.width)

	return lipgloss.JoinVertical(lipgloss.Left,
		uiElements,
		VerticalSpace,
		bottomBar,
	)
}

// assembleComposeLeftBody stacks the 5 sections inside the left panel and
// returns the joined string ready to feed renderTitledPanel.
func (model *Model) assembleComposeLeftBody(innerW, innerH int) string {
	// Render each section once and let JoinVertical handle the spacing.
	typeRow := model.renderComposeTypeRow(innerW, model.focusedElement == focusComposeType)
	scopeRow := model.renderComposeScopeRow(innerW, model.focusedElement == focusComposeScope)
	summary := model.renderComposeSummaryArea(innerW, model.focusedElement == focusComposeSummary)
	pipelineModels := model.renderComposePipelineModelsArea(innerW, model.focusedElement == focusComposePipelineModels)

	// Reserve a chunk for keypoints and let it grow with the panel.
	usedH := lipgloss.Height(typeRow) +
		lipgloss.Height(scopeRow) +
		lipgloss.Height(summary) +
		lipgloss.Height(pipelineModels) +
		8 // accumulated blank-line separators
	keypointsH := max(3, innerH-usedH)
	keypoints := model.renderComposeKeypointsArea(innerW, keypointsH, model.focusedElement == focusComposeKeypoints)

	divider := model.renderComposeDivider(innerW)

	return lipgloss.JoinVertical(lipgloss.Left,
		typeRow,
		"",
		scopeRow,
		"",
		divider,
		"",
		summary,
		"",
		keypoints,
		"",
		pipelineModels,
	)
}

// isComposeFocus reports whether the given focus enum belongs to the
// left-side compose sections, used to pick the panel border color.
func isComposeFocus(f focusableElement) bool {
	switch f {
	case focusComposeType,
		focusComposeScope,
		focusComposeSummary,
		focusComposeKeypoints,
		focusComposePipelineModels,
		focusMsgInput:
		return true
	}
	return false
}

