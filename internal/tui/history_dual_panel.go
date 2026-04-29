package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"
)

// dualPanelLeftWidth is the fixed width of the left column (key points /
// stages list). Bumped from 28 to 37 (+30 %) so long key points have
// breathing room and the column lines up with the compose-side keypoints
// input visual ("  > " prompt).
const dualPanelLeftWidth = 37

type historyStage struct {
	idx    int
	name   string
	output string
	model  string
}

// DualPanelRenderFunc styles a body string with the project's commit
// renderer (renderCommitMessage) so the right viewport matches the look of
// the live compose preview without going through glamour. The parent injects
// it from `*Model.renderCommitMessage` so the panel stays decoupled from the
// model.
type DualPanelRenderFunc func(text string, width int) string

// HistoryDualPanel renders the inspection split below the master list. It is
// designed to live inside the surrounding HistoryView frame, so it does NOT
// draw its own outer border — only the vertical divider between the two
// columns and a single header row per side.
type HistoryDualPanel struct {
	theme  *styles.Theme
	mode   HistoryDualMode
	render DualPanelRenderFunc

	width, height int

	commit    storage.Commit
	hasCommit bool

	keypoints     []string
	keypointIndex int
	bodyVP        viewport.Model

	stages     []historyStage
	stageIndex int
	stageVP    viewport.Model
	// stageStats caches the per-stage telemetry pulled from ai_calls for the
	// currently inspected commit. The array is keyed by stageID so the
	// telemetry row can render in O(1) when the user cycles through the
	// stages list. Stages without a stored row keep their zero value (and
	// HasStats stays false, which makes renderStageStatsLine return "").
	stageStats [4]pipelineStage
}

func NewHistoryDualPanel(theme *styles.Theme) HistoryDualPanel {
	return HistoryDualPanel{
		theme:   theme,
		bodyVP:  viewport.New(),
		stageVP: viewport.New(),
	}
}

func (p *HistoryDualPanel) SetMode(mode HistoryDualMode)      { p.mode = mode }
func (p *HistoryDualPanel) SetRenderer(r DualPanelRenderFunc) { p.render = r }

func (p *HistoryDualPanel) SetSize(width, height int) {
	if p.width == width && p.height == height {
		return
	}
	p.width = width
	p.height = height
	rightW := width - dualPanelLeftWidth - 1 // divider
	if rightW < 10 {
		rightW = 10
	}
	// Body viewport (KeyPoints/Body mode): only the header consumes a row.
	bodyH := height - 1
	if bodyH < 1 {
		bodyH = 1
	}
	// Stage viewport (Stages/Response mode): the right column reserves
	// rows for the chrome around the viewport — header (1) + blank (1) +
	// rule (1) + blank around the rule (2) + telemetry (1). The viewport
	// gets whatever's left so cycling stages always reveals a clear
	// hierarchy: title → content → separator → telemetry.
	stageH := height - 6
	if stageH < 1 {
		stageH = 1
	}
	p.bodyVP.SetWidth(rightW)
	p.bodyVP.SetHeight(bodyH)
	p.stageVP.SetWidth(rightW)
	p.stageVP.SetHeight(stageH)
	p.refreshContent()
}

