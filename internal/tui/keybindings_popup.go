package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// bindKeys joins the rendered Help().Key of each enabled binding with the
// help-style separator. Used by the per-state popup builders so the key
// column reflects whatever is actually populated in the active KeyMap —
// a binding rename propagates here automatically.
func bindKeys(bs ...key.Binding) string {
	parts := make([]string, 0, len(bs))
	for _, b := range bs {
		if !b.Enabled() {
			continue
		}
		if k := b.Help().Key; k != "" {
			parts = append(parts, k)
		}
	}
	return strings.Join(parts, " · ")
}

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

// keybindingsForState returns the popup contents for states that surface
// the `?` shortcut. The active `KeyMap` (model.keys) is the single source
// of truth for displayed key strings; descriptions remain contextual.
// The second return value is false for states that should keep the
// bubbles `help.ShowAll` fallback (compose / release views).
func keybindingsForState(s appState, k KeyMap) ([]keybindingGroup, bool) {
	switch s {
	case stateChoosingCommit:
		return workspaceKeybindings(k), true
	case stateReleaseMainMenu:
		return releaseKeybindings(k), true
	case stateReleaseChoosingCommits:
		return releaseChooseCommitsKeybindings(k), true
	case stateReleaseBuildingText:
		return releaseBuildingTextKeybindings(k), true
	}
	return nil, false
}

// releaseBuildingTextKeybindings is the popup for the release pipeline
// view. Tab cycles between stage cards (and the final card once the
// refine stage finishes); Esc walks back to the picker without
// re-running the pipeline.
func releaseBuildingTextKeybindings(k KeyMap) []keybindingGroup {
	return []keybindingGroup{
		{
			title: "Navigate",
			entries: []helpEntry{
				{
					bindKeys(k.NextField, k.PrevField),
					"cycle stage cards (body → title → refine → final)",
				},
				{bindKeys(k.PgUp, k.PgDown), "scroll the focused stage's output"},
				{bindKeys(k.Esc), "back to commit picker (or cancel a running pipeline)"},
			},
		},
		{
			title: "Pipeline",
			entries: []helpEntry{
				{bindKeys(k.Enter), "open create-release menu (after stage 3 finishes)"},
				{bindKeys(k.Toggle), "retry the full pipeline (body → title → refine)"},
				{
					bindKeys(k.RerunStage1, k.RerunStage2, k.RerunStage3),
					"retry from stage 1 (body) / 2 (title) / 3 (refine); cascades downstream",
				},
				{bindKeys(k.History), "open the focused stage's history"},
			},
		},
		{
			title: "Global",
			entries: []helpEntry{
				{"^1 · ^2 · ^3", "switch tab (history / compose / pipeline)"},
				{bindKeys(k.GlobalQuit), "quit"},
				{bindKeys(k.Help), "this help"},
			},
		},
	}
}

// releaseChooseCommitsKeybindings is the popup for the workspace commit
// picker (Compose tab in Release Mode). Mirrors the workspace history
// popup but with the picker-specific keys (ctrl+a select, ctrl+e
// context-aware swap).
func releaseChooseCommitsKeybindings(k KeyMap) []keybindingGroup {
	return []keybindingGroup{
		{
			title: "Navigate",
			entries: []helpEntry{
				{bindKeys(k.Up, k.Down), "move cursor"},
				{
					bindKeys(k.NextField, k.PrevField),
					"cycle focus (filter → commits → message → files → diff)",
				},
				{bindKeys(k.Filter), "filter"},
				{bindKeys(k.Esc), "back to release menu"},
			},
		},
		{
			title: "Commit picker",
			entries: []helpEntry{
				{bindKeys(k.AddCommit), "add/remove the highlighted commit from the release"},
				{bindKeys(k.CycleFilterMode), "cycle filter mode (TITLE/TYPE/VERSION/BRANCH)"},
				{bindKeys(k.SwapMode) + " (commits)", "swap All commits ⇄ Selected only"},
				{bindKeys(k.SwapMode) + " (files panel)", "toggle filename ⇄ full relative path"},
				{bindKeys(k.Enter), "generate the release text from the selected commits"},
			},
		},
		{
			title: "App",
			entries: []helpEntry{
				{"^1 · ^2 · ^3", "switch tab (history / compose / pipeline)"},
				{"^k", "command palette"},
				{"^l", "logs"},
				{bindKeys(k.GlobalQuit), "quit"},
			},
		},
	}
}

