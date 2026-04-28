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

const dualPanelLeftWidth = 28

type historyStage struct {
	idx    int
	name   string
	output string
	model  string
}

// HistoryDualPanel renders the inspection split below the master list.
//
//	┌─ key points ──── 3 ─┬─ commit body ──── preview ─────────┐
//	│ › se modifico la… │ Reorganizes the documentation       │
//	│   se reordenaron… │ structure inside docs/ so that the  │
//	│                   │ most frequently referenced sections │
//	└───────────────────┴─────────────────────────────────────┘
//
// In Stages mode the left list is replaced by the 3 persisted IA stages
// (summary, body, title) and the right viewport shows the corresponding
// raw output. Cursor state lives here, not on the parent.
type HistoryDualPanel struct {
	theme *styles.Theme
	mode  HistoryDualMode

	width, height int

	commit    storage.Commit
	hasCommit bool

	// Mode A — KeyPoints / Body
	keypoints     []string
	keypointIndex int
	bodyVP        viewport.Model

	// Mode B — Stages / Response
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

func (p *HistoryDualPanel) SetMode(mode HistoryDualMode) { p.mode = mode }

func (p *HistoryDualPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	innerH := height - 2 - 1 // outer border + header row
	if innerH < 1 {
		innerH = 1
	}
	rightW := width - dualPanelLeftWidth - 3 // left col + vertical divider + frame
	if rightW < 10 {
		rightW = 10
	}
	p.bodyVP.SetWidth(rightW)
	p.bodyVP.SetHeight(innerH)
	p.stageVP.SetWidth(rightW)
	p.stageVP.SetHeight(innerH)
	p.refreshContent()
}

// SetCommit re-hydrates both modes against a new commit. Called by the
// parent whenever the master-list selection changes.
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

// Clear empties the panel — used when the master list has no items.
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

func (p *HistoryDualPanel) refreshContent() {
	if !p.hasCommit {
		return
	}
	// Body wraps to viewport width.
	wrap := lipgloss.NewStyle().Width(p.bodyVP.Width()).Render(p.commit.IaCommitRaw)
	p.bodyVP.SetContent(wrap)
	p.bodyVP.GotoTop()

	if p.stageIndex >= 0 && p.stageIndex < len(p.stages) {
		stageWrap := lipgloss.NewStyle().
			Width(p.stageVP.Width()).
			Render(p.stages[p.stageIndex].output)
		p.stageVP.SetContent(stageWrap)
	}
	p.stageVP.GotoTop()
}

// Update consumes navigation keys when the master list is not focused on
// the FilterBar. The parent decides which keys reach the panel.
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

// CycleLeftCursor moves the cursor in the active left list (keypoints or
// stages) by delta. Triggered by Shift+↑/↓ from the parent so the master
// list keeps the bare arrow keys.
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
		stageWrap := lipgloss.NewStyle().
			Width(p.stageVP.Width()).
			Render(p.stages[p.stageIndex].output)
		p.stageVP.SetContent(stageWrap)
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

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.theme.Subtle)

	innerW := p.width - border.GetHorizontalFrameSize()
	innerH := p.height - border.GetVerticalFrameSize()
	if innerW < 20 || innerH < 3 {
		return border.Width(innerW).Height(innerH).Render("")
	}

	leftW := dualPanelLeftWidth
	if leftW > innerW/2 {
		leftW = innerW / 2
	}
	rightW := innerW - leftW - 1 // -1 for vertical divider

	var leftHeader, leftBody, rightHeader, rightBody string
	if p.mode == ModeKeyPointsBody {
		leftHeader = p.renderHeader("key points", fmt.Sprintf("%d", len(p.keypoints)), leftW)
		leftBody = p.renderKeyPointsBody(leftW, innerH-1)
		rightHeader = p.renderHeader("commit body", "preview", rightW)
		rightBody = p.bodyVP.View()
	} else {
		leftHeader = p.renderHeader("ai stages", fmt.Sprintf("%d", len(p.stages)), leftW)
		leftBody = p.renderStagesBody(leftW, innerH-1)
		stageName := ""
		if p.stageIndex >= 0 && p.stageIndex < len(p.stages) {
			stageName = fmt.Sprintf("%d.%s", p.stages[p.stageIndex].idx, p.stages[p.stageIndex].name)
		}
		rightHeader = p.renderHeader(stageName, "output", rightW)
		rightBody = p.stageVP.View()
	}

	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftHeader, leftBody)
	rightCol := lipgloss.JoinVertical(lipgloss.Left, rightHeader, rightBody)

	leftStyled := lipgloss.NewStyle().Width(leftW).Height(innerH).Render(leftCol)
	rightStyled := lipgloss.NewStyle().Width(rightW).Height(innerH).Render(rightCol)

	divider := lipgloss.NewStyle().
		Foreground(p.theme.Subtle).
		Render(strings.Repeat("│\n", innerH))

	row := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, divider, rightStyled)
	return border.Width(innerW).Height(innerH).Render(row)
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
	var lines []string
	for i, kp := range p.keypoints {
		marker := "  "
		style := lipgloss.NewStyle().Foreground(p.theme.Muted)
		if i == p.keypointIndex {
			marker = "› "
			style = lipgloss.NewStyle().Foreground(p.theme.FG).Bold(true)
		}
		text := TruncateString(kp, width-3)
		lines = append(lines, lipgloss.NewStyle().PaddingLeft(1).Render(marker+style.Render(text)))
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
		idxStyle := lipgloss.NewStyle().Foreground(p.theme.Muted)
		nameStyle := lipgloss.NewStyle().Foreground(p.theme.Muted)
		if i == p.stageIndex {
			nameStyle = lipgloss.NewStyle().Foreground(p.theme.FG).Bold(true)
		}
		head := fmt.Sprintf("[%d]  %s", s.idx, nameStyle.Render(s.name))
		_ = idxStyle
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
