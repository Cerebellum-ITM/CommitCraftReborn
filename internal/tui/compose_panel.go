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

	titleText := strings.TrimSpace(o.title)
	if o.icon != "" {
		titleText = o.icon + " " + titleText
	}
	titleRendered := ""
	titleW := 0
	if titleText != "" {
		titleRendered = titleStyle.Render(" " + titleText + " ")
		titleW = lipgloss.Width(titleRendered)
	}

	hintRendered := ""
	hintW := 0
	if o.hintRight != "" {
		hintRendered = hintStyle.Render(" " + o.hintRight + " ")
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
