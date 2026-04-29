package tui

import (
	"fmt"

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
	p.width = width
	p.height = height
	innerH := height - 1 // header row
	if innerH < 1 {
		innerH = 1
	}
	rightW := width - dualPanelLeftWidth - 1 // divider
	if rightW < 10 {
		rightW = 10
	}
	p.bodyVP.SetWidth(rightW)
	p.bodyVP.SetHeight(innerH)
	p.stageVP.SetWidth(rightW)
	p.stageVP.SetHeight(innerH)
	p.refreshContent()
}

// SetCommit re-hydrates both modes against a new commit.
func (p *HistoryDualPanel) SetCommit(c storage.Commit) {
	p.commit = c
	p.hasCommit = true
	p.keypoints = c.KeyPoints
	if p.keypointIndex >= len(p.keypoints) {
		p.keypointIndex = 0
	}
	p.stages = []historyStage{
		{idx: 1, name: "summary", output: c.IaSummary},
		{idx: 2, name: "body", output: c.IaCommitRaw},
		{idx: 3, name: "title", output: c.IaTitle},
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
		p.stageVP.SetContent(p.renderText(p.stages[p.stageIndex].output, p.stageVP.Width()))
	}
	p.stageVP.GotoTop()
}

func (p HistoryDualPanel) Update(msg tea.Msg) (HistoryDualPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "pgup":
			if p.mode == ModeKeyPointsBody {
				p.bodyVP.ScrollUp(p.bodyVP.Height() / 2)
			} else {
				p.stageVP.ScrollUp(p.stageVP.Height() / 2)
			}
			return p, nil
		case "pgdown":
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
	p.stageIndex = clampInt(p.stageIndex+delta, 0, len(p.stages)-1)
	if p.stageIndex >= 0 && p.stageIndex < len(p.stages) {
		p.stageVP.SetContent(p.renderText(p.stages[p.stageIndex].output, p.stageVP.Width()))
		p.stageVP.GotoTop()
	}
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
	rightCol := lipgloss.JoinVertical(lipgloss.Left, rightHeader, rightBody)

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
		if i == p.stageIndex {
			nameStyle = lipgloss.NewStyle().Foreground(p.theme.FG).Bold(true)
		}
		head := fmt.Sprintf("[%d]  %s", s.idx, nameStyle.Render(s.name))
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
