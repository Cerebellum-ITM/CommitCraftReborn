package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"
)

// ReleaseHistoryView is the orchestration layer wrapping the release-side
// FilterBar + ModeBar + DualPanel that surround the release master list.
// Mirrors HistoryView for the workspace; the master list itself
// (`Model.releaseMainList`) stays where it is so existing handlers in
// update_release.go keep working unchanged.
type ReleaseHistoryView struct {
	theme     *styles.Theme
	filterBar ReleaseFilterBar
	modeBar   HistoryModeBar
	dualPanel ReleaseDualPanel

	// commitsCache holds the resolved git messages keyed by release ID so
	// re-selecting a release doesn't re-fork-exec `git show` per hash. The
	// map values are the same shape as git.LookupCommitMessages returns,
	// so they can flow straight into SetRelease.
	commitsCache map[int]map[string]git.CommitMessage
	// inFlight tracks release IDs whose commits are currently being
	// fetched in a background goroutine. Used both to stop the prefetch
	// path from doubling up work on the selected entry and to suppress
	// the spinner once the fetch lands.
	inFlight map[int]bool
	// currentReleaseID is the release ID currently displayed in the dual
	// panel. The async fetch handler compares it against the resolved
	// msg's release ID to decide whether to update the dual panel
	// directly or just warm the cache.
	currentReleaseID int
}

func NewReleaseHistoryView(theme *styles.Theme) ReleaseHistoryView {
	mb := NewHistoryModeBar(theme)
	mb.SetLabels("Commits / Body", "Stages / Response")
	return ReleaseHistoryView{
		theme:        theme,
		filterBar:    NewReleaseFilterBar(theme),
		modeBar:      mb,
		dualPanel:    NewReleaseDualPanel(theme),
		commitsCache: make(map[int]map[string]git.CommitMessage),
		inFlight:     make(map[int]bool),
	}
}

// CachedCommits returns the cached git messages for a release ID and a
// flag indicating whether the cache holds an entry. A nil map with
// `ok == true` is impossible because StoreCommits never stores nil.
func (h *ReleaseHistoryView) CachedCommits(releaseID int) (map[string]git.CommitMessage, bool) {
	m, ok := h.commitsCache[releaseID]
	return m, ok
}

// StoreCommits caches the resolved git messages for a release and clears
// the in-flight flag for that ID. Pass an empty (but non-nil) map when
// the lookup returned no commits so the cache distinguishes "fetched,
// nothing found" from "never fetched".
func (h *ReleaseHistoryView) StoreCommits(releaseID int, msgs map[string]git.CommitMessage) {
	if msgs == nil {
		msgs = map[string]git.CommitMessage{}
	}
	h.commitsCache[releaseID] = msgs
	delete(h.inFlight, releaseID)
}

// BeginFetch atomically reserves a release ID for a background fetch.
// Returns true when the caller should actually run the lookup; false
// when the release is already cached or another goroutine is already
// resolving it.
func (h *ReleaseHistoryView) BeginFetch(releaseID int) bool {
	if _, ok := h.commitsCache[releaseID]; ok {
		return false
	}
	if h.inFlight[releaseID] {
		return false
	}
	h.inFlight[releaseID] = true
	return true
}

// CurrentReleaseID returns the release ID currently bound to the dual
// panel. Zero when nothing is selected.
func (h *ReleaseHistoryView) CurrentReleaseID() int { return h.currentReleaseID }

// SetCurrentReleaseID records which release the dual panel is showing.
// Updated by the selection handler before kicking off any async work
// so the resolved-msg handler can tell whether to draw or just cache.
func (h *ReleaseHistoryView) SetCurrentReleaseID(id int) { h.currentReleaseID = id }

// SetBodyRenderer wires the project's commit-message renderer into the
// dual panel so the right viewport renders release bodies with the same
// look as the live compose preview.
func (h *ReleaseHistoryView) SetBodyRenderer(r DualPanelRenderFunc) {
	h.dualPanel.SetRenderer(r)
}

func (h *ReleaseHistoryView) SetRelease(
	r storage.Release,
	messages map[string]git.CommitMessage,
	calls []storage.AICall,
) {
	h.dualPanel.SetRelease(r, messages, calls)
}
func (h *ReleaseHistoryView) ClearRelease() { h.dualPanel.Clear() }

func (h *ReleaseHistoryView) IsFilterFocused() bool { return h.filterBar.IsFocused() }
func (h *ReleaseHistoryView) FocusFilter() tea.Cmd  { return h.filterBar.Focus() }
func (h *ReleaseHistoryView) BlurFilter()           { h.filterBar.Blur() }
func (h *ReleaseHistoryView) ResetFilter()          { h.filterBar.Reset() }
func (h *ReleaseHistoryView) FilterValue() string   { return h.filterBar.Value() }
func (h *ReleaseHistoryView) CycleFilterMode()      { h.filterBar.CycleMode() }
func (h *ReleaseHistoryView) SetCounts(visible, total int) {
	h.filterBar.SetCounts(visible, total)
}

func (h *ReleaseHistoryView) ToggleMode() {
	h.modeBar.Toggle()
	h.dualPanel.SetMode(h.modeBar.Mode())
}

func (h *ReleaseHistoryView) UpdateFilter(msg tea.Msg) (tea.Cmd, bool) {
	prev := h.filterBar.Value()
	var cmd tea.Cmd
	h.filterBar, cmd = h.filterBar.Update(msg)
	return cmd, h.filterBar.Value() != prev
}

func (h *ReleaseHistoryView) CycleLeftCursor(delta int) {
	h.dualPanel.CycleLeftCursor(delta)
}

func (h *ReleaseHistoryView) UpdatePanel(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	h.dualPanel, cmd = h.dualPanel.Update(msg)
	return cmd
}

func (h *ReleaseHistoryView) JumpToRelease() { h.dualPanel.JumpToRelease() }

func (h ReleaseHistoryView) outerFrame() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(h.theme.Primary)
}

func (h ReleaseHistoryView) innerWidth(totalWidth int) int {
	w := totalWidth - h.outerFrame().GetHorizontalFrameSize()
	if w < 20 {
		w = 20
	}
	return w
}

// MasterListSize is the size the parent should pass to
// model.releaseMainList.SetSize before rendering it and feeding the
// result into View. Mirrors the math inside View so chrome and the list
// stay aligned.
func (h *ReleaseHistoryView) MasterListSize(width, totalHeight int) (int, int) {
	innerW := h.innerWidth(width)
	frameH := h.outerFrame().GetVerticalFrameSize()
	h.modeBar.SetSize(innerW)
	modeRowsBudget := lipgloss.Height(h.modeBar.View())
	available := totalHeight - frameH - 1 - modeRowsBudget - 3
	if available < 6 {
		available = 6
	}
	listH := available / 2
	if listH < 3 {
		listH = 3
	}
	return innerW, listH
}

func (h *ReleaseHistoryView) View(masterListView string, width, totalHeight int) string {
	innerW := h.innerWidth(width)
	frameH := h.outerFrame().GetVerticalFrameSize()

	h.filterBar.SetSize(innerW)
	h.modeBar.SetSize(innerW)

	modeRowsBudget := lipgloss.Height(h.modeBar.View())
	available := totalHeight - frameH - 1 - modeRowsBudget - 3
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

	return h.outerFrame().Width(width).Height(totalHeight).Render(stack)
}
