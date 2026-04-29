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
//	[●  KeyPoints / Body]  [○  Stages / Response]              ⌃E swap
//
// The pills use a rounded border (matching the design); the right-hand hint
// is vertically centered against the pill row so it lines up with the dots
// rather than wrapping to a second line at narrow widths.
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
	// The active pill keeps the secondary brand color on its border so
	// it reads as the selected segment; the idle pill drops to Muted to
	// match the dimmed-border convention used elsewhere (scope chip,
	// section pills) and let the active one win the eye.
	borderColor := m.theme.Muted
	textColor := m.theme.Muted
	dotColor := m.theme.Muted
	if active {
		borderColor = m.theme.Secondary
		textColor = m.theme.Primary
		dotColor = m.theme.Primary
	}

	dotStyle := lipgloss.NewStyle().Foreground(dotColor).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(textColor)
	if active {
		textStyle = textStyle.Bold(true)
	}

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
	}
	return pill.Render(inner)
}

func (m HistoryModeBar) View() string {
	left := m.renderPill("KeyPoints / Body", m.mode == ModeKeyPointsBody)
	right := m.renderPill("Stages / Response", m.mode == ModeStagesResponse)
	pills := lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)

	pillsH := lipgloss.Height(pills)
	pillsW := lipgloss.Width(pills)

	// Build the hint as a block that matches the pill height *and* fills
	// the remaining horizontal space. Without matching heights,
	// JoinHorizontal silently drops the spacer on rows ≥1 and the hint
	// slides to column 0 on rows that have no spacer — that is the cause
	// of "swap" wrapping under the active pill.
	rawHint := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("⌃E swap")
	rightWidth := m.width - pillsW
	if rightWidth < 1 {
		rightWidth = 1
	}
	hintBlock := lipgloss.Place(
		rightWidth,
		pillsH,
		lipgloss.Right,
		lipgloss.Center,
		rawHint,
	)

	row := lipgloss.JoinHorizontal(lipgloss.Top, pills, hintBlock)
	return lipgloss.NewStyle().Width(m.width).MaxWidth(m.width).Render(row)
}
