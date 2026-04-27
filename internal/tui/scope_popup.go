package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/tui/styles"
)

// setScopeMsg is fired by the scope popup when the user accepts a value.
type setScopeMsg struct{ scope string }

// closeScopePopupMsg dismisses the scope popup without changing anything.
type closeScopePopupMsg struct{}

// scopePopupModel is the in-place editor for the scope pill. It hosts a
// file picker bounded by the git root, mirroring the behaviour of the
// stand-alone stateChoosingScope screen but inside a popup so the user can
// re-pick the scope without leaving the compose view.
type scopePopupModel struct {
	list          list.Model
	pwd           string
	gitData       git.StatusData
	showOnlyMod   bool
	useNerdFonts  bool
	width, height int
	theme         *styles.Theme
}

func newScopePopup(
	startPwd string,
	gitData git.StatusData,
	useNerdFonts bool,
	width, height int,
	theme *styles.Theme,
) scopePopupModel {
	pwd := startPwd
	if pwd == "" {
		pwd = gitData.Root
	}
	l, err := NewFileList(pwd, useNerdFonts, gitData)
	if err != nil {
		// NewFileList only fails on os.ReadDir errors; degrade gracefully
		// to an empty list rather than crash the popup.
		l = list.New(nil, FileDelegate{UseNerdFonts: useNerdFonts}, 0, 0)
	}
	// Default to modified-only: the scope is almost always one of the
	// files touched by the pending commit, so showing the full tree first
	// is just noise. Ctrl+R still toggles back to the full listing.
	_ = ChooseUpdateFileListFunction(true)(pwd, &l, gitData)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	// The popup intercepts enter / esc / arrow keys directly, so disable
	// the list's built-in accept/cancel bindings — otherwise typing "/"
	// (a valid filename character) would be swallowed as accept.
	l.KeyMap.AcceptWhileFiltering = key.NewBinding()
	l.KeyMap.CancelWhileFiltering = key.NewBinding()
	// Filter is always-on inside the popup: every printable key types
	// into FilterInput. SetFilterText empties the input and refreshes
	// items; SetFilterState then forces "Filtering" (instead of the
	// FilterApplied that SetFilterText leaves behind) so handleFiltering
	// routes keystrokes into FilterInput.
	l.SetFilterText("")
	l.SetFilterState(list.Filtering)
	return scopePopupModel{
		list:         l,
		pwd:          pwd,
		gitData:      gitData,
		showOnlyMod:  true,
		useNerdFonts: useNerdFonts,
		width:        width,
		height:       height,
		theme:        theme,
	}
}

func (m scopePopupModel) Init() tea.Cmd { return nil }

func (m scopePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(FileItem); ok {
				return m, func() tea.Msg { return setScopeMsg{scope: item.Title()} }
			}
			return m, nil
		case "esc":
			// First esc clears the filter text; a second esc (with the
			// filter already empty) closes the popup.
			if m.list.FilterInput.Value() != "" {
				m.list.SetFilterText("")
				m.list.SetFilterState(list.Filtering)
				return m, nil
			}
			return m, func() tea.Msg { return closeScopePopupMsg{} }
		case "ctrl+r":
			m.showOnlyMod = !m.showOnlyMod
			m.refreshList()
			return m, nil
		case "left":
			if m.canGoUp() {
				m.pwd = filepath.Dir(m.pwd)
				m.refreshList()
			}
			return m, nil
		case "right":
			if item, ok := m.list.SelectedItem().(FileItem); ok && item.IsDir() {
				m.pwd = filepath.Join(m.pwd, item.Title())
				m.refreshList()
			}
			return m, nil
		case "up":
			// While in Filtering state bubbles/list routes everything
			// to FilterInput, so up/down would not move through items.
			// Drive the cursor manually so navigation stays usable.
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

func (m scopePopupModel) View() tea.View {
	box := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	innerW := max(20, m.width-box.GetHorizontalFrameSize())
	innerH := max(6, m.height-box.GetVerticalFrameSize())

	base := m.theme.AppStyles().Base
	title := base.Foreground(m.theme.Secondary).Bold(true).Render("Scope")
	pathLabel := base.
		Foreground(m.theme.Muted).
		Render(fmt.Sprintf("· %s", TruncatePath(m.pwd, 3)))
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, " ", pathLabel)

	hint := base.Foreground(m.theme.Muted).Render(
		"type to filter · ↑↓ nav · ←/→ dirs · ctrl+r modified-only · enter pick · esc clear/close",
	)

	listH := max(3, innerH-lipgloss.Height(header)-lipgloss.Height(hint)-2)
	m.list.SetWidth(innerW)
	m.list.SetHeight(listH)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		m.list.View(),
		"",
		hint,
	)
	return tea.NewView(box.Render(body))
}

// canGoUp returns true when moving to the parent directory would keep us
// inside the git repository. Mirrors the stateChoosingScope guard.
func (m scopePopupModel) canGoUp() bool {
	if m.gitData.Root == "" {
		return false
	}
	parent := filepath.Clean(filepath.Dir(m.pwd))
	root := filepath.Clean(m.gitData.Root)
	if parent == root {
		return true
	}
	return strings.HasPrefix(parent, root+string(filepath.Separator))
}

// refreshList rebuilds the list contents for the current pwd, honouring
// the showOnlyMod toggle. Resets the filter text so a search typed in
// the previous directory does not silently hide entries here, and
// re-enters Filtering state so typing keeps feeding the filter input.
func (m *scopePopupModel) refreshList() {
	updater := ChooseUpdateFileListFunction(m.showOnlyMod)
	if err := updater(m.pwd, &m.list, m.gitData); err == nil {
		m.list.SetFilterText("")
		m.list.SetFilterState(list.Filtering)
		m.list.Select(0)
	}
}

// keyScopePopup opens the scope file picker from inside the
// writing-message state.
var keyScopePopup = key.NewBinding(
	key.WithKeys("ctrl+p"),
	key.WithHelp("ctrl+p", "Edit scope"),
)