// SetCommit re-hydrates both modes against a new commit. The calls slice is
// the full ai_calls payload for the commit (loaded once by the caller) — it
// hydrates the per-stage telemetry rendered above the right viewport in
// Stages/Response mode and is the source of truth for the optional stage 4
// entry: when the changelog refiner ran for this commit, a 4th entry is
// appended to the inspect list. Newer commits also carry the refiner's
// output text in `c.IaChangelog`; legacy rows keep the placeholder behavior.
func (p *HistoryDualPanel) SetCommit(c storage.Commit, calls []storage.AICall) {
	p.commit = c
	p.hasCommit = true
	p.keypoints = c.KeyPoints
	if p.keypointIndex >= len(p.keypoints) {
		p.keypointIndex = 0
	}

	// Reset and rebuild per-stage telemetry from the freshly loaded
	// ai_calls. Stages without a row keep their zero values so
	// renderStageStatsLine returns "" for them.
	for i := range p.stageStats {
		p.stageStats[i] = pipelineStage{ID: stageID(i), Status: statusIdle}
	}
	hasChangelog := false
	for _, call := range calls {
		id, ok := stageIDFromDBLabel(call.Stage)
		if !ok || int(id) < 0 || int(id) >= len(p.stageStats) {
			continue
		}
		st := &p.stageStats[id]
		st.HasStats = true
		st.StatsModel = call.Model
		st.Model = call.Model
		st.PromptTokens = call.PromptTokens
		st.CompletionTokens = call.CompletionTokens
		st.TotalTokens = call.TotalTokens
		st.QueueTime = msToDuration(call.QueueTimeMs)
		st.PromptTime = msToDuration(call.PromptTimeMs)
		st.CompletionTime = msToDuration(call.CompletionTimeMs)
		st.APITotalTime = msToDuration(call.TotalTimeMs)
		st.RequestID = call.RequestID
		st.TPMLimitAtCall = call.TPMLimitAtCall
		st.Latency = st.APITotalTime
		st.Status = statusDone
		st.Progress = 1
		if id == stageChangelog {
			hasChangelog = true
		}
	}
	// Newer commits also flag the changelog stage through the persisted
	// output column even if their ai_calls row is missing (e.g. telemetry
	// flush failed silently).
	if strings.TrimSpace(c.IaChangelog) != "" {
		hasChangelog = true
	}

	p.stages = []historyStage{
		{idx: 1, name: "Change Analyzer", output: c.IaSummary},
		{idx: 2, name: "Commit Body", output: c.IaCommitRaw},
		{idx: 3, name: "Commit Title", output: c.IaTitle},
	}
	if hasChangelog {
		p.stages = append(p.stages, historyStage{
			idx:    4,
			name:   "Changelog Refiner",
			output: c.IaChangelog,
		})
	}
	if p.stageIndex >= len(p.stages) {
		p.stageIndex = 0
	}
	p.refreshContent()
}

func (p *HistoryDualPanel) Clear() {
	p.commit = storage.Commit{}
	p.hasCommit = false
	p.keypoints = nil
	p.keypointIndex = 0
	p.stages = nil
	p.stageIndex = 0
	for i := range p.stageStats {
		p.stageStats[i] = pipelineStage{ID: stageID(i), Status: statusIdle}
	}
	p.bodyVP.SetContent("")
	p.stageVP.SetContent("")
}

func (p *HistoryDualPanel) renderText(text string, width int) string {
	if p.render != nil {
		return p.render(text, width)
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

func (p *HistoryDualPanel) refreshContent() {
	if !p.hasCommit {
		return
	}
	p.bodyVP.SetContent(p.renderText(p.commit.IaCommitRaw, p.bodyVP.Width()))
	p.bodyVP.GotoTop()

	if p.stageIndex >= 0 && p.stageIndex < len(p.stages) {
		out := p.stages[p.stageIndex].output
		if strings.TrimSpace(out) == "" {
			out = lipgloss.NewStyle().
				Foreground(p.theme.Muted).
				Italic(true).
				Render("(stage output not stored)")
		} else {
			out = p.renderText(out, p.stageVP.Width())
		}
		p.stageVP.SetContent(out)
	}
	p.stageVP.GotoTop()
}

func (p HistoryDualPanel) Update(msg tea.Msg) (HistoryDualPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "pgup", "ctrl+up":
			if p.mode == ModeKeyPointsBody {
				p.bodyVP.ScrollUp(p.bodyVP.Height() / 2)
			} else {
				p.stageVP.ScrollUp(p.stageVP.Height() / 2)
			}
			return p, nil
		case "pgdown", "ctrl+down":
			if p.mode == ModeKeyPointsBody {
				p.bodyVP.ScrollDown(p.bodyVP.Height() / 2)
			} else {
				p.stageVP.ScrollDown(p.stageVP.Height() / 2)
			}
			return p, nil
		}
	}
	return p, nil
}

