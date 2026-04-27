package tui

import (
	"fmt"
	"io"
	"sort"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// openConfigPopupKey is the global shortcut that pops the configuration
// popup. Defined alongside the popup so the binding lives with the feature.
var openConfigPopupKey = key.NewBinding(
	key.WithKeys("ctrl+,"),
	key.WithHelp("ctrl+,", "Open config"),
)

// themePreviewMsg is fired while the user moves through the theme list so
// the main model can swap the active theme in place without persisting it.
type themePreviewMsg struct{ name string }

// themeAppliedMsg is fired when the user confirms a theme with Enter. The
// main model persists it and closes the popup.
type themeAppliedMsg struct{ name string }

// closeConfigPopupMsg dismisses the config popup, restoring the original
// theme captured when the popup was opened.
type closeConfigPopupMsg struct{}

type themeItem struct{ name string }

func (t themeItem) Title() string       { return t.name }
func (t themeItem) Description() string { return "" }
func (t themeItem) FilterValue() string { return t.name }

type themeDelegate struct {
	theme *styles.Theme
}

func (d themeDelegate) Height() int                             { return 1 }
func (d themeDelegate) Spacing() int                            { return 0 }
func (d themeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d themeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(themeItem)
	if !ok {
		return
	}
	base := d.theme.AppStyles().Base
	if index == m.Index() {
		indicator := base.Foreground(d.theme.Primary).Bold(true).Render("❯")
		text := base.Foreground(d.theme.FgBase).Render(it.name)
		fmt.Fprintf(w, "%s %s", indicator, text)
		return
	}
	fmt.Fprintf(w, "  %s", base.Foreground(d.theme.FgMuted).Render(it.name))
}

type configPopupModel struct {
	list          list.Model
	original      string
	currentIndex  int
	width, height int
	theme         *styles.Theme
}

func newConfigPopup(width, height int, theme *styles.Theme, currentTheme string) configPopupModel {
	names := styles.AvailableThemes()
	sort.Strings(names)

	items := make([]list.Item, len(names))
	selected := 0
	for i, n := range names {
		items[i] = themeItem{name: n}
		if n == currentTheme {
			selected = i
		}
	}

	l := list.New(items, themeDelegate{theme: theme}, width, height)
	l.Title = "Theme"
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.Select(selected)

	return configPopupModel{
		list:         l,
		original:     currentTheme,
		currentIndex: selected,
		width:        width,
		height:       height,
		theme:        theme,
	}
}

func (m configPopupModel) Init() tea.Cmd { return nil }

func (m configPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			return m, func() tea.Msg { return closeConfigPopupMsg{} }
		case "enter":
			if it, ok := m.list.SelectedItem().(themeItem); ok {
				name := it.name
				return m, func() tea.Msg { return themeAppliedMsg{name: name} }
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	if m.list.Index() != m.currentIndex {
		m.currentIndex = m.list.Index()
		if it, ok := m.list.SelectedItem().(themeItem); ok {
			name := it.name
			previewCmd := func() tea.Msg { return themePreviewMsg{name: name} }
			if cmd != nil {
				return m, tea.Batch(cmd, previewCmd)
			}
			return m, previewCmd
		}
	}
	return m, cmd
}

func (m configPopupModel) View() tea.View {
	boxStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	innerWidth := max(20, m.width-boxStyle.GetHorizontalFrameSize())
	innerHeight := max(4, m.height-boxStyle.GetVerticalFrameSize()-3)

	header := m.theme.AppStyles().Base.
		Foreground(m.theme.Secondary).
		Bold(true).
		Render("Configuration")

	section := m.theme.AppStyles().Base.
		Foreground(m.theme.FgBase).
		Render("Theme")

	hint := m.theme.AppStyles().Base.
		Foreground(m.theme.FgMuted).
		Render("↑↓ navigate · enter save · esc cancel")

	m.list.SetWidth(innerWidth)
	m.list.SetHeight(innerHeight)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		section,
		m.list.View(),
		"",
		hint,
	)
	return tea.NewView(boxStyle.Render(body))
}
