package tui

import (
	"fmt"

	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
)

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

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBarContent,
		VerticalSpace,
		uiElements,
	)
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

	statusBarContent := model.WritingStatusBar.Render()
	helpView := lipgloss.NewStyle().Padding(0, 2).SetString(model.help.View(model.keys)).String()
	statusBarH := lipgloss.Height(statusBarContent)
	helpH := lipgloss.Height(helpView)
	vertSpaceSingle := lipgloss.Height(VerticalSpace)
	availableH := max(1, model.height-statusBarH-helpH-vertSpaceSingle)

	centeredPopup := lipgloss.Place(
		model.width,
		availableH,
		lipgloss.Center,
		lipgloss.Center,
		popupBox,
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		statusBarContent,
		VerticalSpace,
		centeredPopup,
	)
}
