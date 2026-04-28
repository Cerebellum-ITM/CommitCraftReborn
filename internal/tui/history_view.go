package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"
)

// HistoryView is the orchestration layer wrapping the FilterBar + ModeBar +
// DualPanel that surround the master commit list. The master list itself
// (`Model.mainList`) stays where it is so all existing handlers in
// update_commit.go keep working unchanged — HistoryView only owns the new
// chrome and the inspection split below the list.
type HistoryView struct {
	theme     *styles.Theme
	filterBar HistoryFilterBar
	modeBar   HistoryModeBar
	dualPanel HistoryDualPanel
}

func NewHistoryView(theme *styles.Theme) HistoryView {
	return HistoryView{
		theme:     theme,
		filterBar: NewHistoryFilterBar(theme),
		modeBar:   NewHistoryModeBar(theme),
		dualPanel: NewHistoryDualPanel(theme),
	}
}

// SetCommit re-hydrates the inspection panel for the currently selected
// commit. The parent calls this after every selection change in the master
// list.
func (h *HistoryView) SetCommit(c storage.Commit) { h.dualPanel.SetCommit(c) }
func (h *HistoryView) ClearCommit()               { h.dualPanel.Clear() }

// FilterBar accessors used by update_commit.go.
func (h *HistoryView) IsFilterFocused() bool { return h.filterBar.IsFocused() }
func (h *HistoryView) FocusFilter() tea.Cmd  { return h.filterBar.Focus() }
func (h *HistoryView) BlurFilter()           { h.filterBar.Blur() }
func (h *HistoryView) ResetFilter()          { h.filterBar.Reset() }
func (h *HistoryView) FilterValue() string   { return h.filterBar.Value() }
func (h *HistoryView) SetCounts(visible, total int) {
	h.filterBar.SetCounts(visible, total)
}

// ToggleMode flips the DualPanel between KeyPoints/Body and Stages/Response.
func (h *HistoryView) ToggleMode() {
	h.modeBar.Toggle()
	h.dualPanel.SetMode(h.modeBar.Mode())
}

// UpdateFilter feeds keys into the FilterBar (only when focused). Returns the
// resulting cmd along with a flag indicating whether the filter value
// changed, so the parent can re-filter the master list items.
func (h *HistoryView) UpdateFilter(msg tea.Msg) (tea.Cmd, bool) {
	prev := h.filterBar.Value()
	var cmd tea.Cmd
	h.filterBar, cmd = h.filterBar.Update(msg)
	return cmd, h.filterBar.Value() != prev
}

// UpdatePanel forwards messages to the DualPanel (scroll keys, etc.).
func (h *HistoryView) UpdatePanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	h.dualPanel, cmd = h.dualPanel.Update(msg)
	return cmd
}

func (h *HistoryView) CycleLeftCursor(delta int) { h.dualPanel.CycleLeftCursor(delta) }

// Layout: FilterBar (3 rows) / masterListView (flex) / ModeBar (3 rows) / DualPanel (flex).
// Master list and DualPanel share the leftover height roughly 50/50 so both
// surfaces are usable without resizing the terminal.
func (h *HistoryView) View(masterListView string, width, totalHeight int) string {
	h.filterBar.SetSize(width)
	filterRendered := h.filterBar.View()
	filterH := lipgloss.Height(filterRendered)

	h.modeBar.SetSize(width)
	modeRendered := h.modeBar.View()
	modeH := lipgloss.Height(modeRendered)

	remaining := totalHeight - filterH - modeH
	if remaining < 6 {
		remaining = 6
	}
	listH := remaining / 2
	panelH := remaining - listH
	if listH < 3 {
		listH = 3
	}
	if panelH < 3 {
		panelH = 3
	}

	h.dualPanel.SetSize(width, panelH)
	panelRendered := h.dualPanel.View()

	masterPadded := lipgloss.NewStyle().Width(width).Height(listH).Render(masterListView)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		filterRendered,
		masterPadded,
		modeRendered,
		panelRendered,
	)
}

// MasterListSize is the (width, height) the master list should be sized to
// before its View() is rendered and passed to HistoryView.View. Mirrors the
// math in View so the list and the surrounding chrome stay aligned.
func (h *HistoryView) MasterListSize(width, totalHeight int) (int, int) {
	h.filterBar.SetSize(width)
	filterH := lipgloss.Height(h.filterBar.View())
	h.modeBar.SetSize(width)
	modeH := lipgloss.Height(h.modeBar.View())
	remaining := totalHeight - filterH - modeH
	if remaining < 6 {
		remaining = 6
	}
	listH := remaining / 2
	if listH < 3 {
		listH = 3
	}
	return width, listH
}
