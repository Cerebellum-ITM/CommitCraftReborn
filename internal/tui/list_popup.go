package tui

import (
	"fmt"
	"image/color"
	"io"

	"commit_craft_reborn/internal/tui/styles"

	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

var defaultListPopTitle string = "Select an action"

type item struct {
	title, desc string
}

type listPopupModel struct {
	title         string
	width, height int
	selector      list.Model
	keys          KeyMap
	theme         *styles.Theme
	color         color.Color
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type itemDelegate struct {
	list.DefaultDelegate
	Theme          *styles.Theme
	indicatorStyle lipgloss.Style
	textStyle      lipgloss.Style
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func NewListPopupDelegate(theme *styles.Theme) list.ItemDelegate {
	base := theme.AppStyles().Base
	baseFg := theme.FgMuted
	baseState := base.Foreground(baseFg)

	return itemDelegate{
		Theme:          theme,
		textStyle:      baseState,
		indicatorStyle: theme.AppStyles().IndicatorStyle,
	}
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(item)
	if !ok {
		return
	}

	var indicator string
	var textStyle lipgloss.Style

	if index == m.Index() {
		indicator = d.indicatorStyle.Render("❯")
		textStyle = d.textStyle.Foreground(d.Theme.FgBase)
	} else {
		indicator = " "
		textStyle = d.textStyle
	}

	itemString := textStyle.Render(item.title)

	line := fmt.Sprintf(
		"%s %s",
		indicator,
		itemString,
	)
	fmt.Fprint(w, line)
}

func (m listPopupModel) Init() tea.Cmd {
	return nil
}

type PopupListOption func(*listPopupModel)

func ListWithColor(c color.Color) PopupListOption {
	return func(p *listPopupModel) {
		p.color = c
	}
}

func ListWithTitle(t string) PopupListOption {
	return func(p *listPopupModel) {
		p.selector.Title = t
	}
}

func NewListPopup(
	items []string,
	width, height int,
	keys KeyMap,
	theme *styles.Theme,
	opts ...PopupListOption,
) listPopupModel {
	listItems := make([]list.Item, len(items))
	for i, element := range items {
		listItems[i] = item{title: element}
	}

	list := list.New(listItems, NewListPopupDelegate(theme), width, height)
	list.SetHeight(height)
	list.SetWidth(width)
	list.SetShowPagination(false)
	list.Title = defaultListPopTitle
	list.SetShowStatusBar(false)
	list.Help.Styles = theme.AppStyles().Help

	popList := listPopupModel{
		width:    width,
		height:   height,
		selector: list,
		theme:    theme,
		keys:     keys,
		color:    theme.Primary,
	}

	for _, opt := range opts {
		opt(&popList)
	}

	return popList
}

func (m listPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector.SetHeight(m.height)
		m.selector.SetWidth(m.width)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Esc):
			return m, func() tea.Msg { return closeListPopup{} }
		case key.Matches(msg, m.keys.Enter):
			selectedItem, ok := m.selector.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return releaseAction{action: selectedItem.title} }
		}
	}
	m.selector, cmd = m.selector.Update(msg)
	return m, cmd
}

func (m listPopupModel) View() string {
	contentWidth := (m.width / 2) - 4
	contentWidth = max(contentWidth, 20)

	popupBox := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.color).
		Render(m.selector.View())

	return popupBox
}
