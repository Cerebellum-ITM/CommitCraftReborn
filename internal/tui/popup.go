package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// PopupModel define el estado de nuestro popup. (Antes se llamaba Model)
type PopupModel struct {
	commitId, width, height int
	commitMessage           string
	keys                    KeyMap
}

// NewPopup crea un nuevo modelo de popup. (Antes se llamaba New)
func NewPopup(width, height, commitId int, commitMessage string) PopupModel {
	return PopupModel{
		commitId:      commitId,
		commitMessage: commitMessage,
		width:         width,
		height:        height,
		keys:          listKeys(),
	}
}

func (m PopupModel) Init() tea.Cmd {
	return nil
}

// Update maneja la lógica del popup.
func (m PopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Esc):
			return m, func() tea.Msg { return closePopupMsg{} }
		case key.Matches(msg, m.keys.Enter):
			return m, func() tea.Msg {
				return deleteItemMsg{ID: m.commitId}
			}
		}
	}
	return m, nil
}

func (m PopupModel) View() string {
	popupText := "Are you sure you want to delete the commit with the Id=%d (%s)?\n\nPress 'esc' to cancel or 'enter' to delete."

	popupBox := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Render(fmt.Sprintf(popupText, m.commitId, m.commitMessage))

	return popupBox
}