func (p *HistoryDualPanel) CycleLeftCursor(delta int) {
	if p.mode == ModeKeyPointsBody {
		if len(p.keypoints) == 0 {
			return
		}
		p.keypointIndex = clampInt(p.keypointIndex+delta, 0, len(p.keypoints)-1)
		return
	}
	if len(p.stages) == 0 {
		return
	}
	// Stages list wraps around: stepping past the last entry returns to the
	// first one and vice versa. Go's % keeps the sign of the dividend, so
	// add `n` before reducing to keep negative deltas in range.
	n := len(p.stages)
	p.stageIndex = ((p.stageIndex+delta)%n + n) % n
	p.stageVP.SetContent(p.renderText(p.stages[p.stageIndex].output, p.stageVP.Width()))
	p.stageVP.GotoTop()
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (p HistoryDualPanel) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}

	leftW := dualPanelLeftWidth
	if leftW > p.width/2 {
		leftW = p.width / 2
	}
	rightW := p.width - leftW - 1

	var leftHeader, leftBody, rightHeader, rightBody string
	if p.mode == ModeKeyPointsBody {
		leftHeader = p.renderHeader("key points", fmt.Sprintf("%d", len(p.keypoints)), leftW)
		leftBody = p.renderKeyPointsBody(leftW, p.height-1)
		rightHeader = p.renderHeader("commit body", "preview", rightW)
		rightBody = p.bodyVP.View()
	} else {
		leftHeader = p.renderHeader("ai stages", fmt.Sprintf("%d", len(p.stages)), leftW)
		leftBody = p.renderStagesBody(leftW, p.height-1)
		stageName := ""
		if p.stageIndex >= 0 && p.stageIndex < len(p.stages) {
			stageName = fmt.Sprintf("%d.%s", p.stages[p.stageIndex].idx, p.stages[p.stageIndex].name)
		}
		rightHeader = p.renderHeader(stageName, "output", rightW)
		rightBody = p.stageVP.View()
	}

	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftHeader, leftBody)
	var rightCol string
	if p.mode == ModeStagesResponse {
		telemetry := p.renderTelemetryRow(rightW)
		rule := lipgloss.NewStyle().
			Foreground(p.theme.Subtle).
			Render(strings.Repeat("─", rightW))
		// Layout: header → ⏎ → body → ⏎ → rule → ⏎ → telemetry.
		// The blank rows split the panel into three legible bands so the
		// user reads "what stage", then "what it produced", then "what it
		// cost" without the three sections bleeding into each other.
		rightCol = lipgloss.JoinVertical(
			lipgloss.Left,
			rightHeader,
			"",
			rightBody,
			"",
			rule,
			"",
			telemetry,
		)
	} else {
		rightCol = lipgloss.JoinVertical(lipgloss.Left, rightHeader, rightBody)
	}

	// Force every column to exactly (W × p.height). Place pads with
	// spaces both horizontally and vertically so JoinHorizontal can stitch
	// them with the divider without any height/width drift.
	leftStyled := lipgloss.Place(leftW, p.height, lipgloss.Left, lipgloss.Top, leftCol)
	rightStyled := lipgloss.Place(rightW, p.height, lipgloss.Left, lipgloss.Top, rightCol)

	// Build the vertical divider as a column of explicit │ rows joined
	// vertically. JoinHorizontal needs a per-row source to draw a
	// continuous separator; "│\n" repeated in a single string drifts when
	// joined with multi-line columns.
	bars := make([]string, p.height)
	barStyle := lipgloss.NewStyle().Foreground(p.theme.Subtle)
	for i := range bars {
		bars[i] = barStyle.Render("│")
	}
	dividerCol := lipgloss.JoinVertical(lipgloss.Left, bars...)
	divider := lipgloss.Place(1, p.height, lipgloss.Left, lipgloss.Top, dividerCol)

	row := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, divider, rightStyled)
	return lipgloss.Place(p.width, p.height, lipgloss.Left, lipgloss.Top, row)
}

