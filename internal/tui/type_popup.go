package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/tui/styles"
)

// setCommitTypeMsg is fired by the type popup when the user picks a tag.
type setCommitTypeMsg struct {
	tag   string
	color string
}

// closeTypePopupMsg dismisses the type popup without changing anything.
type closeTypePopupMsg struct{}

// commitTypePopupModel is the in-place editor for the commit-type pill in
// the compose view. Wraps a fresh CommitTypeList so the popup doesn't
// share filter/cursor state with the deprecated stateChoosingType screen.
type commitTypePopupModel struct {
	list          list.Model
	width, height int
	theme         *styles.Theme
}

func newCommitTypePopup(
	types []commit.CommitType,
	typeFormat string,
	width, height int,
	theme *styles.Theme,
) commitTypePopupModel {
	l := NewCommitTypeList(types, typeFormat, theme)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetWidth(width)
	l.SetHeight(height)
	// The popup intercepts enter / esc / arrow keys directly; clear the
	// list's accept/cancel bindings so "/" and "enter" are not consumed
	// by the filter state machine.
	l.KeyMap.AcceptWhileFiltering = key.NewBinding()
	l.KeyMap.CancelWhileFiltering = key.NewBinding()
	// Filter is always-on: SetFilterText empties the input and refreshes
	// items; SetFilterState forces "Filtering" so handleFiltering routes
	// keystrokes into FilterInput (SetFilterText alone leaves the list
	// in FilterApplied where printables are ignored).
	l.SetFilterText("")
	l.SetFilterState(list.Filtering)
	return commitTypePopupModel{
		list:   l,
		width:  width,
		height: height,
		theme:  theme,
	}
}

func (m commitTypePopupModel) Init() tea.Cmd { return nil }

func (m commitTypePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch km := msg.(type) {
	case tea.KeyMsg:
		switch km.String() {
		case "esc":
			// First esc clears the filter; second esc closes the popup.
			if m.list.FilterInput.Value() != "" {
				m.list.SetFilterText("")
				m.list.SetFilterState(list.Filtering)
				return m, nil
			}
			return m, func() tea.Msg { return closeTypePopupMsg{} }
		case "enter":
			selected, ok := m.list.SelectedItem().(CommitTypeItem)
			if !ok {
				return m, nil
			}
			tag := selected.Title()
			col := selected.Color()
			return m, func() tea.Msg {
				return setCommitTypeMsg{tag: tag, color: col}
			}
		case "up":
			// In Filtering state bubbles/list forwards arrows to
			// FilterInput, so drive the cursor manually here.
			m.list.CursorUp()
			return m, nil
		case "down":
			m.list.CursorDown()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m commitTypePopupModel) View() tea.View {
	boxStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	innerWidth := max(20, m.width-boxStyle.GetHorizontalFrameSize())
	innerHeight := max(6, m.height-boxStyle.GetVerticalFrameSize())

	base := m.theme.AppStyles().Base
	title := base.
		Foreground(m.theme.Secondary).
		Bold(true).
		Render("Commit type")

	keyStyle := base.Foreground(m.theme.Accent)
	descStyle := base.Foreground(m.theme.Muted)
	sep := descStyle.Render(" · ")
	hint := lipgloss.JoinHorizontal(lipgloss.Top,
		descStyle.Render("type to filter"),
		sep,
		keyStyle.Render("↑↓"), descStyle.Render(" nav"),
		sep,
		keyStyle.Render("enter"), descStyle.Render(" pick"),
		sep,
		keyStyle.Render("esc"), descStyle.Render(" clear/close"),
	)

	listH := max(3, innerHeight-lipgloss.Height(title)-lipgloss.Height(hint)-2)
	m.list.SetWidth(innerWidth)
	m.list.SetHeight(listH)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		m.list.View(),
		"",
		hint,
	)
	return tea.NewView(boxStyle.Render(body))
}

// CommitTypePopupContentWidth returns the minimum width the popup
// needs in order to render every item without truncation. It mirrors
// the row shape used by CommitTypeDelegate.Render
// ("❯ [chip] <desc>"), plus a small margin for the box padding/border
// so the caller can decide whether to widen the popup.
func CommitTypePopupContentWidth(types []commit.CommitType, typeFormat string) int {
	const prefix = 2                                 // "❯ " or "  "
	const chip = styles.CommitTypeChipInnerWidth + 2 // Width + Padding(0,1)
	const sep = 1                                    // " " between chip and desc
	const margin = 8                                 // box padding + border + breathing room
	_ = typeFormat                                   // no longer used by the row shape
	maxW := 0
	for _, ct := range types {
		w := prefix + chip + sep + lipgloss.Width(ct.Description)
		if w > maxW {
			maxW = w
		}
	}
	return maxW + margin
}

// keyTypePopup is the global shortcut that opens the commit-type popup
// from inside the writing-message state. Defined next to the popup so the
// binding lives with the feature.
var keyTypePopup = key.NewBinding(
	key.WithKeys("ctrl+t"),
	key.WithHelp("ctrl+t", "Edit commit type"),
)
