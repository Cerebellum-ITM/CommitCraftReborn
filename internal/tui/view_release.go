package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
)

// buildReleaseChooseCommitsView renders the redesigned layout for
// stateReleaseChoosingCommits: a thin top band (filter + mode bar +
// compact commit list, mirroring the main release view chrome), a
// middle panel with the selected commit's message rendered as
// markdown, and a bottom row split between a per-file picker and a
// diff viewport scoped to the chosen file.
func (model *Model) buildReleaseChooseCommitsView(appStyle lipgloss.Style) string {
	const (
		glamourGutter = 3
		gap           = 1
	)

	statusBarH := lipgloss.Height(model.WritingStatusBar.Render())
	helpH := lipgloss.Height(model.help.View(model.keys))
	vSpaceH := lipgloss.Height(VerticalSpace)

	totalH := model.height -
		appStyle.GetVerticalPadding() -
		statusBarH -
		helpH -
		2*vSpaceH -
		2

	if totalH < 12 {
		totalH = 12
	}

	totalW := model.width - appStyle.GetHorizontalPadding()
	if totalW < 40 {
		totalW = 40
	}

	chromeCols, chromeRows := titledPanelChrome()

	// Workspace commit list height. The "All commits / Selected only"
	// indicator now lives on the panel's top border (see `middle` slot
	// of titledPanelOpts) so it doesn't steal rows from the list anymore.
	const listRows = 9
	model.releaseChooseFilterBar.SetSize(max(10, totalW-chromeCols-2))

	// Top band height = filter(1) + blank(1) + list + chrome
	topInnerH := 1 + 1 + listRows
	topPanelH := topInnerH + chromeRows

	remaining := totalH - topPanelH - gap
	if remaining < 8 {
		remaining = 8
	}
	// 40 / 60 split: middle msg vp takes 40 %, bottom (file list + diff
	// vp) takes 60 % so the diff has enough vertical room to be useful.
	midPanelH := remaining * 40 / 100
	if midPanelH < 5 {
		midPanelH = 5
	}
	bottomPanelH := remaining - midPanelH - gap
	if bottomPanelH < 5 {
		bottomPanelH = 5
	}

	// Apply releaseFilesList counts to the bar so filtering math from the
	// list bubble doesn't print stale numbers.
	model.releaseChooseFilterBar.SetCounts(
		len(model.releaseCommitList.VisibleItems()),
		len(model.releaseCommitList.Items()),
	)

	// ---- top band content ----
	innerTopW := max(1, totalW-chromeCols-2)
	listW := innerTopW
	listH := listRows
	model.releaseCommitList.SetWidth(listW)
	model.releaseCommitList.SetHeight(listH)

	filterRow := lipgloss.PlaceHorizontal(
		innerTopW,
		lipgloss.Left,
		model.releaseChooseFilterBar.View(),
	)
	listRow := lipgloss.Place(
		innerTopW,
		listH,
		lipgloss.Left,
		lipgloss.Top,
		model.releaseCommitList.View(),
	)

	topBody := lipgloss.JoinVertical(lipgloss.Left,
		filterRow,
		"",
		listRow,
	)

	// Only the commit list (the bulk of the top panel) drives the panel
	// border colour. Filter and mode-bar focus already have their own
	// in-band indicators (arrow colour, virtual cursor on the textinput),
	// so reusing the same border for all three would confuse the user
	// into thinking the cursor list still owns up/down keys when it
	// doesn't.
	topBorder := model.borderColorForFocus(
		model.focusedElement == focusReleaseChooseCommitList,
	)
	topPanel := renderTitledPanel(titledPanelOpts{
		icon:        "✦",
		title:       "Workspace commits",
		hintRight:   model.releaseChooseTopHint(),
		middle:      model.releaseChooseModeIndicator(),
		middleRaw:   true,
		content:     topBody,
		width:       totalW,
		height:      topPanelH,
		borderColor: topBorder,
		titleColor:  model.Theme.FG,
		hintColor:   model.Theme.Muted,
	})

	// ---- middle: commit message viewport ----
	innerMidW := max(1, totalW-chromeCols-2)
	innerMidH := max(1, midPanelH-chromeRows)
	model.releaseViewport.SetWidth(innerMidW)
	model.releaseViewport.SetHeight(innerMidH)
	if model.commitLivePreview != "" {
		rendered := model.renderCommitMessage(
			model.commitMessageOnlyForPreview(),
			max(10, innerMidW-glamourGutter),
		)
		model.releaseViewport.SetContent(rendered)
	} else {
		model.releaseViewport.SetContent("")
	}
	midBorder := model.borderColorForFocus(model.focusedElement == focusReleaseChooseMsgVp)
	midPanel := renderTitledPanel(titledPanelOpts{
		icon:        "✎",
		title:       "Commit message",
		hintRight:   "↑/↓ scroll",
		content:     model.releaseViewport.View(),
		width:       totalW,
		height:      midPanelH,
		borderColor: midBorder,
		titleColor:  model.Theme.FG,
		hintColor:   model.Theme.Muted,
	})

	// ---- bottom: file list + diff vp side by side ----
	fileW := max(20, totalW*35/100)
	diffW := max(20, totalW-fileW-gap)
	innerFileW := max(1, fileW-chromeCols-2)
	innerFileH := max(1, bottomPanelH-chromeRows)
	innerDiffW := max(1, diffW-chromeCols-2)
	innerDiffH := max(1, bottomPanelH-chromeRows)

	model.releaseFilesList.SetSize(innerFileW, innerFileH)
	model.releaseDiffViewport.SetWidth(innerDiffW)
	model.releaseDiffViewport.SetHeight(innerDiffH)

	fileBorder := model.borderColorForFocus(model.focusedElement == focusReleaseChooseFileList)
	diffBorder := model.borderColorForFocus(model.focusedElement == focusReleaseChooseDiffVp)

	filesPanel := renderTitledPanel(titledPanelOpts{
		icon:        "≡",
		title:       "Files changed",
		hintRight:   model.releaseChooseFilesHint(),
		content:     model.releaseFilesList.View(),
		width:       fileW,
		height:      bottomPanelH,
		borderColor: fileBorder,
		titleColor:  model.Theme.FG,
		hintColor:   model.Theme.Muted,
	})
	diffPanel := renderTitledPanel(titledPanelOpts{
		icon:        "Δ",
		title:       "Diff",
		hintRight:   model.releaseChooseDiffHint(),
		content:     model.releaseDiffViewport.View(),
		width:       diffW,
		height:      bottomPanelH,
		borderColor: diffBorder,
		titleColor:  model.Theme.FG,
		hintColor:   model.Theme.Muted,
	})
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, filesPanel, " ", diffPanel)

	return lipgloss.JoinVertical(lipgloss.Left,
		topPanel,
		"",
		midPanel,
		"",
		bottomRow,
	)
}

