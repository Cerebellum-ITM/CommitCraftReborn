package tui

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	"charm.land/glamour/v2"
	glamourstyles "charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"commit_craft_reborn/internal/tui/statusbar"
	"commit_craft_reborn/internal/tui/styles"
)

// viewPipeline renders the Pipeline tab. Layout (top → bottom, two
// outer columns):
//
//	╭─ changed files ──╮ ╭─ pipeline · 3 stages ──────────╮
//	│ ❯ M layout.go    │ │ ╭─ stage 1 · summary    done ─╮ │
//	│   +62 -4         │ │ │ <viewport content>           │ │
//	│   M model.go     │ │ │ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │ │
//	│   +34 -11        │ │ ╰──────────────────────────────╯ │
//	│ ─────            │ │ ╭─ stage 2 · raw commit  idle ─╮ │
//	│ 5 files +250 -17 │ │ ...                              │
//	╰──────────────────╯ │ ╭─ diff · layout.go · +62 -4 ──╮ │
//	                     │ │ @@ ... @@                    │ │
//	                     │ ╰──────────────────────────────╯ │
//	                     ╰──────────────────────────────────╯
func (model *Model) viewPipeline(width, height int) string {
	model.hydratePipelineFromCompose()

	leftW := max(28, width*30/100)
	rightW := width - leftW - 1
	stacked := width < 90 || rightW < 50

	if stacked {
		filesH := max(8, height/2-1)
		rightH := height - filesH - 1
		left := model.renderFilesPanel(width, filesH)
		right := model.renderPipelinePanel(width, rightH)
		return lipgloss.JoinVertical(lipgloss.Left, left, "", right)
	}

	left := model.renderFilesPanel(leftW, height)
	right := model.renderPipelinePanel(rightW, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

// hydratePipelineFromCompose mirrors persisted compose outputs onto the
// pipeline stages and per-stage viewports. Only touches stages that are
// Idle so an in-flight run isn't disturbed.
func (model *Model) hydratePipelineFromCompose() {
	// Viewports are populated lazily by renderStageCard on each frame, so
	// here we only need to flip stale Idle → Done states for stages whose
	// raw output is already in the model (e.g. when reopening the tab).
	stageIDs := []stageID{stageSummary, stageBody, stageTitle, stageChangelog}
	for _, id := range stageIDs {
		st := &model.pipeline.stages[id]
		if st.Status == statusIdle && strings.TrimSpace(model.stageRawOutput(id)) != "" {
			st.Status = statusDone
			st.Progress = 1
		}
	}
}

// renderFilesPanel draws the left column: the changed-files list with
// status letters and per-row +N -M (rendered by pipelineFilesDelegate),
// plus a footer line with totals.
func (model *Model) renderFilesPanel(width, height int) string {
	theme := model.Theme
	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)

	footer := model.pipelineFilesFooter()
	footerH := lipgloss.Height(footer)
	listH := max(1, height-2-footerH-1) // 2 borders, 1 gap row above footer

	model.pipelineDiffList.SetSize(innerW, listH)
	listView := model.pipelineDiffList.View()

	mutedRule := lipgloss.NewStyle().
		Foreground(theme.Subtle).
		Render(strings.Repeat("─", innerW))

	content := strings.Join([]string{listView, mutedRule, footer}, "\n")

	return renderTitledPanel(titledPanelOpts{
		title:       "changed files",
		content:     content,
		width:       width,
		height:      height,
		borderColor: theme.Subtle,
		titleColor:  theme.FG,
		hintColor:   theme.Muted,
	})
}

// renderPipelinePanel draws the right column: stage cards (collapsed to one
// line each except the focused one, which absorbs all remaining space),
// an optional final-commit card, and the diff sub-block as the last child.
// The diff keeps its configured floor + leftover behavior; the focused
// stage takes whatever's left after collapsed siblings, diff, and final
// card are accounted for.
func (model *Model) renderPipelinePanel(width, height int) string {
	theme := model.Theme
	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)

	cfg := model.globalConfig.TUI.Pipeline
	defaultBody := cfg.StageDefaultHeight
	if defaultBody < 2 {
		defaultBody = 4
	}
	diffMin := cfg.DiffMinHeight
	if diffMin < 3 {
		diffMin = 6
	}
	// User-requested boost: the diff sub-block reserves an extra 20% of
	// the right panel's inner height on top of whatever the config asked
	// for, so reviewing the diff while the focused stage is expanded
	// stays comfortable on tall and short terminals alike.
	innerH := max(1, height-2)
	diffMin += innerH * 20 / 100

	// Card chrome = 2 borders + 1 underline + 1 spacer => +4 over body rows.
	cardH := func(body int) int { return body + 4 }
	gap := 1
	collapsedRows := 1 // single-line summary per non-focused stage

	numStages := model.pipeline.activeStages
	if numStages < 1 {
		numStages = 3
	}

	showFinal := model.pipeline.allDone() && strings.TrimSpace(model.commitTranslate) != ""
	focusedFinal := model.pipeline.focusedFinal && showFinal

	baseFinalBody := 0
	finalCardH := 0
	if showFinal {
		baseFinalBody = computeFinalBodyRows(model.commitTranslate, innerW)
		finalCardH = baseFinalBody + 2
	}

	// When a stage is focused, only N-1 stages collapse. When the final
	// card is focused (or no card is focused), every stage collapses.
	collapsedCount := numStages
	if !focusedFinal {
		collapsedCount = numStages - 1
	}
	collapsedTotal := collapsedRows * collapsedCount
	gapsBetween := gap * (numStages - 1)

	reserved := collapsedTotal + gapsBetween + diffMin + gap
	if showFinal {
		reserved += gap + finalCardH
	}

	focusedCardH := 0
	focusedBody := defaultBody
	finalBodyRows := baseFinalBody

	if focusedFinal {
		// All stages collapsed; the freed slot grows the final card.
		extra := innerH - reserved
		if extra > 0 {
			finalBodyRows = baseFinalBody + extra
			finalCardH = finalBodyRows + 2
		}
	} else {
		focusedCardH = innerH - reserved
		if focusedCardH < cardH(defaultBody) {
			focusedCardH = cardH(defaultBody)
		}
		focusedBody = focusedCardH - 4
		if focusedBody < defaultBody {
			focusedBody = defaultBody
			focusedCardH = cardH(focusedBody)
		}
	}

	// Diff size: configured floor + leftover when the focused card was
	// clamped to its minimum.
	used := collapsedTotal + gapsBetween + gap
	if focusedCardH > 0 {
		used += focusedCardH
	}
	if showFinal {
		used += gap + finalCardH
	}
	diffH := innerH - used
	if diffH < diffMin {
		diffH = diffMin
	}

	parts := make([]string, 0, 8)
	for i := 0; i < numStages; i++ {
		isStageFocused := !focusedFinal && model.pipeline.focusedStage == stageID(i)
		if isStageFocused {
			parts = append(parts, model.renderStageCard(i, innerW, focusedCardH, focusedBody))
		} else {
			parts = append(parts, model.renderStageCardCollapsed(i, innerW))
		}
		if i < numStages-1 {
			parts = append(parts, "")
		}
	}

	if showFinal {
		parts = append(parts, "", model.renderFinalCommitCard(innerW, finalBodyRows, focusedFinal))
	}

	if diffH >= 3 {
		parts = append(parts, "", model.renderDiffSubBlock(innerW, diffH))
	}

	content := strings.Join(parts, "\n")

	return renderTitledPanel(titledPanelOpts{
		title:       fmt.Sprintf("pipeline · %d stages", numStages),
		content:     content,
		width:       width,
		height:      height,
		borderColor: theme.Subtle,
		titleColor:  theme.AI,
		hintColor:   theme.Muted,
	})
}

