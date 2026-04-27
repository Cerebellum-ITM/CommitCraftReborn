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
	l := NewCommitTypeList(types, typeFormat)
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
	innerHeight := max(4, m.height-boxStyle.GetVerticalFrameSize()-1)

	title := m.theme.AppStyles().Base.
		Foreground(m.theme.Secondary).
		Bold(true).
		Render("Commit type")

	m.list.SetWidth(innerWidth)
	m.list.SetHeight(innerHeight - 1)

	body := lipgloss.JoinVertical(lipgloss.Left, title, m.list.View())
	return tea.NewView(boxStyle.Render(body))
}

// keyTypePopup is the global shortcut that opens the commit-type popup
// from inside the writing-message state. Defined next to the popup so the
// binding lives with the feature.
var keyTypePopup = key.NewBinding(
	key.WithKeys("ctrl+t"),
	key.WithHelp("ctrl+t", "Edit commit type"),
)
