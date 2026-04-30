package tui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// outputSegmentDefs returns the list of content segments available in
// the Output view's right pane. Stage 4 (changelog) is conditional on
// having content for it.
type outputSegDef struct {
	id    outputSegment
	label string // short label for the horizontal selector
	tech  string // technical / display title
}

func (model *Model) outputSegmentDefs() []outputSegDef {
	defs := []outputSegDef{
		{outSegFinal, "Final", "Final Message"},
		{outSegSummary, "Change Analyzer", "Stage 1 · Change Analyzer"},
		{outSegBody, "Commit Body", "Stage 2 · Commit Body"},
		{outSegTitle, "Commit Title", "Stage 3 · Commit Title"},
	}
	if strings.TrimSpace(model.iaChangelogEntry) != "" {
		defs = append(defs, outputSegDef{
			outSegChangelog, "Changelog", "Stage 4 · Changelog Refiner",
		})
	}
	return defs
}

// segmentText returns the raw text rendered into the right-pane
// viewport for the currently selected segment.
func (model *Model) outputSegmentText() string {
	switch model.outputSegment {
	case outSegSummary:
		return model.iaSummaryOutput
	case outSegBody:
		return model.iaCommitRawOutput
	case outSegTitle:
		return model.iaTitleRawOutput
	case outSegChangelog:
		return model.iaChangelogEntry
	default:
		return outputCommitMessageOrFallback(model, model.currentCommit)
	}
}

// buildOutputView renders the post-generation review screen.
func (model *Model) buildOutputView(appStyle lipgloss.Style) string {
	const glamourGutter = 3

	statusBarHeight := lipgloss.Height(model.WritingStatusBar.Render())
	verticalSpaceHeight := lipgloss.Height(VerticalSpace)
	helpViewHeight := lipgloss.Height(model.help.View(model.keys))
	totalAvailableContentHeight := model.height -
		appStyle.GetVerticalPadding() -
		helpViewHeight -
		statusBarHeight -
		verticalSpaceHeight -
		2

	leftW := max(40, model.width*45/100)
	rightW := max(30, model.width-leftW-1)
	panelH := max(15, totalAvailableContentHeight)

	chromeCols, chromeRows := titledPanelChrome()
	innerLeftW := max(1, leftW-chromeCols-2)
	innerLeftH := max(1, panelH-chromeRows-1)
	innerRightW := max(1, rightW-chromeCols-2)
	innerRightH := max(1, panelH-chromeRows-1)

	// Right pane: top row is the segmented selector, the rest is the
	// content viewport.
	segBar := model.renderOutputSegmentBar(innerRightW)
	segBarH := lipgloss.Height(segBar)
	vpH := max(1, innerRightH-segBarH-1)

	model.iaViewport.SetWidth(innerRightW)
	model.iaViewport.SetHeight(vpH)
	model.iaViewport.SetContent(
		model.renderCommitMessage(model.outputSegmentText(), max(10, innerRightW-glamourGutter)),
	)

	// Left pane: long-form report inside its own viewport so it scrolls.
	model.outputReportViewport.SetWidth(innerLeftW)
	model.outputReportViewport.SetHeight(innerLeftH)
	model.outputReportViewport.SetContent(model.assembleOutputLeftBody(innerLeftW))

	leftFocused := model.focusedElement == focusOutputReport
	rightFocused := model.focusedElement == focusOutputContent
	leftBorder := model.Theme.Subtle
	if leftFocused {
		leftBorder = model.Theme.Primary
	}
	rightBorder := model.Theme.Subtle
	if rightFocused {
		rightBorder = model.Theme.Primary
	}

	leftHint := "tab to focus"
	if leftFocused {
		leftHint = "↑↓/pgup/pgdn scroll"
	}
	rightHint := "tab to focus"
	if rightFocused {
		rightHint = "←/→ segment · ↑↓ scroll"
	}

	leftPanel := renderTitledPanel(titledPanelOpts{
		icon:        "✦",
		title:       "report",
		hintRight:   leftHint,
		content:     model.outputReportViewport.View(),
		width:       leftW,
		height:      panelH,
		borderColor: leftBorder,
		titleColor:  model.Theme.FG,
		hintColor:   model.Theme.Muted,
	})

	rightContent := lipgloss.JoinVertical(lipgloss.Left, segBar, "", model.iaViewport.View())
	rightPanel := renderTitledPanel(titledPanelOpts{
		icon:        "⏎",
		title:       "preview",
		hintRight:   rightHint,
		content:     rightContent,
		width:       rightW,
		height:      panelH,
		borderColor: rightBorder,
		titleColor:  model.Theme.FG,
		hintColor:   model.Theme.Muted,
	})

	uiElements := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
	return lipgloss.JoinVertical(lipgloss.Left, uiElements)
}

