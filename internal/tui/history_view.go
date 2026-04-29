package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"
)

// HistoryView is the orchestration layer wrapping the FilterBar + ModeBar +
// DualPanel that surround the master commit list. The whole view sits inside
// a single rounded outer frame; the inner sections are stacked and separated
// by horizontal rules so every component reads as a slice of the same panel.
//
// The master list itself (`Model.mainList`) stays where it is so all existing
// handlers in update_commit.go keep working unchanged — HistoryView only
// owns the new chrome and the inspection split.
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

// SetBodyRenderer wires the project's commit-message renderer into the dual
// panel so the right viewport styles the AI body using the same code path
// as the live compose view.
func (h *HistoryView) SetBodyRenderer(r DualPanelRenderFunc) { h.dualPanel.SetRenderer(r) }

func (h *HistoryView) SetCommit(c storage.Commit, hasChangelog bool) {
	h.dualPanel.SetCommit(c, hasChangelog)
}
func (h *HistoryView) ClearCommit() { h.dualPanel.Clear() }

func (h *HistoryView) IsFilterFocused() bool { return h.filterBar.IsFocused() }
func (h *HistoryView) FocusFilter() tea.Cmd  { return h.filterBar.Focus() }
func (h *HistoryView) BlurFilter()           { h.filterBar.Blur() }
func (h *HistoryView) ResetFilter()          { h.filterBar.Reset() }
func (h *HistoryView) FilterValue() string   { return h.filterBar.Value() }
func (h *HistoryView) CycleFilterMode()      { h.filterBar.CycleMode() }
func (h *HistoryView) SetCounts(visible, total int) {
	h.filterBar.SetCounts(visible, total)
}

func (h *HistoryView) ToggleMode() {
	h.modeBar.Toggle()
	h.dualPanel.SetMode(h.modeBar.Mode())
}

func (h *HistoryView) UpdateFilter(msg tea.Msg) (tea.Cmd, bool) {
	prev := h.filterBar.Value()
	var cmd tea.Cmd
	h.filterBar, cmd = h.filterBar.Update(msg)
	return cmd, h.filterBar.Value() != prev
}

func (h *HistoryView) UpdatePanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	h.dualPanel, cmd = h.dualPanel.Update(msg)
	return cmd
}

func (h *HistoryView) CycleLeftCursor(delta int) { h.dualPanel.CycleLeftCursor(delta) }

// outerFrame is the rounded border drawn around the whole history view.
func (h HistoryView) outerFrame() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(h.theme.Primary)
}

// innerWidth returns the width available for content sections after the outer
// rounded frame consumes its 2-column horizontal frame. All inner components
// (FilterBar, MasterList, ModeBar, DualPanel) render at this exact width so
// they line up perfectly under the same vertical pipes.
func (h HistoryView) innerWidth(totalWidth int) int {
	w := totalWidth - h.outerFrame().GetHorizontalFrameSize()
	if w < 20 {
		w = 20
	}
	return w
}

// dividerLine renders a single horizontal rule of width characters using the
// frame color so the inner sections read as visually separated.
func (h HistoryView) dividerLine(width int) string {
	return lipgloss.NewStyle().
		Foreground(h.theme.Subtle).
		Render(strings.Repeat("─", width))
}

// MasterListSize is the size the parent should pass to model.mainList.SetSize
// before rendering it and feeding the result into View. Mirrors the math
// inside View so chrome and the list stay aligned.
func (h *HistoryView) MasterListSize(width, totalHeight int) (int, int) {
	innerW := h.innerWidth(width)
	frameH := h.outerFrame().GetVerticalFrameSize()
	h.modeBar.SetSize(innerW)
	modeRowsBudget := lipgloss.Height(h.modeBar.View())
	available := totalHeight - frameH - 1 /*filter*/ - modeRowsBudget - 3 /*dividers*/
	if available < 6 {
		available = 6
	}
	listH := available / 2
	if listH < 3 {
		listH = 3
	}
	return innerW, listH
}

// View composes the four inner sections inside one rounded outer frame:
//
//	┌──────────── filter row ────────────┐
//	│ › filter [...]              n/total│
//	│ ─────────────────────────────────  │
//	│ master list rows…                  │
//	│ ─────────────────────────────────  │
//	│ [● KP/Body]  [○ Stages/Out]    ⌃E  │
//	│ ─────────────────────────────────  │
//	│ key points     │ commit body       │
//	│ › item         │ wrapped body…     │
//	└────────────────────────────────────┘
func (h *HistoryView) View(masterListView string, width, totalHeight int) string {
	innerW := h.innerWidth(width)
	frameH := h.outerFrame().GetVerticalFrameSize()

	h.filterBar.SetSize(innerW)
	h.modeBar.SetSize(innerW)

	// Modebar with bordered pills is 3 rows tall; account for that in the
	// height budget so the pills fit without squeezing the dual panel.
	modeRowsBudget := lipgloss.Height(h.modeBar.View())
	available := totalHeight - frameH - 1 /*filter*/ - modeRowsBudget - 3 /*dividers*/
	if available < 6 {
		available = 6
	}
	listH := available / 2
	panelH := available - listH
	if listH < 3 {
		listH = 3
	}
	if panelH < 3 {
		panelH = 3
	}

	h.dualPanel.SetSize(innerW, panelH)

	// PlaceHorizontal pads each rendered section to exactly innerW chars
	// per row. Without this, a section that returns a row narrower than
	// innerW makes JoinVertical pad inconsistently and the outer frame's
	// right pipe drifts. Master list and dual panel additionally force
	// their height via Place so the vertical layout is exact.
	filterRow := lipgloss.PlaceHorizontal(innerW, lipgloss.Left, h.filterBar.View())
	masterRow := lipgloss.Place(innerW, listH, lipgloss.Left, lipgloss.Top, masterListView)
	modeRow := lipgloss.PlaceHorizontal(innerW, lipgloss.Left, h.modeBar.View())
	panelRow := lipgloss.Place(innerW, panelH, lipgloss.Left, lipgloss.Top, h.dualPanel.View())

	divider := lipgloss.NewStyle().
		Foreground(h.theme.Subtle).
		Render(strings.Repeat("─", innerW))

	stack := lipgloss.JoinVertical(
		lipgloss.Left,
		filterRow,
		divider,
		masterRow,
		divider,
		modeRow,
		divider,
		panelRow,
	)

	// IMPORTANT: pass the *total* width (= innerW + border) to Width().
	// lipgloss Style.Render subtracts horizontalBorderSize from the
	// passed width before wrapping content, so Width(innerW) would wrap
	// the inner stack at innerW-2 cells and split every row into two.
	// We want the rendered total to be `width` cells, with content area
	// = innerW. Same reasoning applies to Height for the vertical axis —
	// without it the frame ignores totalHeight and lets the stack
	// overflow downward.
	return h.outerFrame().Width(width).Height(totalHeight).Render(stack)
}
