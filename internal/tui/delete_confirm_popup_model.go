package tui

import (
	"image/color"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
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
			return m, programQuitCmd()
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

func (m DeleteConfirmPopupModel) View() tea.View {
	contentWidth := (m.width / 2) - 4
	contentWidth = max(contentWidth, 28)

	// Pick the noun + glyph from the table being targeted so the popup
	// reads correctly whether the user is deleting a commit or a
	// release. Falls back to "record" when the table is unset.
	noun := "record"
	glyph := ""
	switch m.db {
	case commitDb:
		noun = "commit"
		glyph = m.theme.AppSymbols().GitCommit
	case releaseDb:
		noun = "release"
		glyph = m.theme.AppSymbols().Tag
	}

	// Header pill — warning-tinted DELETE chip + "<glyph> <noun> #<id>"
	// title. Mirrors the source-pill / statusbar pill aesthetic so the
	// popup feels native to the rest of the UI rather than a stock
	// confirmation box.
	chip := lipgloss.NewStyle().
		Background(m.theme.Warning).
		Foreground(m.theme.Surface).
		Bold(true).
		Padding(0, 1).
		Render("DELETE")
	title := lipgloss.NewStyle().
		Foreground(m.theme.Warning).
		Bold(true).
		Render(strings.TrimSpace(glyph+" ") + noun + " #" + strconv.Itoa(m.Id))
	header := lipgloss.JoinHorizontal(lipgloss.Top, chip, "  ", title)

	// Message preview — italic + muted, truncated to the popup width so
	// long key-points don't blow the box. The leading guillemets frame
	// the preview as a quote rather than a continuation of the prompt.
	preview := strings.TrimSpace(m.Message)
	if preview == "" {
		preview = "(no preview available)"
	}
	previewBudget := contentWidth - 4
	if previewBudget < 8 {
		previewBudget = 8
	}
	preview = TruncateString(preview, previewBudget)
	previewView := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Italic(true).
		Render("« " + preview + " »")

	question := lipgloss.NewStyle().
		Foreground(m.theme.AppStyles().Base.GetForeground()).
		Render("This action cannot be undone.")

	helpView := m.help.View(m.keys)

	renderedContent := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		previewView,
		"",
		question,
		"",
		helpView,
	)

	popupBox := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Warning).
		Render(renderedContent)

	return tea.NewView(popupBox)
}
