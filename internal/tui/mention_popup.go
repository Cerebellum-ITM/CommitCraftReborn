package tui

import (
	"fmt"
	"io"

	"commit_craft_reborn/internal/tui/styles"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type mentionFileItem struct {
	path string
}

func (m mentionFileItem) Title() string       { return m.path }
func (m mentionFileItem) Description() string { return "" }
func (m mentionFileItem) FilterValue() string { return m.path }

type mentionFileDelegate struct {
	list.DefaultDelegate
	theme          *styles.Theme
	indicatorStyle lipgloss.Style
	textStyle      lipgloss.Style
}

func (d mentionFileDelegate) Height() int                              { return 1 }
func (d mentionFileDelegate) Spacing() int                             { return 0 }
func (d mentionFileDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d mentionFileDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(mentionFileItem)
	if !ok {
		return
	}

	var indicator string
	var textStyle lipgloss.Style

	if index == m.Index() {
		indicator = d.indicatorStyle.Render("❯")
		textStyle = d.textStyle.Foreground(d.theme.FgBase)
	} else {
		indicator = " "
		textStyle = d.textStyle
	}

	fmt.Fprintf(w, "%s %s", indicator, textStyle.Render(it.path))
}

type mentionFilePopupModel struct {
	selector list.Model
	width    int
	height   int
	theme    *styles.Theme
	keys     KeyMap
}

type mentionFileSelectedMsg struct{ filename string }
type closeMentionPopupMsg struct{}

func newMentionFilePopup(files []string, width, height int, theme *styles.Theme) mentionFilePopupModel {
	listItems := make([]list.Item, len(files))
	for i, f := range files {
		listItems[i] = mentionFileItem{path: f}
	}

	base := theme.AppStyles().Base
	delegate := mentionFileDelegate{
		theme:          theme,
		textStyle:      base.Foreground(theme.FgMuted),
		indicatorStyle: theme.AppStyles().IndicatorStyle,
	}

	listHeight := min(len(files)+4, 12)
	l := list.New(listItems, delegate, width/2, listHeight)
	l.Title = "@ Reference a file"
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(true)
	l.Help.Styles = theme.AppStyles().Help

	return mentionFilePopupModel{
		selector: l,
		width:    width,
		height:   height,
		theme:    theme,
		keys:     listKeys(),
	}
}

func (m mentionFilePopupModel) Init() tea.Cmd { return nil }

func (m mentionFilePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Esc):
			return m, func() tea.Msg { return closeMentionPopupMsg{} }
		case key.Matches(msg, m.keys.Enter):
			selected, ok := m.selector.SelectedItem().(mentionFileItem)
			if !ok {
				return m, nil
			}
			filename := selected.path
			return m, func() tea.Msg { return mentionFileSelectedMsg{filename: filename} }
		}
	}
	m.selector, cmd = m.selector.Update(msg)
	return m, cmd
}

func (m mentionFilePopupModel) View() tea.View {
	contentWidth := (m.width / 2) - 4
	if contentWidth < 30 {
		contentWidth = 30
	}

	popupBox := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent).
		Render(m.selector.View())

	return tea.NewView(popupBox)
}
