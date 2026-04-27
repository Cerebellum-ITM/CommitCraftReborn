package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// editMessageAppliedMsg carries the edited commit message back to the
// main update loop. Triggered by ctrl+s inside the popup.
type editMessageAppliedMsg struct{ value string }

// closeEditMessagePopupMsg dismisses the popup without applying changes.
type closeEditMessagePopupMsg struct{}

type editMessagePopupModel struct {
	input         textarea.Model
	width, height int
	theme         *styles.Theme
}

func newEditMessagePopup(
	width, height int,
	initial string,
	theme *styles.Theme,
) editMessagePopupModel {
	ta := textarea.New()
	ta.SetStyles(theme.AppStyles().TextArea)
	ta.Prompt = "┃ "
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("enter"))
	ta.SetWidth(max(20, width-6))
	ta.SetHeight(max(6, height-8))
	ta.SetValue(initial)
	ta.MoveToEnd()
	ta.Focus()

	return editMessagePopupModel{
		input:  ta,
		width:  width,
		height: height,
		theme:  theme,
	}
}

func (m editMessagePopupModel) Init() tea.Cmd { return nil }

func (m editMessagePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			return m, func() tea.Msg { return closeEditMessagePopupMsg{} }
		case "ctrl+s":
			value := m.input.Value()
			return m, func() tea.Msg { return editMessageAppliedMsg{value: value} }
		case "ctrl+d":
			line := m.input.Line()
			lines := strings.Split(m.input.Value(), "\n")
			if line >= 0 && line < len(lines) {
				lines = append(lines[:line], lines[line+1:]...)
				m.input.SetValue(strings.Join(lines, "\n"))
				for m.input.Line() > line {
					m.input.CursorUp()
				}
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m editMessagePopupModel) View() tea.View {
	base := m.theme.AppStyles().Base
	title := base.Foreground(m.theme.Secondary).Bold(true).Render("Edit AI commit message")
	hint := base.Foreground(m.theme.FgMuted).
		Render("ctrl+s apply · ctrl+d delete line · enter newline · esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.input.View(),
		"",
		hint,
	)

	boxStyle := lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocus)

	return tea.NewView(boxStyle.Render(body))
}
