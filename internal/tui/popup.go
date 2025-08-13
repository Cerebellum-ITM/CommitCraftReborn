package tui

import (
	"github.com/charmbracelet/bubbles/v2/key"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// PopupModel define el estado de nuestro popup. (Antes se llamaba Model)
type PopupModel struct {
	width, height int
	keys          KeyMap
}

// NewPopup crea un nuevo modelo de popup. (Antes se llamaba New)
func NewPopup(width, height int) PopupModel {
	return PopupModel{
		width:  width,
		height: height,
		keys:   listKeys(),
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
		case key.Matches(msg, m.keys.Esc), key.Matches(msg, m.keys.Logs):
			return m, func() tea.Msg { return closePopupMsg{} }
		}
	}
	return m, nil
}

func (m PopupModel) View() string {
	popupText := "Esta es mi ventana flotante, estilo CRUSH.\n\nPresiona 'esc' o 'ctrl+l' para cerrarla."

	popupBox := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Render(popupText)

	return popupBox
}
