package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// braille8Levels is the smoothing ramp used by renderThinQuotaBar. Each
// glyph adds one dot row to the previous one, so as progress crosses a
// cell the boundary "grows" in 8 steps instead of jumping. ⠀ is U+2800
// (BRAILLE PATTERN BLANK), not ASCII space, so cell width stays uniform.
var braille8Levels = [9]rune{
	'⠀', // 0/8 empty
	'⡀', // 1/8
	'⣀', // 2/8
	'⣄', // 3/8
	'⣤', // 4/8
	'⣦', // 5/8
	'⣶', // 6/8
	'⣷', // 7/8
	'⣿', // 8/8 full
}

// renderThinQuotaBar draws a compact "label  ████░░░░  used / limit" row
// used by the compose pipeline-models section and the model picker
// footer. width is the total horizontal budget for the row; the bar
// stretches to fill whatever is left after label and the right-aligned
// usage text. When limit <= 0 the row falls back to a muted "—" placeholder
// so callers can render it for models that haven't been called yet.
func renderThinQuotaBar(
	theme *styles.Theme,
	label string,
	used, limit, width int,
) string {
	base := theme.AppStyles().Base
	const labelWidth = 4

	labelText := strings.ToUpper(strings.TrimSpace(label))
	if len(labelText) > labelWidth {
		labelText = labelText[:labelWidth]
	}
	labelPadded := fmt.Sprintf("%-*s", labelWidth, labelText)
	labelStyled := base.Foreground(theme.Muted).Render(labelPadded)

	if limit <= 0 {
		placeholder := base.Foreground(theme.Subtle).Italic(true).Render("— no data yet")
		return labelStyled + " " + placeholder
	}

	pct := used * 100 / limit
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	usageText := fmt.Sprintf("%s / %s", formatQuantity(used), formatQuantity(limit))
	usageStyled := base.Foreground(theme.Muted).Render(usageText)

	// Width breakdown: label(4) + space(1) + bar + space(1) + usage.
	chrome := lipgloss.Width(labelStyled) + 1 + 1 + lipgloss.Width(usageStyled)
	barW := width - chrome
	if barW < 4 {
		barW = 4
	}
	if barW > 32 {
		barW = 32
	}
	fillColor := theme.Primary
	switch {
	case pct >= 90:
		fillColor = theme.Error
	case pct >= 70:
		fillColor = theme.Warning
	}
	bar := renderBrailleRamp(used, limit, barW, base, fillColor, theme.Subtle)

	return labelStyled + " " + bar + " " + usageStyled
}

// renderBrailleRamp draws a width-cell horizontal bar where each cell has
// 8 sub-levels of fill, so the bar's leading edge grows in 8 steps as
// progress crosses every cell. Filled cells share the foreground color;
// empty cells use the muted subtle color so the track is always visible.
// Empty cells render as the BRAILLE PATTERN BLANK — invisible but
// width-preserving. Callers that want a visible track skeleton should
// use renderBrailleRampWithEmpty instead.
func renderBrailleRamp(
	used, total, width int,
	base lipgloss.Style,
	fill, track color.Color,
) string {
	return renderBrailleRampWithEmpty(used, total, width, base, fill, track, braille8Levels[0])
}

// renderBrailleRampWithEmpty is the same as renderBrailleRamp but lets
// the caller pick the rune used for empty cells. Useful when the bar
// needs to remain visible even at 0% — e.g. the per-stage TPM bar that
// always shows its full extent so the user sees how much of the bucket
// each call could have consumed.
func renderBrailleRampWithEmpty(
	used, total, width int,
	base lipgloss.Style,
	fill, track color.Color,
	emptyRune rune,
) string {
	if width <= 0 || total <= 0 {
		return ""
	}
	if used < 0 {
		used = 0
	}
	if used > total {
		used = total
	}
	totalSubunits := width * 8
	filledSubunits := used * totalSubunits / total
	if filledSubunits < 0 {
		filledSubunits = 0
	}
	if filledSubunits > totalSubunits {
		filledSubunits = totalSubunits
	}
	fullCells := filledSubunits / 8
	partial := filledSubunits % 8

	var b strings.Builder
	b.Grow(width * 4)
	if fullCells > 0 {
		b.WriteString(
			base.Foreground(fill).Render(strings.Repeat(string(braille8Levels[8]), fullCells)),
		)
	}
	emptyCells := width - fullCells
	if partial > 0 && emptyCells > 0 {
		b.WriteString(base.Foreground(fill).Render(string(braille8Levels[partial])))
		emptyCells--
	}
	if emptyCells > 0 {
		b.WriteString(
			base.Foreground(track).Render(strings.Repeat(string(emptyRune), emptyCells)),
		)
	}
	return b.String()
}

// formatQuantity prints an integer with a "k" suffix once it crosses
// 1000 so usage rows stay compact even when limits run into the
// hundreds of thousands of tokens per day.
func formatQuantity(n int) string {
	if n < 0 {
		n = 0
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 10000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000.0)
}
