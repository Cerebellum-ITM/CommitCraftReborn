package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// closeKeybindingsPopupMsg dismisses the keybindings popup.
type closeKeybindingsPopupMsg struct{}

// keybindingGroup is a labelled cluster of (key, description) rows.
type keybindingGroup struct {
	title   string
	entries []helpEntry
}

type keybindingsPopupModel struct {
	width, height int
	theme         *styles.Theme
	groups        []keybindingGroup
}

func newKeybindingsPopup(
	width, height int,
	theme *styles.Theme,
	groups []keybindingGroup,
) keybindingsPopupModel {
	return keybindingsPopupModel{
		width:  width,
		height: height,
		theme:  theme,
		groups: groups,
	}
}

func (m keybindingsPopupModel) Init() tea.Cmd { return nil }

func (m keybindingsPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc", "?", "q":
			return m, func() tea.Msg { return closeKeybindingsPopupMsg{} }
		}
	}
	return m, nil
}

func (m keybindingsPopupModel) View() tea.View {
	base := m.theme.AppStyles().Base
	titleStyle := base.Foreground(m.theme.Secondary).Bold(true)
	groupStyle := base.Foreground(m.theme.Primary).Bold(true)
	keyStyle := base.Foreground(m.theme.Accent)
	descStyle := base.Foreground(m.theme.FgMuted)

	keyColW := 0
	for _, g := range m.groups {
		for _, e := range g.entries {
			if w := lipgloss.Width(e.key); w > keyColW {
				keyColW = w
			}
		}
	}
	keyColW += 2

	var sections []string
	for i, g := range m.groups {
		var lines []string
		lines = append(lines, groupStyle.Render(g.title))
		for _, e := range g.entries {
			pad := keyColW - lipgloss.Width(e.key)
			if pad < 1 {
				pad = 1
			}
			row := keyStyle.Render(e.key) +
				strings.Repeat(" ", pad) +
				descStyle.Render(e.desc)
			lines = append(lines, row)
		}
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, lines...))
		if i < len(m.groups)-1 {
			sections = append(sections, "")
		}
	}

	helpStyles := m.theme.AppStyles().Help
	hintPairs := [][2]string{
		{"?", "close"},
		{"esc", "close"},
		{"q", "close"},
	}
	hintParts := make([]string, 0, len(hintPairs)*2-1)
	for i, p := range hintPairs {
		if i > 0 {
			hintParts = append(hintParts, helpStyles.ShortSeparator.Render(" · "))
		}
		hintParts = append(hintParts,
			helpStyles.ShortKey.Render(p[0])+" "+helpStyles.ShortDesc.Render(p[1]),
		)
	}
	hint := strings.Join(hintParts, "")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Keybindings"),
		"",
		lipgloss.JoinVertical(lipgloss.Left, sections...),
		"",
		hint,
	)

	boxStyle := lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	return tea.NewView(boxStyle.Render(body))
}

// keybindingsForState returns the popup contents for states that surface the
// `?` shortcut. The second return value is false for states that should keep
// the bubbles help.ShowAll fallback (compose / release views).
func keybindingsForState(s appState) ([]keybindingGroup, bool) {
	switch s {
	case stateChoosingCommit:
		return workspaceKeybindings(), true
	case stateReleaseMainMenu:
		return releaseKeybindings(), true
	case stateReleaseChoosingCommits:
		return releaseChooseCommitsKeybindings(), true
	}
	return nil, false
}

// releaseChooseCommitsKeybindings is the popup for the workspace commit
// picker (Compose tab in Release Mode). Mirrors the workspace history
// popup but with the picker-specific keys (ctrl+a select, ctrl+e
// context-aware swap).
func releaseChooseCommitsKeybindings() []keybindingGroup {
	return []keybindingGroup{
		{
			title: "Navigate",
			entries: []helpEntry{
				{"↑↓", "move cursor"},
				{"tab / shift+tab", "cycle focus (filter → commits → message → files → diff)"},
				{"/", "filter"},
				{"esc", "back to release menu"},
			},
		},
		{
			title: "Commit picker",
			entries: []helpEntry{
				{"^a", "add/remove the highlighted commit from the release"},
				{"^f", "cycle filter mode (TITLE/TYPE/VERSION/BRANCH)"},
				{"^e (commits)", "swap All commits ⇄ Selected only"},
				{"^e (files panel)", "toggle filename ⇄ full relative path"},
				{"↵", "generate the release text from the selected commits"},
			},
		},
		{
			title: "Inspect",
			entries: []helpEntry{
				{"pgup/pgdn", "scroll the focused viewport"},
			},
		},
		{
			title: "App",
			entries: []helpEntry{
				{"^1 / ^2 / ^3", "switch tab (history / compose / pipeline)"},
				{"^k", "command palette"},
				{"^l", "logs"},
				{"^x", "quit"},
			},
		},
	}
}

// releaseKeybindings lists the shortcuts available on the Release history
// tab. Mirrors workspaceKeybindings but with release-flavoured labels —
// filter modes are TITLE/TYPE/VERSION/BRANCH and the dual panel cycles
// commits/stages instead of key points/stages.
func releaseKeybindings() []keybindingGroup {
	return []keybindingGroup{
		{
			title: "Navigate",
			entries: []helpEntry{
				{"↑↓", "move cursor"},
				{"↵", "open release"},
				{"/", "filter"},
				{"^f", "cycle filter mode (TITLE/TYPE/VERSION/BRANCH)"},
			},
		},
		{
			title: "Inspect panel",
			entries: []helpEntry{
				{"^e", "swap inspect mode (Commits/Body ↔ Stages/Response)"},
				{"^]", "next commit / stage"},
				{"^[", "prev commit / stage"},
				{"R", "jump to release entry"},
				{"pgup/pgdn · ^↑/^↓", "scroll right viewport"},
			},
		},
		{
			title: "Releases",
			entries: []helpEntry{
				{"r / tab", "create a release"},
				{"d / x", "delete"},
			},
		},
		{
			title: "App",
			entries: []helpEntry{
				{"^k", "command palette"},
				{"^s", "switch app mode"},
				{"^l", "logs"},
				{"^x", "quit"},
			},
		},
	}
}

// workspaceKeybindings lists the shortcuts available on the History tab,
// grouped so the popup reads as a quick reference instead of a flat list.
func workspaceKeybindings() []keybindingGroup {
	return []keybindingGroup{
		{
			title: "Navigate",
			entries: []helpEntry{
				{"↑↓", "move cursor"},
				{"↵", "open commit"},
				{"/", "filter"},
				{"^f", "cycle filter mode"},
			},
		},
		{
			title: "Inspect panel",
			entries: []helpEntry{
				{"^e", "swap inspect mode (KP/Body ↔ Stages/Response)"},
				{"^]", "next stage / key point"},
				{"^[", "prev stage / key point"},
				{"pgup/pgdn · ^↑/^↓", "scroll right viewport"},
			},
		},
		{
			title: "Commits",
			entries: []helpEntry{
				{"n / tab", "new commit"},
				{"e", "edit commit"},
				{"r", "create release"},
				{"d / x", "delete"},
				{"^d", "toggle drafts view"},
			},
		},
		{
			title: "App",
			entries: []helpEntry{
				{"^s", "switch app mode"},
				{"^c", "create local config template"},
				{"^l", "logs"},
				{"^x", "quit"},
			},
		},
	}
}
