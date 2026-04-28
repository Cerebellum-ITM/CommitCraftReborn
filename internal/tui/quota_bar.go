package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

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
	filled := barW * pct / 100
	if filled < 0 {
		filled = 0
	}
	if filled > barW {
		filled = barW
	}
	fillColor := theme.Primary
	switch {
	case pct >= 90:
		fillColor = theme.Error
	case pct >= 70:
		fillColor = theme.Warning
	}
	bar := base.Foreground(fillColor).Render(strings.Repeat("█", filled)) +
		base.Foreground(theme.Subtle).Render(strings.Repeat("░", barW-filled))

	return labelStyled + " " + bar + " " + usageStyled
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