// borderColorForFocus picks the panel border color based on whether one
// of its inner zones currently has focus. Centralises the focus/idle
// switch so the new layout never hardcodes a color.
func (model *Model) borderColorForFocus(focused bool) color.Color {
	if focused {
		return model.Theme.Primary
	}
	return model.Theme.Subtle
}

// commitMessageOnlyForPreview strips the "```\n<diff>```" trailer that
// NewReleaseCommitList glues onto the Preview field so the middle msg
// vp shows just the commit message (the diff has its own panel now).
func (model *Model) commitMessageOnlyForPreview() string {
	preview := model.commitLivePreview
	if i := strings.Index(preview, "```\n"); i != -1 {
		return strings.TrimRight(preview[:i], "\n ")
	}
	return preview
}

// releaseChooseTopHint is the hint shown at the top-right of the
// workspace-commits panel. Mirrors the existing release main view —
// "ctrl+f" to cycle the filter mode, "ctrl+a" to add the cursor's
// commit to the selection.
func (model *Model) releaseChooseTopHint() string {
	hs := model.Theme.AppStyles().Help
	return hs.ShortKey.Render("ctrl+a") + hs.ShortDesc.Render(" select") +
		hs.ShortSeparator.Render(" · ") +
		hs.ShortKey.Render("ctrl+f") + hs.ShortDesc.Render(" filter mode") +
		hs.ShortSeparator.Render(" · ") +
		hs.ShortKey.Render("ctrl+e") + hs.ShortDesc.Render(" swap")
}

// releaseChooseModeIndicator renders the "All commits / Selected only"
// segmented indicator inline on the workspace-commits panel top border
// (passed via the `middle` slot of titledPanelOpts). Replaces the old
// in-body HistoryModeBar pill row, freeing rows for the commit list.
func (model *Model) releaseChooseModeIndicator() string {
	theme := model.Theme
	hs := theme.AppStyles().Help

	left := "All commits"
	right := "Selected only"
	mode := model.releaseChooseModeBar.Mode()

	render := func(label string, active bool) string {
		dot := "○"
		color := theme.Muted
		if active {
			dot = "●"
			color = theme.Primary
		}
		dotStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
		labelStyle := lipgloss.NewStyle().Foreground(color)
		if active {
			labelStyle = labelStyle.Bold(true)
		}
		return dotStyle.Render(dot) + " " + labelStyle.Render(label)
	}

	return render(left, mode == ModeKeyPointsBody) +
		hs.ShortSeparator.Render("  ·  ") +
		render(right, mode == ModeStagesResponse)
}

