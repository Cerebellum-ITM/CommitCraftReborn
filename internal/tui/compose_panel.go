package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// titledPanelOpts groups the configurable bits of renderTitledPanel so we
// don't grow a multi-positional-argument function as the design evolves.
type titledPanelOpts struct {
	icon        string // optional glyph rendered before the title
	title       string
	hintRight   string
	content     string
	width       int
	height      int
	borderColor color.Color
	titleColor  color.Color
	hintColor   color.Color
	// iconColor, when non-nil, paints the icon with a different color than
	// the title. Used by the Pipeline tab so the per-stage status dot can
	// stay green/red regardless of the title color.
	iconColor color.Color
	// hintRaw, when true, skips the outer hint style so callers can pass
	// pre-styled content (e.g. statusbar pills). The Pipeline tab uses
	// this to embed the per-stage status pill in the top edge.
	hintRaw bool
	// middle is an optional pre-styled string laid out between the title
	// (left) and the hint (right) on the top edge. Used by the Pipeline
	// tab to surface per-stage telemetry (tokens / duration / TPM bar).
	middle string
	// middleRaw mirrors hintRaw — when true, the middle string is
	// embedded verbatim, with no surrounding style applied.
	middleRaw bool
}

// renderTitledPanel draws a rounded-border panel with the title embedded
// in the top edge and a hint on the top-right edge. Used for the
// "summary" and "ai suggestion" panels in the new compose layout.
//
//	╭─⊕ summary ─────────────── press Tab ─╮
//	│   ...content...                       │
//	╰───────────────────────────────────────╯
func renderTitledPanel(o titledPanelOpts) string {
	const (
		topLeft     = "╭"
		topRight    = "╮"
		bottomLeft  = "╰"
		bottomRight = "╯"
		horizontal  = "─"
		vertical    = "│"
	)

	bs := lipgloss.NewStyle().Foreground(o.borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(o.titleColor).Bold(true)
	hintStyle := lipgloss.NewStyle().Foreground(o.hintColor)

	titleRendered := ""
	titleW := 0
	titleText := strings.TrimSpace(o.title)
	switch {
	case o.icon != "" && titleText != "":
		iconColor := o.iconColor
		if iconColor == nil {
			iconColor = o.titleColor
		}
		iconRendered := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(o.icon)
		titleRendered = " " + iconRendered + " " + titleStyle.Render(titleText) + " "
		titleW = lipgloss.Width(titleRendered)
	case titleText != "":
		titleRendered = titleStyle.Render(" " + titleText + " ")
		titleW = lipgloss.Width(titleRendered)
	case o.icon != "":
		iconColor := o.iconColor
		if iconColor == nil {
			iconColor = o.titleColor
		}
		titleRendered = " " + lipgloss.NewStyle().
			Foreground(iconColor).
			Bold(true).
			Render(o.icon) +
			" "
		titleW = lipgloss.Width(titleRendered)
	}

	hintRendered := ""
	hintW := 0
	if o.hintRight != "" {
		if o.hintRaw {
			hintRendered = " " + o.hintRight + " "
		} else {
			hintRendered = hintStyle.Render(" " + o.hintRight + " ")
		}
		hintW = lipgloss.Width(hintRendered)
	}

	middleRendered := ""
	middleW := 0
	if o.middle != "" {
		if o.middleRaw {
			middleRendered = " " + o.middle + " "
		} else {
			middleRendered = hintStyle.Render(" " + o.middle + " ")
		}
		middleW = lipgloss.Width(middleRendered)
	}

	// top edge layout depends on whether a middle section is present:
	//   without middle: ╭─<title><filler><hint>─╮
	//   with middle:    ╭─<title><filler1><middle><filler2><hint>─╮
	// The two fillers are split evenly so the middle reads as centered
	// between the title and the hint.
	topLine := ""
	if middleW == 0 {
		fillerN := o.width - 4 - titleW - hintW
		if fillerN < 1 {
			fillerN = 1
		}
		topLine = bs.Render(topLeft+horizontal) +
			titleRendered +
			bs.Render(strings.Repeat(horizontal, fillerN)) +
			hintRendered +
			bs.Render(horizontal+topRight)
	} else {
		// If the middle is too long to fit alongside title+hint+min-fillers,
		// truncate it (preserving the leading "↳" cue) so the top edge
		// never overflows the panel width.
		const minFillerEachSide = 1
		maxMiddle := o.width - 4 - titleW - hintW - 2*minFillerEachSide
		if maxMiddle < 0 {
			maxMiddle = 0
		}
		if middleW > maxMiddle && maxMiddle > 1 {
			middleRendered = ansi.Truncate(middleRendered, maxMiddle, "…")
			middleW = lipgloss.Width(middleRendered)
		}
		fillerTotal := o.width - 4 - titleW - middleW - hintW
		if fillerTotal < 2 {
			fillerTotal = 2
		}
		filler1 := fillerTotal / 2
		filler2 := fillerTotal - filler1
		topLine = bs.Render(topLeft+horizontal) +
			titleRendered +
			bs.Render(strings.Repeat(horizontal, filler1)) +
			middleRendered +
			bs.Render(strings.Repeat(horizontal, filler2)) +
			hintRendered +
			bs.Render(horizontal+topRight)
	}

	bottomLine := bs.Render(bottomLeft + strings.Repeat(horizontal, o.width-2) + bottomRight)

	innerW := max(1, o.width-2)
	contentW := max(1, innerW-2) // 1 char padding inside each vertical border
	bodyH := max(1, o.height-2)

	rows := strings.Split(o.content, "\n")
	// Pad/truncate to exactly bodyH lines.
	if len(rows) < bodyH {
		for len(rows) < bodyH {
			rows = append(rows, "")
		}
	} else if len(rows) > bodyH {
		rows = rows[:bodyH]
	}

	body := make([]string, 0, bodyH)
	for _, line := range rows {
		w := lipgloss.Width(line)
		switch {
		case w > contentW:
			line = ansi.Truncate(line, contentW, "…")
		case w < contentW:
			line += strings.Repeat(" ", contentW-w)
		}
		body = append(body, bs.Render(vertical)+" "+line+" "+bs.Render(vertical))
	}

	return strings.Join(append([]string{topLine}, append(body, bottomLine)...), "\n")
}

// titledPanelChrome computes the height/width overhead a titled panel
// imposes on its content so callers can budget the inner area correctly.
// Currently 2 rows (top + bottom border) and 2 columns (left + right
// border).
func titledPanelChrome() (cols, rows int) { return 2, 2 }
