package tui

import (
	"fmt"
	"image/color"
	"strconv"

	"commit_craft_reborn/internal/tui/styles"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

var DefaultPopupColor color.Color = lipgloss.Color("63")

type PopupOption func(*DeleteConfirmPopupModel)

type DeleteConfirmPopupModel struct {
	Id, width, height int
	Message           string
	keys              KeyMap
	db                CommitCraftTables
	color             color.Color
	help              help.Model
	theme             *styles.Theme
}

func WithColor(c color.Color) PopupOption {
	return func(p *DeleteConfirmPopupModel) {
		p.color = c
	}
}

func WithTheme(t *styles.Theme) PopupOption {
	return func(p *DeleteConfirmPopupModel) {
		p.theme = t
	}
}

func NewPopup(
	width, height, Id int,
	Message string,
	db CommitCraftTables,
	opts ...PopupOption,
) DeleteConfirmPopupModel {
	popup := DeleteConfirmPopupModel{
		Id:      Id,
		Message: Message,
		width:   width,
		height:  height,
		keys:    popupKeys(),
		db:      db,
		color:   DefaultPopupColor,
		help:    help.New(),
	}

	for _, opt := range opts {
		opt(&popup)
	}

	popup.help.Styles = popup.theme.AppStyles().Help
	return popup
}

func (m DeleteConfirmPopupModel) Init() tea.Cmd {
	return nil
}

func (m DeleteConfirmPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit, m.keys.GlobalQuit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Esc):
			return m, func() tea.Msg { return closePopupMsg{} }
		case key.Matches(msg, m.keys.Enter):
			return m, func() tea.Msg {
				return deleteItemMsg{ID: m.Id, Db: m.db}
			}
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, func() tea.Msg {
				return tea.WindowSizeMsg{Width: m.width, Height: m.height}
			}

		}
	}
	return m, nil
}

func (m DeleteConfirmPopupModel) View() string {
	contentWidth := (m.width / 2) - 4
	contentWidth = max(contentWidth, 20)
	body := m.theme.AppStyles().Base.Render("Are you sure you want to delete the Item with the Id=")
	message := m.theme.AppStyles().IndicatorStyle.Render(m.Message)
	id := m.theme.AppStyles().IndicatorStyle.Render(strconv.Itoa(m.Id))
	popupMessage := fmt.Sprintf(
		"%s%s?\n\t(%s).",
		body,
		id,
		message,
	)
	popupContent := TruncateMessageLines(popupMessage, contentWidth)
	helpView := m.help.View(m.keys)
	renderedContent := lipgloss.JoinVertical(lipgloss.Top,
		popupContent,
		VerticalSpace,
		helpView,
	)

	popupBox := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.color).
		Render(renderedContent)

	return popupBox
}
