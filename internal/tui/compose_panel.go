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

	// top edge: ╭─<title><filler><hint>─╮
	// chars: ╭(1) ─(1) title fillerN hint ─(1) ╮(1) → width
	fillerN := o.width - 4 - titleW - hintW
	if fillerN < 1 {
		fillerN = 1
	}
	topLine := bs.Render(topLeft+horizontal) +
		titleRendered +
		bs.Render(strings.Repeat(horizontal, fillerN)) +
		hintRendered +
		bs.Render(horizontal+topRight)

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
