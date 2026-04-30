package tui

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// closeCommandPaletteMsg dismisses the palette without running anything.
type closeCommandPaletteMsg struct{}

// commandRunMsg is fired when the user picks an entry. The id is matched
// in update.go to dispatch the corresponding action.
type commandRunMsg struct{ id string }

// paletteCommand describes a single entry in the Ctrl+K palette. ID is
// stable across renders — update.go dispatches on it. Title is what the
// user reads; Description is the secondary line.
type paletteCommand struct {
	ID          string
	Title       string
	Description string
	Icon        string
}

const (
	cmdGenerateLocalConfig = "config.local.create"
	cmdShowTagPalette      = "tags.show"
)

// builtinCommands is the seed registry. Add entries here as new actions
// become available; the palette renders them in declaration order.
func builtinCommands(useNerdFonts bool) []paletteCommand {
	cfg := ""
	tags := ""
	if !useNerdFonts {
		cfg = "*"
		tags = "#"
	}
	return []paletteCommand{
		{
			ID:          cmdGenerateLocalConfig,
			Title:       "Generate local config file",
			Description: "Create .commitcraft.toml in the current directory if missing",
			Icon:        cfg,
		},
		{
			ID:          cmdShowTagPalette,
			Title:       "Show tag palette",
			Description: "Open the color reference for every commit-type tag",
			Icon:        tags,
		},
	}
}

type commandItem struct {
	cmd paletteCommand
}

func (c commandItem) Title() string       { return c.cmd.Title }
func (c commandItem) Description() string { return c.cmd.Description }
func (c commandItem) FilterValue() string {
	return c.cmd.Title + " " + c.cmd.Description
}

type commandPaletteDelegate struct {
	theme *styles.Theme
}

func (d commandPaletteDelegate) Height() int                             { return 2 }
func (d commandPaletteDelegate) Spacing() int                            { return 0 }
func (d commandPaletteDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d commandPaletteDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(commandItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	base := d.theme.AppStyles().Base

	var titleStyle, descStyle, iconStyle lipgloss.Style
	if selected {
		titleStyle = base.Foreground(d.theme.Primary).Bold(true)
		descStyle = base.Foreground(d.theme.FgBase)
		iconStyle = base.Foreground(d.theme.Secondary).Bold(true)
	} else {
		titleStyle = base.Foreground(d.theme.FgBase)
		descStyle = base.Foreground(d.theme.Muted)
		iconStyle = base.Foreground(d.theme.Muted)
	}

	cursor := "  "
	if selected && d.theme.Secondary != nil {
		cursor = base.Foreground(d.theme.Secondary).Bold(true).Render("❯ ")
	}

	icon := it.cmd.Icon
	if icon == "" {
		icon = " "
	}
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		cursor,
		iconStyle.Render(icon),
		" ",
		titleStyle.Render(it.cmd.Title),
	)
	descLine := "    " + descStyle.Render(it.cmd.Description)

	io.WriteString(w, header+"\n"+descLine)
}

type commandPalettePopupModel struct {
	list          list.Model
	width, height int
	theme         *styles.Theme
}

func newCommandPalettePopup(
	width, height int,
	theme *styles.Theme,
	useNerdFonts bool,
) commandPalettePopupModel {
	cmds := builtinCommands(useNerdFonts)
	items := make([]list.Item, len(cmds))
	for i, c := range cmds {
		items[i] = commandItem{cmd: c}
	}
	l := list.New(items, commandPaletteDelegate{theme: theme}, width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.KeyMap.AcceptWhileFiltering = key.NewBinding()
	l.KeyMap.CancelWhileFiltering = key.NewBinding()
	l.SetFilterText("")
	l.SetFilterState(list.Filtering)

	return commandPalettePopupModel{
		list:   l,
		width:  width,
		height: height,
		theme:  theme,
	}
}

func (m commandPalettePopupModel) Init() tea.Cmd { return nil }

func (m commandPalettePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			if m.list.FilterInput.Value() != "" {
				m.list.SetFilterText("")
				m.list.SetFilterState(list.Filtering)
				return m, nil
			}
			return m, func() tea.Msg { return closeCommandPaletteMsg{} }
		case "ctrl+k":
			return m, func() tea.Msg { return closeCommandPaletteMsg{} }
		case "enter":
			selected, ok := m.list.SelectedItem().(commandItem)
			if !ok {
				return m, nil
			}
			id := selected.cmd.ID
			return m, func() tea.Msg { return commandRunMsg{id: id} }
		case "up":
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

func (m commandPalettePopupModel) View() tea.View {
	boxStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	innerWidth := max(30, m.width-boxStyle.GetHorizontalFrameSize())
	innerHeight := max(8, m.height-boxStyle.GetVerticalFrameSize())

	base := m.theme.AppStyles().Base
	title := base.Foreground(m.theme.Secondary).Bold(true).Render("Command palette")

	help := m.theme.AppStyles().Help
	hint := strings.Join([]string{
		help.ShortKey.Render("type") + " " + help.ShortDesc.Render("filter"),
		help.ShortSeparator.Render(" · "),
		help.ShortKey.Render("↑↓") + " " + help.ShortDesc.Render("nav"),
		help.ShortSeparator.Render(" · "),
		help.ShortKey.Render("enter") + " " + help.ShortDesc.Render("run"),
		help.ShortSeparator.Render(" · "),
		help.ShortKey.Render("esc") + " " + help.ShortDesc.Render("close"),
	}, "")

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