// stageRawOutput returns the model field that backs each stage's viewport.
// Used by the renderer so the per-stage content stays a pure projection of
// the model with no extra cache to keep in sync.
func (model *Model) stageRawOutput(id stageID) string {
	switch id {
	case stageSummary:
		return model.iaSummaryOutput
	case stageBody:
		return model.iaCommitRawOutput
	case stageTitle:
		return model.iaTitleRawOutput
	case stageChangelog:
		return model.iaChangelogEntry
	}
	return ""
}

// renderStageContent formats a stage's raw output for display. Stage 1
// (the change analyzer summary) flows through Glamour with the Tokyo Night
// theme because its output is full markdown with headings, lists and code
// fences. Stages 2-4 reuse `renderCommitMessage`, the same plain-text +
// inline-code styling the Compose tab applies to the AI suggestion panel —
// commit bodies and title text are NOT real markdown and Glamour mangles
// their indentation and lazy line breaks.
func (model *Model) renderStageContent(id stageID, width int) string {
	raw := model.stageRawOutput(id)
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	if width < 1 {
		width = 1
	}
	if id == stageSummary {
		renderer, err := glamour.NewTermRenderer(
			glamour.WithStyles(glamourstyles.TokyoNightStyleConfig),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return raw
		}
		out, err := renderer.Render(raw)
		if err != nil {
			return raw
		}
		return strings.TrimRight(out, "\n")
	}
	const glamourGutter = 3
	return model.renderCommitMessage(raw, max(1, width-glamourGutter))
}

