package tui

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/tui/styles"
)

type CommitTypeDelegate struct {
	list.DefaultDelegate
	TypeFormat string
	Color      string
	Theme      *styles.Theme
}
type CommitTypeItem struct {
	commit.CommitType
}

func (cti CommitTypeItem) Title() string { return cti.CommitType.Tag }
func (cti CommitTypeItem) Color() string { return cti.CommitType.Color }

func (cti CommitTypeItem) Description() string { return cti.CommitType.Description }

func (cti CommitTypeItem) FilterValue() string {
	return cti.CommitType.Tag
}

func (d CommitTypeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(CommitTypeItem)
	if !ok {
		return
	}

	commitType := it.Title()
	commitDesc := it.Description()
	commitColor := it.Color()
	formattedCommitType := fmt.Sprintf(d.TypeFormat, commitType)

	var renderedType, renderedDesc string

	if index == m.Index() {
		styleType := lipgloss.NewStyle().
			Foreground(lipgloss.Color(commitColor)).
			Bold(true)

		styleDesc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")) // Amarillo claro

		var cursor string
		if d.Theme != nil && d.Theme.Secondary != nil {
			cursor = lipgloss.NewStyle().
				Foreground(d.Theme.Secondary).
				Bold(true).
				Render("❯")
		} else {
			cursor = "❯"
		}

		renderedType = styleType.Render(formattedCommitType)
		renderedDesc = styleDesc.Render(commitDesc)

		fmt.Fprintf(w, "%s %s - %s", cursor, renderedType, renderedDesc)
	} else {
		styleType := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")) // Gris

		styleDesc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")) // Gris más oscuro

		renderedType = styleType.Render(formattedCommitType)
		renderedDesc = styleDesc.Render(commitDesc)

		fmt.Fprintf(w, "  %s - %s", renderedType, renderedDesc)
	}
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
