package tui

import (
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// HistoryDualMode picks which view the DualPanel is rendering on the right
// half of the History layout.
type HistoryDualMode int

const (
	ModeKeyPointsBody HistoryDualMode = iota
	ModeStagesResponse
)

// HistoryModeBar renders the segmented mode switcher:
//
//	[●  KeyPoints / Body]  [○  Stages / Response]              ⌃M swap
type HistoryModeBar struct {
	theme *styles.Theme
	mode  HistoryDualMode
	width int
}

func NewHistoryModeBar(theme *styles.Theme) HistoryModeBar {
	return HistoryModeBar{theme: theme}
}

func (m *HistoryModeBar) SetSize(width int)            { m.width = width }
func (m *HistoryModeBar) SetMode(mode HistoryDualMode) { m.mode = mode }
func (m HistoryModeBar) Mode() HistoryDualMode         { return m.mode }
func (m *HistoryModeBar) Toggle() {
	if m.mode == ModeKeyPointsBody {
		m.mode = ModeStagesResponse
	} else {
		m.mode = ModeKeyPointsBody
	}
}

func (m HistoryModeBar) renderPill(label string, active bool) string {
	dot := "○"
	if active {
		dot = "●"
	}
	borderColor := m.theme.Subtle
	textColor := m.theme.Muted
	dotColor := m.theme.Muted
	bg := lipgloss.Color("")
	if active {
		borderColor = m.theme.Primary
		textColor = m.theme.Primary
		dotColor = m.theme.Primary
		bg = lipgloss.Color("")
	}

	dotStyle := lipgloss.NewStyle().Foreground(dotColor).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(textColor)

	inner := lipgloss.JoinHorizontal(
		lipgloss.Top,
		dotStyle.Render(dot),
		"  ",
		textStyle.Render(label),
	)

	pill := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 2)
	if active {
		pill = pill.Background(m.theme.Surface)
		_ = bg
	}
	return pill.Render(inner)
}

func (m HistoryModeBar) View() string {
	left := m.renderPill("KeyPoints / Body", m.mode == ModeKeyPointsBody)
	right := m.renderPill("Stages / Response", m.mode == ModeStagesResponse)
	hint := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("⌃M swap")

	pills := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	pillsW := lipgloss.Width(pills)
	hintW := lipgloss.Width(hint)
	pad := m.width - pillsW - hintW
	if pad < 1 {
		pad = 1
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		pills,
		lipgloss.NewStyle().Width(pad).Render(""),
		hint,
	)
}
