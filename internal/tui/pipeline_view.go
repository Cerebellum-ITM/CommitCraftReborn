package tui

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"commit_craft_reborn/internal/tui/statusbar"

	"charm.land/lipgloss/v2"
)

// viewPipeline renders the Pipeline tab. It hydrates stage status from the
// Compose-tab outputs (iaSummaryOutput / iaCommitRawOutput / iaTitleRawOutput)
// so the user always sees the result of the last AI run, regardless of
// which tab triggered it.
func (model *Model) viewPipeline(width, height int) string {
	model.hydratePipelineFromCompose()

	leftW := max(34, width*38/100)
	rightW := width - leftW - 1
	stacked := width < 90 || rightW < 50

	if stacked {
		filesH := max(6, height/2-1)
		stagesH := height - filesH - 1
		files := model.renderPipelineFilesPanel(width, filesH)
		stages := model.renderPipelineStagesPanel(width, stagesH)
		return lipgloss.JoinVertical(lipgloss.Left, files, "", stages)
	}

	files := model.renderPipelineFilesPanel(leftW, height)
	stages := model.renderPipelineStagesPanel(rightW, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, files, " ", stages)
}

// hydratePipelineFromCompose mirrors the persisted compose outputs onto
// the pipeline stages so the tab feels consistent across switches. Only
// touches stages currently in `Idle`; runs/failures/cancellations win.
func (model *Model) hydratePipelineFromCompose() {
	pairs := [3]struct {
		id  stageID
		out string
	}{
		{stageSummary, model.iaSummaryOutput},
		{stageBody, model.iaCommitRawOutput},
		{stageTitle, model.iaTitleRawOutput},
	}
	for _, p := range pairs {
		if model.pipeline.stages[p.id].Status == statusIdle && strings.TrimSpace(p.out) != "" {
			model.pipeline.stages[p.id].Status = statusDone
			model.pipeline.stages[p.id].Progress = 1
			_ = model.pipeline.progress[p.id].SetPercent(1)
		}
	}
}

func (model *Model) renderPipelineFilesPanel(width, height int) string {
	theme := model.Theme
	borderColor := theme.Subtle
	if model.pipeline.focus == pipelineFocusFiles {
		borderColor = theme.Primary
	}

	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)

	count := len(model.pipelineDiffList.Items())
	hint := fmt.Sprintf("%d files", count)

	model.pipelineDiffList.SetSize(innerW, max(1, height-4))
	content := model.pipelineDiffList.View()

	return renderTitledPanel(titledPanelOpts{
		icon:        theme.AppSymbols().Console,
		title:       "changed files",
		hintRight:   hint,
		content:     content,
		width:       width,
		height:      height,
		borderColor: borderColor,
		titleColor:  theme.FG,
		hintColor:   theme.Muted,
	})
}

func (model *Model) renderPipelineStagesPanel(width, height int) string {
	theme := model.Theme
	borderColor := theme.Subtle
	if model.pipeline.focus == pipelineFocusStages {
		borderColor = theme.Primary
	}

	cw, _ := titledPanelChrome()
	innerW := max(1, width-cw-2)
	innerH := max(1, height-2)

	rows := model.renderStageRows(innerW)
	body := strings.Join(rows, "\n")

	if model.pipeline.allDone() && strings.TrimSpace(model.commitTranslate) != "" {
		final := model.renderPipelineFinalBlock(innerW)
		if final != "" {
			body = body + "\n\n" + final
		}
	}

	// Pad to the inner height
	cur := strings.Count(body, "\n") + 1
	if cur < innerH {
		body += strings.Repeat("\n", innerH-cur)
	}

	hint := "3 stages"
	if model.pipeline.anyRunning() {
		hint = "running…"
	} else if model.pipeline.allDone() {
		hint = "done · ⏎ accept"
	}

	return renderTitledPanel(titledPanelOpts{
		icon:        "",
		title:       "pipeline",
		hintRight:   hint,
		content:     body,
		width:       width,
		height:      height,
		borderColor: borderColor,
		titleColor:  theme.AI,
		hintColor:   theme.Muted,
	})
}

