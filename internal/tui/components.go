package tui

import (
	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/storage"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

type CommitTypeDelegate struct {
	list.DefaultDelegate
	TypeFormat string
	Color      string
}
type CommitTypeItem struct {
	commit.CommitType
}

func (cti CommitTypeItem) Title() string { return cti.CommitType.Tag }
func (cti CommitTypeItem) Color() string { return cti.CommitType.Color }

func (cti CommitTypeItem) Description() string { return cti.CommitType.Description }

func (cti CommitTypeItem) FilterValue() string {
	return cti.CommitType.Tag + " " + cti.CommitType.Description
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

		renderedType = styleType.Render(formattedCommitType)
		renderedDesc = styleDesc.Render(commitDesc)

		fmt.Fprintf(w, "❯ %s - %s", renderedType, renderedDesc)
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

// HistoryCommitItem
type HistoryCommitItem struct {
	commit storage.Commit
}

func (hci HistoryCommitItem) Title() string {
	return fmt.Sprintf("[%s] %s", hci.commit.Type, hci.commit.Scope)
}

func (hci HistoryCommitItem) Description() string {
	return hci.commit.CreatedAt.Format("2006-01-02 15:04")
}

func (hci HistoryCommitItem) FilterValue() string { return hci.Title() + hci.commit.MessageEN }

func NewHistoryCommitList(workspaceCommits []storage.Commit) list.Model {
	items := make([]list.Item, len(workspaceCommits))
	for i, c := range workspaceCommits {
		items[i] = HistoryCommitItem{commit: c}
	}

	historyList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	historyList.Title = "Commit History"
	return historyList
}

func NewCommitTypeList(commitTypes []commit.CommitType, commitFormat string) list.Model {
	items := make([]list.Item, len(commitTypes))
	for i, ct := range commitTypes {
		items[i] = CommitTypeItem{CommitType: ct}
	}

	delegate := CommitTypeDelegate{
		TypeFormat: commitFormat,
	}
	typeList := list.New(items, delegate, 0, 0)
	typeList.Title = "Choose Commit Type"
	typeList.SetShowHelp(false)

	return typeList
}