// renderStageCardCollapsed draws the one-line summary used for non-focused
// stages: status icon, "stage N · title", and the right-aligned status pill.
// No borders, no body — just a compact row that fits exactly the panel
// width so cycling focus only resizes the chosen card.
func (model *Model) renderStageCardCollapsed(idx, width int) string {
	theme := model.Theme
	st := &model.pipeline.stages[idx]

	icon := stageIcon(st.Status, model.pipeline.spinner.View())
	iconStyled := lipgloss.NewStyle().
		Foreground(stageIconColor(model, st.Status)).
		Bold(true).
		Render(icon)

	titleText := fmt.Sprintf("stage %d · %s", idx+1, strings.ToLower(st.Title))
	titleStyle := lipgloss.NewStyle().Foreground(theme.FG).Bold(true)
	if st.Status == statusIdle {
		titleStyle = titleStyle.Foreground(theme.Muted).Bold(false)
	}
	titleStyled := titleStyle.Render(titleText)

	level, label := stageLevelLabel(st.Status)
	pill := statusbar.RenderStatus(level, label)

	left := iconStyled + "  " + titleStyled
	leftW := lipgloss.Width(left)
	pillW := lipgloss.Width(pill)
	gap := width - leftW - pillW
	if gap < 1 {
		gap = 1
	}
	return left + renderStageCardDivider(theme, gap, st.Status) + pill
}

// renderStageCardDivider draws the decorative `─` filler between a
// collapsed stage's title and its right-aligned status pill. It always
// renders exactly `gap` cells so the pill column stays aligned across all
// stages; only the visual treatment changes. Idle stages get a more
// muted line so it doesn't compete with the active stage above.
func renderStageCardDivider(theme *styles.Theme, gap int, status stageStatus) string {
	if gap <= 4 {
		return strings.Repeat(" ", max(1, gap))
	}
	color := theme.Subtle
	if status == statusIdle {
		color = theme.Muted
	}
	line := strings.Repeat("─", gap-2)
	return " " + lipgloss.NewStyle().Foreground(color).Render(line) + " "
}

// computeFinalBodyRows decides how tall the final-commit card's body
// should be. It wraps `commitTranslate` to `innerW`, counts the lines,
// and clamps to a sensible range so the card grows for multi-paragraph
// commits without crowding the diff.
func computeFinalBodyRows(commitMsg string, innerW int) int {
	lines := wrapToLines(commitMsg, innerW, 12)
	rows := len(lines)
	if rows < 3 {
		rows = 3
	}
	if rows > 8 {
		rows = 8
	}
	return rows
}

