package tui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/tui/styles"
)

// closeTagPickerPopupMsg dismisses the tag picker popup without writing.
type closeTagPickerPopupMsg struct{}

// tagPickerSaveMsg fires when the user confirms the picker. The handler
// is responsible for writing the entries to the workspace TOML and
// reloading model.finalCommitTypes so the new tags become selectable.
type tagPickerSaveMsg struct {
	picked []commit.CommitType
}

// tagPickerPopupModel renders the multi-select popup that lets the user
// add commit-type tags (with their four-color palette) to the workspace
// `.commitcraft.toml`. Only tags with a known palette that are NOT
// already in `existing` are surfaced — once a tag has been added it
// stops appearing the next time the popup is opened.
type tagPickerPopupModel struct {
	width, height int
	theme         *styles.Theme
	rows          []tagPickerRow
	cursor        int
	body          viewport.Model
	emptyState    bool
}

type tagPickerRow struct {
	tag         string
	description string
	bgBlock     string
	fgBlock     string
	bgMsg       string
	fgMsg       string
	selected    bool
}

func newTagPickerPopup(
	width, height int,
	theme *styles.Theme,
	existing []commit.CommitType,
) tagPickerPopupModel {
	taken := make(map[string]struct{}, len(existing))
	for _, t := range existing {
		taken[strings.ToUpper(t.Tag)] = struct{}{}
	}

	addable := commit.GetAddableCommitTypes()
	rows := make([]tagPickerRow, 0, len(addable))
	for _, t := range addable {
		if _, dup := taken[strings.ToUpper(t.Tag)]; dup {
			continue
		}
		rows = append(rows, tagPickerRow{
			tag:         t.Tag,
			description: t.Description,
			bgBlock:     t.BgBlock,
			fgBlock:     t.FgBlock,
			bgMsg:       t.BgMsg,
			fgMsg:       t.FgMsg,
		})
	}

	vp := viewport.New()
	m := tagPickerPopupModel{
		width:      width,
		height:     height,
		theme:      theme,
		rows:       rows,
		body:       vp,
		emptyState: len(rows) == 0,
	}
	m.refreshBody()
	return m
}

func (m tagPickerPopupModel) Init() tea.Cmd { return nil }

func (m tagPickerPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc", "q":
			return m, func() tea.Msg { return closeTagPickerPopupMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.refreshBody()
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
				m.refreshBody()
			}
			return m, nil
		case " ", "space", "x":
			if len(m.rows) > 0 {
				m.rows[m.cursor].selected = !m.rows[m.cursor].selected
				m.refreshBody()
			}
			return m, nil
		case "a":
			// Toggle-all: select every row when at least one is unset,
			// otherwise clear the whole list. Avoids the user having to
			// space through each row when they want everything.
			anyUnset := false
			for _, r := range m.rows {
				if !r.selected {
					anyUnset = true
					break
				}
			}
			for i := range m.rows {
				m.rows[i].selected = anyUnset
			}
			m.refreshBody()
			return m, nil
		case "enter":
			picked := make([]commit.CommitType, 0, len(m.rows))
			for _, r := range m.rows {
				if r.selected {
					picked = append(picked, commit.CommitType{
						Tag:         r.tag,
						Description: r.description,
						BgBlock:     r.bgBlock,
						FgBlock:     r.fgBlock,
						BgMsg:       r.bgMsg,
						FgMsg:       r.fgMsg,
					})
				}
			}
			return m, func() tea.Msg { return tagPickerSaveMsg{picked: picked} }
		}
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(msg)
	return m, cmd
}

const (
	tagPickerCheckColW = 3 // "[x]"
	tagPickerChipColW  = styles.CommitTypeChipInnerWidth + 2
	tagPickerColGap    = "  "
)

func (m *tagPickerPopupModel) refreshBody() {
	boxFrame := 2*1 + 2*2 // border + padding(1, 2)
	inner := max(40, m.width-boxFrame)
	headerLines := 4 // title + blank + subtitle + blank
	hintLines := 2
	bodyH := max(4, m.height-2*1-2*1-headerLines-hintLines)

	m.body.SetWidth(inner)
	m.body.SetHeight(bodyH)
	m.body.SetContent(m.renderRows(inner))
	m.body.GotoTop()
}

func (m tagPickerPopupModel) renderRows(width int) string {
	if len(m.rows) == 0 {
		return m.theme.AppStyles().Base.
			Foreground(m.theme.Muted).
			Render("No more tags to add — every palette entry is already in your config.")
	}

	base := m.theme.AppStyles().Base
	muted := base.Foreground(m.theme.Muted)
	accent := base.Foreground(m.theme.Accent).Bold(true)

	descBudget := width - tagPickerCheckColW - tagPickerChipColW -
		2*lipgloss.Width(tagPickerColGap) - 2
	if descBudget < 12 {
		descBudget = 12
	}

	var b strings.Builder
	for i, row := range m.rows {
		check := "[ ]"
		if row.selected {
			check = "[x]"
		}
		var checkCell string
		if row.selected {
			checkCell = accent.Render(check)
		} else {
			checkCell = muted.Render(check)
		}

		chip := chipCell(m.theme, row.tag, true)
		desc := truncate(row.description, descBudget)
		descCell := base.Foreground(m.theme.FgMuted).Render(desc)

		row := lipgloss.JoinHorizontal(lipgloss.Top,
			checkCell, tagPickerColGap,
			chip, tagPickerColGap,
			descCell,
		)
		if i == m.cursor {
			row = lipgloss.NewStyle().
				Background(m.theme.Surface).
				Bold(true).
				Render(row)
		}
		b.WriteString(row)
		if i < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m tagPickerPopupModel) View() tea.View {
	base := m.theme.AppStyles().Base
	help := m.theme.AppStyles().Help

	title := base.Foreground(m.theme.Secondary).Bold(true).Render("Add commit tag types")
	subtitle := base.Foreground(m.theme.FgMuted).
		Render("Selected tags get appended to .commitcraft.toml with their four-color palette.")

	hintPairs := [][2]string{
		{"↑↓", "navigate"},
		{"space", "toggle"},
		{"a", "select all"},
		{"↵", "save"},
		{"esc", "cancel"},
	}
	parts := make([]string, 0, len(hintPairs)*2-1)
	for i, p := range hintPairs {
		if i > 0 {
			parts = append(parts, help.ShortSeparator.Render(" · "))
		}
		parts = append(parts,
			help.ShortKey.Render(p[0])+" "+help.ShortDesc.Render(p[1]),
		)
	}
	hint := strings.Join(parts, "")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		subtitle,
		"",
		m.body.View(),
		"",
		hint,
	)

	boxStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	return tea.NewView(boxStyle.Render(body))
}
