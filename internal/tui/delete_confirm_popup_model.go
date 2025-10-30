package tui

import (
	"fmt"
	"image/color"

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
}

func WithColor(c color.Color) PopupOption {
	return func(p *DeleteConfirmPopupModel) {
		p.color = c
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
		keys:    listKeys(),
		db:      db,
		color:   DefaultPopupColor,
	}

	for _, opt := range opts {
		opt(&popup)
	}

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
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Esc):
			return m, func() tea.Msg { return closePopupMsg{} }
		case key.Matches(msg, m.keys.Enter):
			return m, func() tea.Msg {
				return deleteItemMsg{ID: m.Id, Db: m.db}
			}
		}
	}
	return m, nil
}

func (m DeleteConfirmPopupModel) View() string {
	popupMessage := fmt.Sprintf(
		"Are you sure you want to delete the Item with the Id=%d?\n(%s)\nPress 'esc' to cancel or 'enter' to delete.",
		m.Id,
		m.Message,
	)
	contentWidth := (m.width / 2) - 4
	contentWidth = max(contentWidth, 20)
	renderedContent := TruncateMessageLines(popupMessage, contentWidth)

	popupBox := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.color).
		Render(renderedContent)

	return popupBox
}
