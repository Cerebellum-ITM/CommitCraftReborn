package tui

import (
	"fmt"

	"commit_craft_reborn/internal/tui/styles"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// versionUpdatedMsg is emitted after the local .commitcraft.toml has been
// patched with the new release version. The TUI uses it to display a status
// bar message and close the popup.
type versionUpdatedMsg struct {
	version string
	err     error
}

// closeVersionPopupMsg dismisses the popup without saving.
type closeVersionPopupMsg struct{}

type versionPopupModel struct {
	input         textinput.Model
	width, height int
	theme         *styles.Theme
	currentValue  string
	lastTag       string
}

func newVersionPopup(
	width, height int,
	currentValue, lastTag string,
	theme *styles.Theme,
) versionPopupModel {
	ti := textinput.New()
	ti.Prompt = "  "
	ti.Placeholder = "v0.0.0"

	// Pick the best initial proposal: bump the last tag's patch component, or
	// fall back to the current configured value, or a sensible default.
	proposal := BumpVersionPatch(lastTag)
	if proposal == "" {
		proposal = currentValue
	}
	if proposal == "" {
		proposal = "v0.1.0"
	}
	ti.SetValue(proposal)
	ti.SetCursor(len(proposal))
	ti.Focus()

	return versionPopupModel{
		input:        ti,
		width:        width,
		height:       height,
		theme:        theme,
		currentValue: currentValue,
		lastTag:      lastTag,
	}
}

func (m versionPopupModel) Init() tea.Cmd { return nil }

func (m versionPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			return m, func() tea.Msg { return closeVersionPopupMsg{} }
		case "enter":
			value := m.input.Value()
			return m, func() tea.Msg {
				err := UpdateLocalConfigVersion(value)
				return versionUpdatedMsg{version: value, err: err}
			}
		case "ctrl+a":
			m.input.SetValue(bumpDigitAtCursor(m.input.Value(), m.input.Position(), +1))
			return m, nil
		case "ctrl+x":
			m.input.SetValue(bumpDigitAtCursor(m.input.Value(), m.input.Position(), -1))
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m versionPopupModel) View() tea.View {
	base := m.theme.AppStyles().Base

	title := base.Foreground(m.theme.Secondary).Bold(true).Render("Set release version")

	tagInfo := "no tags found in repo"
	if m.lastTag != "" {
		tagInfo = fmt.Sprintf("Last tag: %s", m.lastTag)
	}
	currentInfo := "no value yet"
	if m.currentValue != "" {
		currentInfo = fmt.Sprintf("Current: %s", m.currentValue)
	}
	muted := base.Foreground(m.theme.FgMuted)
	info := lipgloss.JoinVertical(
		lipgloss.Left,
		muted.Render(tagInfo),
		muted.Render(currentInfo),
	)

	hint := muted.Render("ctrl+a inc · ctrl+x dec · enter save · esc cancel")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		info,
		"",
		base.Foreground(m.theme.FgBase).Render("New version:"),
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

// bumpDigitAtCursor finds the digit run at (or immediately to the left of)
// the cursor, parses it as an integer, applies delta, and writes it back so
// vim-style ctrl+a / ctrl+x can walk a version string component-by-component.
// If no digit is found at the cursor location, the value is returned
// unchanged.
func bumpDigitAtCursor(value string, cursor int, delta int) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	if cursor > len(runes) {
		cursor = len(runes)
	}

	// Find a digit at the cursor; if cursor sits on a non-digit, scan
	// forward to the next digit run on the same line. This matches vim's
	// behaviour: in "v1.2.3" with cursor on "v", ctrl+a bumps the "1".
	pos := cursor
	if pos == len(runes) || !isDigit(runes[pos]) {
		for pos < len(runes) && !isDigit(runes[pos]) {
			pos++
		}
	}
	if pos >= len(runes) {
		return value
	}

	start := pos
	for start > 0 && isDigit(runes[start-1]) {
		start--
	}
	end := pos
	for end < len(runes) && isDigit(runes[end]) {
		end++
	}

	var num int
	for i := start; i < end; i++ {
		num = num*10 + int(runes[i]-'0')
	}
	num += delta
	if num < 0 {
		num = 0
	}

	out := append([]rune{}, runes[:start]...)
	out = append(out, []rune(fmt.Sprintf("%d", num))...)
	out = append(out, runes[end:]...)
	return string(out)
}

func isDigit(r rune) bool { return r >= '0' && r <= '9' }

// versionPopupKey is the global keybinding that opens the release-version
// editor. Defined here so the binding lives next to the popup it triggers.
var versionPopupKey = key.NewBinding(
	key.WithKeys("ctrl+v"),
	key.WithHelp("ctrl+v", "Set release version"),
)