// renderStageCard draws one stage card. bodyRows controls how many lines
// of the AI output viewport are visible; the stage's progress underline
// always lives in the last inner row of the card.
func (model *Model) renderStageCard(idx, width, height, bodyRows int) string {
	theme := model.Theme
	st := &model.pipeline.stages[idx]
	now := time.Now()

	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)

	bar := renderStageBar(theme, st, model.pipeline.pulsePhase, idx, innerW)

	// When the stage has telemetry to show, reserve the bottom-most body
	// row for the stats line so the card height stays unchanged. The
	// viewport simply renders one fewer line of AI output.
	statsLine := renderStageStatsLine(theme, st, innerW)
	viewportRows := bodyRows
	if statsLine != "" && bodyRows > 1 {
		viewportRows = bodyRows - 1
	}

	vp := model.stageViewportModel(stageID(idx))
	vp.SetWidth(innerW)
	vp.SetHeight(viewportRows)
	// Content is rendered fresh each frame so it adapts to width changes
	// (panel resize, focus growth) and stays in sync with the latest model
	// outputs without a separate cache.
	vp.SetContent(model.renderStageContent(stageID(idx), innerW))
	bodyRendered := vp.View()

	// Some viewports return fewer rows than requested when the source is
	// empty. Pad so the underline stays anchored to the bottom border.
	bodyLineCount := strings.Count(bodyRendered, "\n") + 1
	if bodyLineCount < viewportRows {
		bodyRendered += strings.Repeat("\n", viewportRows-bodyLineCount)
	}

	content := bodyRendered
	if statsLine != "" {
		content += "\n" + statsLine
	}
	content += "\n" + bar

	icon := stageIcon(st.Status, model.pipeline.spinner.View())
	titleText := fmt.Sprintf("stage %d · %s", idx+1, strings.ToLower(st.Title))
	level, label := stageLevelLabel(st.Status)
	pill := statusbar.RenderStatus(level, label)

	borderColor := theme.Subtle
	if model.pipeline.focusedStage == stageID(idx) {
		borderColor = theme.Primary
	}
	if st.Status == statusRunning {
		borderColor = theme.AI
	}
	if st.Status == statusDone && now.Before(st.flashExpiresAt) {
		borderColor = theme.SuccessDim
	}
	if st.Status == statusFailed {
		borderColor = theme.Error
	}

	return renderTitledPanel(titledPanelOpts{
		icon:        icon,
		iconColor:   stageIconColor(model, st.Status),
		title:       titleText,
		hintRight:   pill,
		hintRaw:     true,
		content:     content,
		width:       width,
		height:      height,
		borderColor: borderColor,
		titleColor:  theme.FG,
		hintColor:   theme.Muted,
	})
}

// renderFinalCommitCard renders the assembled commit (title + body) as
// the last sub-block before the diff. bodyRows is the inner row budget
// computed by renderPipelinePanel. The body uses the same plain-text +
// inline-code styling that stages 2-4 (and the Compose AI suggestion
// panel) apply via `renderCommitMessage`, so the user always sees the
// commit message rendered the same way regardless of which card they
// look at.
func (model *Model) renderFinalCommitCard(width, bodyRows int, focused bool) string {
	theme := model.Theme
	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)

	const glamourGutter = 3
	rendered := model.renderCommitMessage(
		strings.TrimSpace(model.commitTranslate),
		max(1, innerW-glamourGutter),
	)

	// Pad to exactly bodyRows so the surrounding panel chrome stays
	// anchored and the diff sub-block below doesn't shift around.
	lineCount := strings.Count(rendered, "\n") + 1
	if lineCount < bodyRows {
		rendered += strings.Repeat("\n", bodyRows-lineCount)
	}

	hint := lipgloss.NewStyle().Foreground(theme.AI).Bold(true).Render("⏎ accept & commit")

	borderColor := theme.Success
	if focused {
		borderColor = theme.Primary
	}

	return renderTitledPanel(titledPanelOpts{
		icon:        "●",
		iconColor:   theme.Success,
		title:       "final commit ready",
		hintRight:   hint,
		hintRaw:     true,
		content:     rendered,
		width:       width,
		height:      bodyRows + 2,
		borderColor: borderColor,
		titleColor:  theme.FG,
		hintColor:   theme.AI,
	})
}

