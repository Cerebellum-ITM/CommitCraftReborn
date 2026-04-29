package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

// renderReleaseLoading paints a centered loading panel while the async
// release-history sync is in flight. Designed to read as a deliberate
// frame instead of a glitch: a wide rounded box with the spinner glyph,
// a gradient title, and a small hint about what the lookup is doing.
//
//	╭─────────────────────────────────────╮
//	│                                     │
//	│   ⠋  Loading releases               │
//	│      resolving commit subjects…     │
//	│                                     │
//	╰─────────────────────────────────────╯
func (model *Model) renderReleaseLoading(width, height int) string {
	theme := model.Theme

	// Spinner.View advances on every spinner.TickMsg. The global update
	// handler keeps ticking it while releaseLoading is true, so the
	// glyph rotates in place.
	spinnerGlyph := lipgloss.NewStyle().
		Foreground(theme.Primary).
		Bold(true).
		Render(model.spinner.View())

	titleStyle := lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(theme.Muted).Italic(true)
	hintStyle := lipgloss.NewStyle().Foreground(theme.Muted)

	title := titleStyle.Render("Loading releases")
	subtitle := subtitleStyle.Render("resolving commit subjects…")
	hint := hintStyle.Render(fmt.Sprintf("workspace: %s", TruncatePath(model.pwd, 2)))

	// Two-column body: spinner glyph on the left, stacked text on the
	// right. JoinHorizontal aligns at the top so the spinner sits next
	// to the title row.
	textCol := lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", hint)
	body := lipgloss.JoinHorizontal(lipgloss.Top, spinnerGlyph+"  ", textCol)

	box := lipgloss.NewStyle().
		Padding(1, 3).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Render(body)

	w := width
	if w < 1 {
		w = 1
	}
	h := height
	if h < 1 {
		h = 1
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
