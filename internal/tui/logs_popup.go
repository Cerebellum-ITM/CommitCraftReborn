package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// logLineMsg is emitted by the logger subscription command whenever a new log
// line lands in the ring buffer. The TUI appends it to the popup viewport.
type logLineMsg struct{ line string }

// logsChannelClosedMsg signals that the subscription channel was closed and
// that no further log lines will arrive.
type logsChannelClosedMsg struct{}

// logsPopupWidth and logsPopupHeight compute the popup dimensions relative to
// the terminal size. We cap the width so wide terminals don't render an
// unreadable single-line block.
func (model *Model) logsPopupSize() (int, int) {
	w := model.width - 8
	if w > 140 {
		w = 140
	}
	if w < 40 {
		w = 40
	}
	h := model.height - 6
	if h < 10 {
		h = 10
	}
	return w, h
}

// refreshLogsViewport rewrites the viewport content from the ring buffer
// snapshot and sticks the view to the bottom so the latest line is visible.
func (model *Model) refreshLogsViewport() {
	lines := model.log.Snapshot()
	model.logViewport.SetContent(strings.Join(lines, "\n"))
	model.logViewport.GotoBottom()
}

// renderLogsPopup returns the styled popup ready to be placed on top of the
// main view. The outer border uses the theme accent colours so it matches the
// rest of the TUI.
func (model *Model) renderLogsPopup() string {
	w, h := model.logsPopupSize()

	model.logViewport.SetWidth(w - 4)
	model.logViewport.SetHeight(h - 4)

	title := model.Theme.AppStyles().Base.
		Foreground(model.Theme.Secondary).
		Bold(true).
		Render("Live logs")

	hint := model.Theme.AppStyles().Base.
		Foreground(model.Theme.FgMuted).
		Render("esc / ctrl+l close · ↑↓ scroll")

	body := model.logViewport.View()

	inner := lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", hint)

	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(model.Theme.BorderFocus).
		Render(inner)
}