// wrapToLines hard-wraps text into at most maxRows lines fitting within
// width columns. Truncates with an ellipsis on the last line when the
// content overflows.
func wrapToLines(text string, width, maxRows int) []string {
	if width <= 0 || maxRows <= 0 {
		return nil
	}
	out := make([]string, 0, maxRows)
	for _, raw := range strings.Split(text, "\n") {
		if len(out) >= maxRows {
			break
		}
		if raw == "" {
			out = append(out, "")
			continue
		}
		for ansi.StringWidth(raw) > width && len(out) < maxRows-1 {
			cut := width
			if cut > len(raw) {
				cut = len(raw)
			}
			out = append(out, raw[:cut])
			raw = raw[cut:]
		}
		if len(out) < maxRows {
			if ansi.StringWidth(raw) > width {
				raw = ansi.Truncate(raw, width, "…")
			}
			out = append(out, raw)
		}
	}
	return out
}

// renderDiffSubBlock draws the bottom diff card inside the right panel.
// Its content comes from pipeline.diffViewport, which is populated by
// setDiffFromSelectedFile every time the file cursor moves.
func (model *Model) renderDiffSubBlock(width, height int) string {
	theme := model.Theme
	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)

	model.pipeline.diffViewport.SetWidth(innerW)
	model.pipeline.diffViewport.SetHeight(max(1, height-2))

	title := "diff"
	hint := ""
	if it, ok := model.pipelineDiffList.SelectedItem().(DiffFileItem); ok {
		title = "diff · " + it.FilePath
		if ns, found := model.pipeline.numstat[it.FilePath]; found {
			pos := "+0"
			neg := "-0"
			if ns.Adds > 0 {
				pos = fmt.Sprintf("+%d", ns.Adds)
			}
			if ns.Dels > 0 {
				neg = fmt.Sprintf("-%d", ns.Dels)
			}
			hint = lipgloss.NewStyle().Foreground(theme.Add).Render(pos) +
				" " +
				lipgloss.NewStyle().Foreground(theme.Del).Render(neg)
		}
	}

	return renderTitledPanel(titledPanelOpts{
		title:       title,
		hintRight:   hint,
		hintRaw:     hint != "",
		content:     model.pipeline.diffViewport.View(),
		width:       width,
		height:      height,
		borderColor: theme.Subtle,
		titleColor:  theme.FG,
		hintColor:   theme.Muted,
	})
}

// stageViewportModel returns a pointer to the underlying viewport.Model
// for one of the three stages. Used by pgup/pgdn handlers in
// updatePipeline so scroll calls land on the right widget.
func (model *Model) stageViewportModel(id stageID) *viewport.Model {
	switch id {
	case stageSummary:
		return &model.pipelineViewport1
	case stageBody:
		return &model.pipelineViewport2
	case stageTitle:
		return &model.pipelineViewport3
	case stageChangelog:
		return &model.pipelineViewport4
	}
	return nil
}

// renderStageStatsLine renders a single compact telemetry row for a stage:
// "↳ 1.2k tok (in 1.1k · out 87) · 318ms". Returns the empty string when
// no telemetry is available for the stage so the caller can skip the line
// (and the row reservation that goes with it).
func renderStageStatsLine(theme *styles.Theme, st *pipelineStage, width int) string {
	if !st.HasStats || st.TotalTokens <= 0 {
		return ""
	}
	base := theme.AppStyles().Base
	arrow := base.Foreground(theme.Muted).Render("↳ ")
	tokens := base.Foreground(theme.FG).Render(formatTokenCount(st.TotalTokens) + " tok")
	breakdown := base.Foreground(theme.Muted).Render(fmt.Sprintf(
		" (in %s · out %s)",
		formatTokenCount(st.PromptTokens),
		formatTokenCount(st.CompletionTokens),
	))
	sep := base.Foreground(theme.Subtle).Render(" · ")
	dur := st.Latency
	if dur <= 0 && st.APITotalTime > 0 {
		dur = st.APITotalTime
	}
	durStr := base.Foreground(theme.Secondary).Render(formatStageDuration(dur))
	line := arrow + tokens + breakdown + sep + durStr
	if w := ansi.StringWidth(line); w > width && width > 0 {
		line = ansi.Truncate(line, width, "…")
	}
	return line
}

