package tui

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/tui/styles"
)

type CommitTypeDelegate struct {
	list.DefaultDelegate
	TypeFormat string
	Theme      *styles.Theme
}
type CommitTypeItem struct {
	commit.CommitType
}

func (cti CommitTypeItem) Title() string { return cti.CommitType.Tag }

func (cti CommitTypeItem) Description() string { return cti.CommitType.Description }

func (cti CommitTypeItem) FilterValue() string {
	return cti.CommitType.Tag
}

func (d CommitTypeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(CommitTypeItem)
	if !ok {
		return
	}

	tag := strings.ToUpper(it.Title())
	if len(tag) > styles.CommitTypeChipInnerWidth {
		tag = tag[:styles.CommitTypeChipInnerWidth]
	}
	desc := it.Description()
	selected := index == m.Index()

	// Same selection rule as the History MasterList:
	//   - chip uses Block (strong) palette on the cursor row, Msg (dim)
	//     palette otherwise — and is always rendered with the same
	//     fixed inner width + center alignment so descriptions line up
	//     across rows and the tag sits balanced inside the chip.
	//   - description uses Msg + Bold under the cursor, plain Muted
	//     text otherwise.
	var chipStyle, descStyle lipgloss.Style
	if selected {
		chipStyle = styles.CommitTypeBlockStyle(d.Theme, it.Title()).Bold(true)
		descStyle = styles.CommitTypeMsgStyle(d.Theme, it.Title()).Bold(true)
	} else {
		chipStyle = styles.CommitTypeMsgStyle(d.Theme, it.Title())
		descStyle = lipgloss.NewStyle().Foreground(d.Theme.Muted)
	}
	chip := chipStyle.
		Width(styles.CommitTypeChipInnerWidth).
		Padding(0, 1).
		Align(lipgloss.Center).
		Render(tag)

	cursor := "  "
	if selected && d.Theme != nil && d.Theme.Secondary != nil {
		cursor = lipgloss.NewStyle().
			Foreground(d.Theme.Secondary).
			Bold(true).
			Render("❯ ")
	}

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		cursor,
		chip,
		" ",
		descStyle.Render(desc),
	)
	io.WriteString(w, row)
}

func NewCommitTypeList(
	commitTypes []commit.CommitType,
	commitFormat string,
	theme *styles.Theme,
) list.Model {
	items := make([]list.Item, len(commitTypes))
	for i, ct := range commitTypes {
		items[i] = CommitTypeItem{CommitType: ct}
	}

	delegate := CommitTypeDelegate{
		TypeFormat: commitFormat,
		Theme:      theme,
	}
	typeList := list.New(items, delegate, 0, 0)
	typeList.Title = "Choose Commit Type"
	typeList.SetFilteringEnabled(true)
	typeList.KeyMap.AcceptWhileFiltering = key.NewBinding(
		key.WithKeys("enter", "/", "ctrl+k", "ctrl+j"),
	)
	typeList.SetShowHelp(false)

	return typeList
}
