package tui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"commit_craft_reborn/internal/tui/statusbar"
	"commit_craft_reborn/internal/tui/styles"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	pairs := [3]struct {
		id  stageID
		out string
		vp  string
	}{
		{stageSummary, model.iaSummaryOutput, model.iaSummaryOutput},
		{stageBody, model.iaCommitRawOutput, model.iaCommitRawOutput},
		{stageTitle, model.iaTitleRawOutput, model.iaTitleRawOutput},
	}
	for _, p := range pairs {
		st := &model.pipeline.stages[p.id]
		if st.Status == statusIdle && strings.TrimSpace(p.out) != "" {
			st.Status = statusDone
			st.Progress = 1
		}
		// Keep the per-stage viewport content in sync with the model so
		// reopening the tab always shows the latest run.
		switch p.id {
		case stageSummary:
			model.pipelineViewport1.SetContent(p.vp)
		case stageBody:
			model.pipelineViewport2.SetContent(p.vp)
		case stageTitle:
			model.pipelineViewport3.SetContent(p.vp)
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

// renderPipelinePanel draws the right column: 3 stage cards (focused one
// is taller), an optional final-commit card, and the diff sub-block as
// the last child.
func (model *Model) renderPipelinePanel(width, height int) string {
	theme := model.Theme
	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)

	cfg := model.globalConfig.TUI.Pipeline
	defaultBody := cfg.StageDefaultHeight
	if defaultBody < 2 {
		defaultBody = 4
	}
	focusedBody := cfg.StageFocusedHeight
	if focusedBody < defaultBody {
		focusedBody = defaultBody + 4
	}
	diffMin := cfg.DiffMinHeight
	if diffMin < 3 {
		diffMin = 6
	}

	// Total card height = body rows + 2 borders + 1 underline row + 1
	// blank spacer between body and underline = body + 4.
	cardH := func(body int) int { return body + 4 }
	defaultCardH := cardH(defaultBody)
	focusedCardH := cardH(focusedBody)
	gap := 1

	stagesH := defaultCardH*2 + focusedCardH + gap*2
	finalCardH := 0
	showFinal := model.pipeline.allDone() && strings.TrimSpace(model.commitTranslate) != ""
	if showFinal {
		finalCardH = pipelineFinalCardH
		stagesH += gap + finalCardH
	}

	innerH := max(1, height-2)
	diffH := max(diffMin, innerH-stagesH-gap)
	if diffH > innerH-stagesH-gap {
		diffH = innerH - stagesH - gap
	}
	if diffH < 0 {
		diffH = 0
	}

	parts := make([]string, 0, 8)
	for i := 0; i < 3; i++ {
		body := defaultBody
		if model.pipeline.focusedStage == stageID(i) {
			body = focusedBody
		}
		parts = append(parts, model.renderStageCard(i, innerW, cardH(body), body))
		if i < 2 {
			parts = append(parts, "")
		}
	}

	if showFinal {
		parts = append(parts, "", model.renderFinalCommitCard(innerW))
	}

	if diffH >= 3 {
		parts = append(parts, "", model.renderDiffSubBlock(innerW, diffH))
	}

	content := strings.Join(parts, "\n")

	return renderTitledPanel(titledPanelOpts{
		title:       "pipeline · 3 stages",
		content:     content,
		width:       width,
		height:      height,
		borderColor: theme.Subtle,
		titleColor:  theme.AI,
		hintColor:   theme.Muted,
	})
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

	vp := model.stageViewportModel(stageID(idx))
	vp.SetWidth(innerW)
	vp.SetHeight(bodyRows)
	bodyRendered := vp.View()

	// Some viewports return fewer rows than requested when the source is
	// empty. Pad so the underline stays anchored to the bottom border.
	bodyLineCount := strings.Count(bodyRendered, "\n") + 1
	if bodyLineCount < bodyRows {
		bodyRendered += strings.Repeat("\n", bodyRows-bodyLineCount)
	}

	content := bodyRendered + "\n" + bar

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

const pipelineFinalCardH = 4

func (model *Model) renderFinalCommitCard(width int) string {
	theme := model.Theme
	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)

	first := strings.TrimSpace(strings.SplitN(model.commitTranslate, "\n", 2)[0])
	first = ansi.Truncate(first, innerW, "…")

	var fg color.Color
	switch model.pipeline.fadeFrame {
	case 0:
		fg = theme.Muted
	case 1:
		fg = theme.AcceptDim
	default:
		fg = theme.Success
	}

	body := lipgloss.NewStyle().Foreground(fg).Bold(true).Render(first)
	hint := lipgloss.NewStyle().Foreground(theme.AI).Bold(true).Render("⏎ accept & commit")

	return renderTitledPanel(titledPanelOpts{
		icon:        "●",
		iconColor:   theme.Success,
		title:       "final commit ready",
		hintRight:   hint,
		hintRaw:     true,
		content:     body,
		width:       width,
		height:      pipelineFinalCardH,
		borderColor: theme.Success,
		titleColor:  theme.FG,
		hintColor:   theme.AI,
	})
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
	}
	return nil
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
		tail := lipgloss.NewStyle().Foreground(theme.Subtle).Render(strings.Repeat(full, width-filled))
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