// formatTokenCount renders an integer token count using a "k" suffix once
// it crosses 1000 so the stats line stays compact in the card.
func formatTokenCount(n int) string {
	if n < 0 {
		n = 0
	}
	if n < 1000 {
		return strconv.Itoa(n)
	}
	if n < 10000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	return fmt.Sprintf("%dk", n/1000)
}

// formatStageDuration prints a human-friendly duration matching the
// existing card aesthetic: ms when sub-second, "1.2s" otherwise.
func formatStageDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// renderStageBar draws the thick coloured underline at the bottom of
// every stage card. We deliberately skip bubbles/progress so the stroke
// matches the reference design (solid line, not gradient) and we don't
// have to thread `progress.FrameMsg` through View().
func renderStageBar(theme *styles.Theme, st *pipelineStage, pulsePhase, idx, width int) string {
	if width < 1 {
		return ""
	}
	full := "━"
	switch st.Status {
	case statusDone:
		return lipgloss.NewStyle().Foreground(theme.Success).Render(strings.Repeat(full, width))
	case statusFailed:
		return lipgloss.NewStyle().Foreground(theme.Error).Render(strings.Repeat(full, width))
	case statusCancelled:
		return lipgloss.NewStyle().Foreground(theme.Warning).Render(strings.Repeat(full, width))
	case statusRunning:
		filled := int(float64(width) * indeterminateValue(pulsePhase, idx))
		if filled < 1 {
			filled = 1
		}
		if filled > width {
			filled = width
		}
		head := lipgloss.NewStyle().Foreground(theme.AI).Render(strings.Repeat(full, filled))
		tail := lipgloss.NewStyle().
			Foreground(theme.Subtle).
			Render(strings.Repeat(full, width-filled))
		return head + tail
	}
	return lipgloss.NewStyle().Foreground(theme.Subtle).Render(strings.Repeat(full, width))
}

// indeterminateValue produces a smooth 0..1..0 pulse for the running
// stage's underline. Each stage is phase-shifted so concurrent rows
// don't move in lockstep.
func indeterminateValue(phase, stageIdx int) float64 {
	const period = 24
	p := (phase + stageIdx*8) % period
	half := period / 2
	if p > half {
		p = period - p
	}
	v := float64(p) / float64(half)
	return 0.5 - 0.5*math.Cos(math.Pi*v)
}

// stageIcon picks the leading glyph per status. Running uses the shared
// spinner frame so all running rows share the cadence.
func stageIcon(s stageStatus, spinFrame string) string {
	switch s {
	case statusRunning:
		if spinFrame != "" {
			return spinFrame
		}
		return "●"
	case statusDone:
		return "●"
	case statusFailed:
		return "✗"
	case statusCancelled:
		return "⊘"
	}
	return "○"
}

func stageIconColor(model *Model, s stageStatus) color.Color {
	t := model.Theme
	switch s {
	case statusRunning:
		return t.AI
	case statusDone:
		return t.Success
	case statusFailed:
		return t.Error
	case statusCancelled:
		return t.Warning
	}
	return t.Muted
}

// stageLevelLabel maps a status to (level, label) for the right-aligned
// status pill.
func stageLevelLabel(s stageStatus) (statusbar.LogLevel, string) {
	switch s {
	case statusRunning:
		return statusbar.LevelAI, "running"
	case statusDone:
		return statusbar.LevelSuccess, "done"
	case statusFailed:
		return statusbar.LevelError, "failed"
	case statusCancelled:
		return statusbar.LevelWarning, "cancelled"
	}
	return statusbar.LevelInfo, "idle"
}

// colorizeDiffLine paints +/-/@@ headers in a plain diff. Used by both
// renderDiffSubBlock (via setDiffFromSelectedFile) and any other diff
// preview surfaces that want consistent colours.
func colorizeDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(line)
	case strings.HasPrefix(line, "@@"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(line)
	case strings.HasPrefix(line, "+"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render(line)
	case strings.HasPrefix(line, "-"):
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(line)
	}
	return line
}