func (model *Model) releaseChooseFilesHint() string {
	hs := model.Theme.AppStyles().Help
	return hs.ShortKey.Render("↑/↓") + hs.ShortDesc.Render(" pick file")
}

func (model *Model) releaseChooseDiffHint() string {
	pct := int(model.releaseDiffViewport.ScrollPercent() * 100)
	hs := model.Theme.AppStyles().Help
	return hs.ShortKey.Render("pgup/pgdn") + hs.ShortDesc.Render(" scroll") +
		hs.ShortSeparator.Render(" · ") +
		hs.ShortDesc.Render(fmt.Sprintf("%d%%", pct))
}

func (model *Model) buildReleaseView(appStyle lipgloss.Style) string {
	const glamourGutter = 3

	var (
		releaseCommitListHeader string
		releaseCommitListFooter string
		headerViewport          string
		footerViewport          string
		glamourContentStr       string
		viewportStyle           lipgloss.Style
	)

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
	glamourStyle := styles.TokyoNightStyleConfig
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(glamourStyle),
		glamour.WithWordWrap(glamourRenderWidth),
	)

	glamourContentStr, _ = renderer.Render(model.commitLivePreview)
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

	return uiElements
}

func (model *Model) buildRewordSelectView() string {
	const glamourGutter = 3

	popupW := int(float64(model.width) * rewordPopupRatio)
	popupH := int(float64(model.height) * rewordPopupRatio)
	// 2 chars for the rounded border on each axis
	innerW := max(10, popupW-2)
	innerH := max(4, popupH-2)
	halfW := innerW / 2

	listHeader := model.buildStyledBorder(
		"focus",
		"Select a commit to reword",
		HeaderStyle,
		halfW,
		AlignHeader,
	)
	listFooter := model.buildStyledBorder(
		"focus",
		"↑/↓ navigate  ·  enter reword  ·  esc back",
		FooterStyle,
		halfW,
		AlignFooter,
	)
	previewHeader := model.buildStyledBorder("blur", "Commit diff", HeaderStyle, halfW, AlignHeader)
	previewFooter := model.buildStyledBorder(
		"blur",
		fmt.Sprintf("%3.f%%", model.releaseViewport.ScrollPercent()*100),
		FooterStyle,
		halfW,
		AlignFooter,
	)

	headerH := lipgloss.Height(listHeader)
	footerH := lipgloss.Height(listFooter)
	vertSpaceH := 2 * lipgloss.Height(VerticalSpace)
	listHeight := max(1, innerH-headerH-footerH-vertSpaceH)

	model.releaseCommitList.SetWidth(halfW)
	model.releaseCommitList.SetHeight(listHeight)
	model.releaseViewport.SetWidth(halfW)
	model.releaseViewport.SetHeight(listHeight)

	glamourRenderWidth := max(
		10,
		halfW-model.releaseViewport.Style.GetHorizontalFrameSize()-glamourGutter,
	)
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStyles(styles.DarkStyleConfig),
		glamour.WithWordWrap(glamourRenderWidth),
	)
	glamourContentStr, _ := renderer.Render(model.commitLivePreview)
	model.releaseViewport.SetContent(glamourContentStr)

	listFocusLine := lipgloss.NewStyle().Height(listHeight).Render("┃")
	listCompositeView := lipgloss.JoinHorizontal(
		lipgloss.Center,
		listFocusLine,
		model.releaseCommitList.View(),
	)

	leftContent := lipgloss.JoinVertical(lipgloss.Left,
		listHeader,
		VerticalSpace,
		listCompositeView,
		VerticalSpace,
		listFooter,
	)
	rightContent := lipgloss.JoinVertical(lipgloss.Left,
		previewHeader,
		VerticalSpace,
		model.releaseViewport.View(),
		VerticalSpace,
		previewFooter,
	)

	innerContent := lipgloss.JoinHorizontal(lipgloss.Top, leftContent, rightContent)

	popupBox := lipgloss.NewStyle().
		Width(innerW).
		Height(innerH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(model.Theme.Primary).
		Render(innerContent)

	statusBarH := lipgloss.Height(model.WritingStatusBar.Render())
	helpView := lipgloss.NewStyle().Padding(0, 2).SetString(model.help.View(model.keys)).String()
	helpH := lipgloss.Height(helpView)
	vertSpaceSingle := lipgloss.Height(VerticalSpace)
	availableH := max(1, model.height-statusBarH-helpH-vertSpaceSingle)

	return lipgloss.Place(
		model.width,
		availableH,
		lipgloss.Center,
		lipgloss.Center,
		popupBox,
	)
}