func (model *Model) renderStageRows(innerW int) []string {
	theme := model.Theme
	now := time.Now()

	rows := make([]string, 0, 3*3)
	for i := 0; i < 3; i++ {
		st := &model.pipeline.stages[i]

		icon := stageIcon(st.Status, model.pipeline.spinner.View())
		title := fmt.Sprintf("stage %d/3 · %s", i+1, st.Title)

		level, label := stageLevelLabel(st.Status)
		pill := statusbar.RenderStatus(level, label)

		// Line 1: icon · title · pill
		iconStyled := lipgloss.NewStyle().Foreground(stageIconColor(model, st.Status)).Render(icon)
		titleStyled := lipgloss.NewStyle().Foreground(theme.FG).Bold(st.Status == statusRunning).Render(title)
		head := iconStyled + "  " + titleStyled

		headW := lipgloss.Width(head)
		pillW := lipgloss.Width(pill)
		gap := max(1, innerW-headW-pillW)
		line1 := head + strings.Repeat(" ", gap) + pill

		// Line 2: model · progress · percent
		modelText := lipgloss.NewStyle().Foreground(theme.Muted).Render("model " + st.Model)
		modelW := lipgloss.Width(modelText)
		percent := stagePercentText(st)
		percentStyled := lipgloss.NewStyle().Foreground(theme.Muted).Render(percent)
		percentW := lipgloss.Width(percentStyled)

		barW := max(8, innerW-modelW-percentW-2)
		model.pipeline.progress[i].SetWidth(barW)

		// While running, drive the indeterminate pulse onto the bar.
		if st.Status == statusRunning {
			pulse := indeterminateValue(model.pipeline.pulsePhase, i)
			st.Progress = pulse
			_ = model.pipeline.progress[i].SetPercent(pulse)
		}

		bar := model.pipeline.progress[i].View()
		line2 := modelText + " " + bar + " " + percentStyled

		// Apply per-row decorations: success flash, failure shake.
		line1, line2 = decorateStageRow(st, now, line1, line2)

		rows = append(rows, line1, line2)
		if i < 2 {
			rows = append(rows, "")
		}
	}
	return rows
}

func (model *Model) renderPipelineFinalBlock(innerW int) string {
	theme := model.Theme
	var fg = theme.Muted
	switch model.pipeline.fadeFrame {
	case 0:
		fg = theme.Muted
	case 1:
		fg = theme.AcceptDim
	default:
		fg = theme.Success
	}

	final := strings.TrimSpace(model.commitTranslate)
	if final == "" {
		return ""
	}
	style := lipgloss.NewStyle().Foreground(fg)
	header := lipgloss.NewStyle().Foreground(theme.Muted).Render("─ final commit ─")
	body := style.Render(final)
	rows := strings.Split(body, "\n")
	if len(rows) > 3 {
		rows = rows[:3]
	}
	out := append([]string{header}, rows...)
	_ = innerW
	return strings.Join(out, "\n")
}

// stageIcon picks the leading glyph per status. The Running case takes the
// shared spinner frame so all running rows share the cadence.
func stageIcon(s stageStatus, spinFrame string) string {
	switch s {
	case statusRunning:
		if spinFrame != "" {
			return spinFrame
		}
		return "•"
	case statusDone:
		return "✓"
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

// stageLevelLabel maps a status to the label and statusbar level used for
// the right-aligned status pill on each stage row.
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

func stagePercentText(s *pipelineStage) string {
	switch s.Status {
	case statusRunning:
		return "running…"
	case statusDone:
		if s.Latency > 0 {
			return fmt.Sprintf("%dms", s.Latency.Milliseconds())
		}
		return "100%"
	case statusFailed:
		return "error"
	case statusCancelled:
		return "—"
	}
	return "0%"
}

// indeterminateValue produces a smooth 0..1..0 pulse so the indeterminate
// progress bar visibly moves while we wait for the synchronous AI call.
// Each stage gets a phase offset so concurrent rows don't move in lockstep.
func indeterminateValue(phase, stageIdx int) float64 {
	// Period of 24 ticks (≈1.92s at 80ms cadence). Triangle wave keeps the
	// motion legible without overshoot.
	const period = 24
	p := (phase + stageIdx*8) % period
	half := period / 2
	if p > half {
		p = period - p
	}
	v := float64(p) / float64(half)
	// Clamp & ease — a small sine softens the corners.
	v = 0.5 - 0.5*math.Cos(math.Pi*v)
	return v
}

// decorateStageRow applies the failure shake offset to the rendered stage
// rows. The success flash is currently expressed via the icon/title color
// already chosen above; richer effects can be layered later.
func decorateStageRow(st *pipelineStage, now time.Time, line1, line2 string) (string, string) {
	_ = now
	if st.Status == statusFailed && st.shakeFrame > 0 {
		switch st.shakeFrame {
		case 1:
			line1 = " " + line1
			line2 = " " + line2
		case 2:
			if strings.HasPrefix(line1, " ") {
				line1 = line1[1:]
			}
			if strings.HasPrefix(line2, " ") {
				line2 = line2[1:]
			}
		}
	}
	return line1, line2
}