// renderTelemetryRow draws the per-stage telemetry strip that sits between
// the right header and the stage viewport in Stages/Response mode. It maps
// the active list index back to a stageID and reuses renderStageStatsLine
// (the same renderer the live pipeline cards use), so the look stays in
// sync with the compose-side telemetry without duplicating the format.
func (p HistoryDualPanel) renderTelemetryRow(width int) string {
	muted := lipgloss.NewStyle().Foreground(p.theme.Muted).Italic(true).PaddingLeft(1)
	if p.stageIndex < 0 || p.stageIndex >= len(p.stages) {
		return muted.Render("(no stage selected)")
	}
	id := stageID(p.stages[p.stageIndex].idx - 1)
	if int(id) < 0 || int(id) >= len(p.stageStats) {
		return muted.Render("(stage out of range)")
	}
	st := p.stageStats[id]
	// renderStageStatsLine bails out when there's no stored telemetry —
	// surface a small placeholder so the reserved row never reads empty.
	innerW := width - 2 // PaddingLeft(1) + a trailing margin
	if innerW < 1 {
		innerW = 1
	}
	line := renderStageStatsLine(p.theme, &st, innerW, false)
	if line == "" {
		return muted.Render("(no telemetry stored)")
	}
	return lipgloss.NewStyle().PaddingLeft(1).Render(line)
}

func (p HistoryDualPanel) renderHeader(label, suffix string, width int) string {
	labelStyle := lipgloss.NewStyle().Foreground(p.theme.Primary).Bold(true)
	suffixStyle := lipgloss.NewStyle().Foreground(p.theme.Muted)
	left := labelStyle.Render(label)
	right := suffixStyle.Render(suffix)
	pad := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if pad < 1 {
		pad = 1
	}
	return lipgloss.NewStyle().PaddingLeft(1).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			left,
			lipgloss.NewStyle().Width(pad).Render(""),
			right,
		),
	)
}

func (p HistoryDualPanel) renderKeyPointsBody(width, height int) string {
	if len(p.keypoints) == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Foreground(p.theme.Muted).
			PaddingLeft(1).
			Render("(no key points)")
	}
	// Mirror the compose-side KeyPointsInput visual: a leading "  > "
	// prompt rendered in the brand secondary color, with continuation
	// lines using "::: " instead. The cursor on the History side is
	// purely visual (no input editing), so we just bold the active
	// keypoint's text.
	prompt := lipgloss.NewStyle().Foreground(p.theme.Secondary).Render("  > ")
	promptW := lipgloss.Width(prompt)

	var lines []string
	for i, kp := range p.keypoints {
		style := lipgloss.NewStyle().Foreground(p.theme.Muted)
		if i == p.keypointIndex {
			style = lipgloss.NewStyle().Foreground(p.theme.FG).Bold(true)
		}
		text := TruncateString(kp, width-promptW-1)
		lines = append(lines, prompt+style.Render(text))
		if len(lines) >= height {
			break
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (p HistoryDualPanel) renderStagesBody(width, height int) string {
	if len(p.stages) == 0 {
		return ""
	}
	var lines []string
	for i, s := range p.stages {
		nameStyle := lipgloss.NewStyle().Foreground(p.theme.Muted)
		glyphColor := p.theme.Muted
		if i == p.stageIndex {
			nameStyle = lipgloss.NewStyle().Foreground(p.theme.FG).Bold(true)
			glyphColor = p.theme.Secondary
		}
		glyph := lipgloss.NewStyle().Foreground(glyphColor).Bold(true).Render("✦")
		head := fmt.Sprintf("%s [%d]  %s", glyph, s.idx, nameStyle.Render(s.name))
		lines = append(lines, lipgloss.NewStyle().PaddingLeft(1).Render(head))
		if s.model != "" {
			modelStyle := lipgloss.NewStyle().Foreground(p.theme.Primary)
			lines = append(
				lines,
				lipgloss.NewStyle().PaddingLeft(6).Render(modelStyle.Render(s.model)),
			)
		}
		if len(lines) >= height {
			break
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
