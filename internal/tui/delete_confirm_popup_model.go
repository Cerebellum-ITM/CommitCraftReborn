package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type DeleteConfirmPopupModel struct {
	Id, width, height int
	Message           string
	keys              KeyMap
	db                CommitCraftTables
}

func NewPopup(width, height, Id int, Message string, db CommitCraftTables) DeleteConfirmPopupModel {
	return DeleteConfirmPopupModel{
		Id:      Id,
		Message: Message,
		width:   width,
		height:  height,
		keys:    listKeys(),
		db:      db,
	}
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
		"Are you sure you want to delete the commit with the Id=%d?\n(%s)\nPress 'esc' to cancel or 'enter' to delete.",
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
		BorderForeground(lipgloss.Color("63")).
		Render(renderedContent)

	return popupBox
}