// renderOutputSegmentBar draws a horizontal radio-style selector listing
// the available content segments. The active segment uses a filled disc
// + Secondary color; the rest use an empty disc + Muted color.
//
//	● Final  ○ Change Analyzer  ○ Commit Body  ○ Commit Title
func (model *Model) renderOutputSegmentBar(width int) string {
	theme := model.Theme
	base := theme.AppStyles().Base
	active := base.Foreground(theme.Secondary).Bold(true)
	inactive := base.Foreground(theme.Muted)

	defs := model.outputSegmentDefs()
	parts := make([]string, 0, len(defs)*2)
	for i, d := range defs {
		var item string
		if d.id == model.outputSegment {
			item = active.Render("● " + d.label)
		} else {
			item = inactive.Render("○ " + d.label)
		}
		if i > 0 {
			parts = append(parts, inactive.Render("  "))
		}
		parts = append(parts, item)
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	if lipgloss.Width(bar) > width {
		bar = truncateOutputLine(bar, width)
	}
	return bar
}

// assembleOutputLeftBody composes the report panel content. Important
// values use Theme.Secondary; muted prose uses Theme.Subtle. Each stage
// row includes a token consumption bar plus the technical stage name.
func (model *Model) assembleOutputLeftBody(width int) string {
	theme := model.Theme
	base := theme.AppStyles().Base
	heading := base.Foreground(theme.Primary).Bold(true)
	label := base.Foreground(theme.Subtle)
	value := base.Foreground(theme.Secondary).Bold(true)
	plain := base.Foreground(theme.FG)
	dim := base.Foreground(theme.Subtle)
	chip := base.Foreground(theme.Secondary).Background(theme.Surface).Padding(0, 1).Bold(true)

	lines := []string{}

	// Inputs
	lines = append(lines, heading.Render("Inputs"))
	lines = append(lines, label.Render("tag      ")+" "+chip.Render(orDash(model.commitType)))
	scope := strings.Join(model.commitScopes, ", ")
	if scope == "" {
		scope = orDash(model.commitScope)
	}
	lines = append(lines, label.Render("scope    ")+" "+value.Render(scope))
	if len(model.keyPoints) == 0 {
		lines = append(lines, label.Render("keypoints")+" "+dim.Render("(none)"))
	} else {
		lines = append(lines, label.Render("keypoints"))
		for _, kp := range model.keyPoints {
			lines = append(lines, "  "+plain.Render("• "+truncateOutputLine(kp, width-4)))
		}
	}
	lines = append(lines, "")

	// AI telemetry — find max tokens across active stages so the bars
	// share a normalized scale.
	maxTokens := 0
	for _, st := range model.pipeline.stages {
		if st.HasStats && st.TotalTokens > maxTokens {
			maxTokens = st.TotalTokens
		}
	}

	lines = append(lines, heading.Render("AI telemetry"))
	hasAny := false
	for i, st := range model.pipeline.stages {
		if !st.HasStats {
			continue
		}
		hasAny = true
		title := st.Title
		if title == "" {
			title = []string{"Change Analyzer", "Commit Body", "Commit Title", "Changelog Refiner"}[i]
		}
		header := value.Render(fmt.Sprintf("Stage %d · %s", i+1, title))
		modelLine := label.Render("model   ") + " " + value.Render(orDash(st.StatsModel))
		tokens := fmt.Sprintf("%d in / %d out / %d total",
			st.PromptTokens, st.CompletionTokens, st.TotalTokens)
		tokensLine := label.Render("tokens  ") + " " + value.Render(tokens)
		barW := max(8, width-22)
		bar := renderTokenBar(st.TotalTokens, maxTokens, barW, theme.Secondary, theme.Subtle, base)
		barLine := label.Render("usage   ") + " " + bar
		latency := fmt.Sprintf("queue %s · prompt %s · completion %s · api %s",
			fmtDur(st.QueueTime), fmtDur(st.PromptTime),
			fmtDur(st.CompletionTime), fmtDur(st.APITotalTime))
		latencyLine := label.Render(
			"latency ",
		) + " " + plain.Render(
			truncateOutputLine(latency, width-12),
		)
		reqLine := label.Render(
			"req id  ",
		) + " " + dim.Render(
			truncateOutputLine(orDash(st.RequestID), width-12),
		)
		lines = append(lines, header, modelLine, tokensLine, barLine, latencyLine, reqLine, "")
	}
	if !hasAny {
		lines = append(lines, dim.Render("(no telemetry recorded)"), "")
	}

	// Stage outputs preview
	lines = append(lines, heading.Render("Stage outputs"))
	type stageOut struct {
		name string
		text string
	}
	outs := []stageOut{
		{"Change Analyzer (s1)", model.iaSummaryOutput},
		{"Commit Body (s2)", model.iaCommitRawOutput},
		{"Commit Title (s3)", model.iaTitleRawOutput},
	}
	if model.iaChangelogEntry != "" {
		outs = append(outs, stageOut{"Changelog Refiner (s4)", model.iaChangelogEntry})
	}
	for _, o := range outs {
		if strings.TrimSpace(o.text) == "" {
			lines = append(lines, value.Render(o.name)+" "+dim.Render("(empty)"))
			continue
		}
		lines = append(lines, value.Render(o.name))
		for _, ln := range strings.Split(strings.TrimRight(o.text, "\n"), "\n") {
			lines = append(lines, "  "+plain.Render(truncateOutputLine(ln, width-4)))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// renderTokenBar paints a token-usage bar as `▰▰▰▰▱▱▱▱  1234`. Width is
// the cell budget for the bar+label combined; the numeric tail always
// stays visible so a 0-cell bar is still informative.
func renderTokenBar(
	tokens, maxTokens, width int,
	fillColor, emptyColor color.Color,
	base lipgloss.Style,
) string {
	tail := fmt.Sprintf(" %d", tokens)
	tailW := lipgloss.Width(tail)
	cells := max(1, width-tailW)
	filled := 0
	if maxTokens > 0 {
		filled = (tokens * cells) / maxTokens
		if filled > cells {
			filled = cells
		}
	}
	fill := base.Foreground(fillColor).Render(strings.Repeat("▰", filled))
	empty := base.Foreground(emptyColor).Render(strings.Repeat("▱", cells-filled))
	return fill + empty + base.Foreground(fillColor).Render(tail)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func truncateOutputLine(s string, width int) string {
	if width < 1 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= width {
			return candidate
		}
	}
	return "…"
}

func fmtDur(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