// releaseKeybindings lists the shortcuts available on the Release history
// tab. Mirrors workspaceKeybindings but with release-flavoured labels —
// filter modes are TITLE/TYPE/VERSION/BRANCH and the dual panel cycles
// commits/stages instead of key points/stages.
func releaseKeybindings(k KeyMap) []keybindingGroup {
	return []keybindingGroup{
		{
			title: "Navigate",
			entries: []helpEntry{
				{bindKeys(k.Up, k.Down), "move cursor"},
				{bindKeys(k.Enter), "open release"},
				{bindKeys(k.Filter), "filter"},
				{bindKeys(k.CycleFilterMode), "cycle filter mode (TITLE/TYPE/VERSION/BRANCH)"},
			},
		},
		{
			title: "Inspect panel",
			entries: []helpEntry{
				{bindKeys(k.SwapMode), "swap inspect mode (Commits/Body ↔ Stages/Response)"},
				{bindKeys(k.CycleNext), "next commit / stage"},
				{bindKeys(k.CyclePrev), "prev commit / stage"},
				{bindKeys(k.EditIaCommit), "jump to release entry"},
				{bindKeys(k.PgUp, k.PgDown), "scroll right viewport"},
			},
		},
		{
			title: "Releases",
			entries: []helpEntry{
				{bindKeys(k.ReleaseCommit), "create a release"},
				{bindKeys(k.Delete), "delete"},
			},
		},
		{
			title: "App",
			entries: []helpEntry{
				{"^k", "command palette"},
				{bindKeys(k.SwitchMode), "switch app mode"},
				{"^l", "logs"},
				{bindKeys(k.GlobalQuit), "quit"},
			},
		},
	}
}

// workspaceKeybindings lists the shortcuts available on the History tab,
// grouped so the popup reads as a quick reference instead of a flat list.
func workspaceKeybindings(k KeyMap) []keybindingGroup {
	return []keybindingGroup{
		{
			title: "Navigate",
			entries: []helpEntry{
				{bindKeys(k.Up, k.Down), "move cursor"},
				{bindKeys(k.Enter), "open commit"},
				{bindKeys(k.Filter), "filter"},
				{bindKeys(k.CycleFilterMode), "cycle filter mode"},
			},
		},
		{
			title: "Inspect panel",
			entries: []helpEntry{
				{bindKeys(k.SwapMode), "swap inspect mode (KP/Body ↔ Stages/Response)"},
				{bindKeys(k.CycleNext), "next stage / key point"},
				{bindKeys(k.CyclePrev), "prev stage / key point"},
				{bindKeys(k.PgUp, k.PgDown), "scroll right viewport"},
			},
		},
		{
			title: "Commits",
			entries: []helpEntry{
				{bindKeys(k.AddCommit), "new commit"},
				{bindKeys(k.EditIaCommit), "edit commit"},
				{bindKeys(k.ReleaseCommit), "create release"},
				{bindKeys(k.Delete), "delete"},
				{bindKeys(k.ToggleDrafts), "toggle drafts view"},
			},
		},
		{
			title: "App",
			entries: []helpEntry{
				{bindKeys(k.SwitchMode), "switch app mode"},
				{bindKeys(k.CreateLocalTomlConfig), "create local config template"},
				{bindKeys(k.AddCommitTypes), "add commit tag types to local config"},
				{"^l", "logs"},
				{bindKeys(k.GlobalQuit), "quit"},
			},
		},
	}
}
