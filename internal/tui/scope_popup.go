package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/tui/styles"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	return scopePopupModel{
		list:         l,
		pwd:          pwd,
		gitData:      gitData,
		useNerdFonts: useNerdFonts,
		width:        width,
		height:       height,
		theme:        theme,
	}
}

func (m scopePopupModel) Init() tea.Cmd { return nil }

func (m scopePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		// While typing in the filter the navigation arrows belong to the
		// list (item filtering); only the textual hand-off (enter / esc)
		// is intercepted by the popup itself.
		filtering := m.list.FilterState() == list.Filtering

		if !filtering {
			switch km.String() {
			case "esc":
				return m, func() tea.Msg { return closeScopePopupMsg{} }
			case "ctrl+r":
				m.showOnlyMod = !m.showOnlyMod
				m.refreshList()
				return m, nil
			case "left", "h":
				if m.canGoUp() {
					m.pwd = filepath.Dir(m.pwd)
					m.refreshList()
				}
				return m, nil
			case "right", "l":
				if item, ok := m.list.SelectedItem().(FileItem); ok && item.IsDir() {
					m.pwd = filepath.Join(m.pwd, item.Title())
					m.refreshList()
				}
				return m, nil
			}
		}

		if km.String() == "enter" && !filtering {
			if item, ok := m.list.SelectedItem().(FileItem); ok {
				return m, func() tea.Msg { return setScopeMsg{scope: item.Title()} }
			}
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
		"↑↓ nav · → enter dir · ← parent · ctrl+r modified-only · enter pick · esc cancel",
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
// the showOnlyMod toggle.
func (m *scopePopupModel) refreshList() {
	updater := ChooseUpdateFileListFunction(m.showOnlyMod)
	if err := updater(m.pwd, &m.list, m.gitData); err == nil {
		m.list.Select(0)
	}
}

// keyScopePopup opens the scope file picker from inside the
// writing-message state.
var keyScopePopup = key.NewBinding(
	key.WithKeys("ctrl+p"),
	key.WithHelp("ctrl+p", "Edit scope"),
)
