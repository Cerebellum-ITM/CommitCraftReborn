package tui

import (
	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/storage"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

type commitTypeDelegate struct {
	list.DefaultDelegate
}

func (d commitTypeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	// 1. Hacemos una aserción de tipo para acceder a nuestro 'Item'.
	it, ok := listItem.(Item)
	if !ok {
		return
	}

	parts := strings.SplitN(it.Title(), " - ", 2)
	commitType := parts[0]
	commitDesc := ""
	if len(parts) > 1 {
		commitDesc = parts[1]
	}

	var renderedType, renderedDesc string

	if index == m.Index() {

		styleType := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")). // Magenta
			Bold(true)

		styleDesc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")) // Amarillo claro

		renderedType = styleType.Render(commitType)
		renderedDesc = styleDesc.Render(commitDesc)

		fmt.Fprintf(w, "❯ %s - %s", renderedType, renderedDesc)

	} else {
		styleType := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")) // Gris

		styleDesc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")) // Gris más oscuro

		renderedType = styleType.Render(commitType)
		renderedDesc = styleDesc.Render(commitDesc)

		fmt.Fprintf(w, "  %s - %s", renderedType, renderedDesc)
	}
}

type Item struct {
	title string
}

type CommitItem struct {
	commit storage.Commit
}

func (c CommitItem) Title() string {
	return fmt.Sprintf("[%s] %s", c.commit.Type, c.commit.Scope)
}

func (c CommitItem) Description() string {
	return c.commit.CreatedAt.Format("2006-01-02 15:04")
}

func (c CommitItem) FilterValue() string { return c.Title() + c.commit.MessageEN }

func (listItem Item) Title() string       { return listItem.title }
func (listItem Item) Description() string { return "" }
func (listItem Item) FilterValue() string { return listItem.title }

func setupList(workspaceCommits []storage.Commit) list.Model {
	items := make([]list.Item, len(workspaceCommits))
	for i, c := range workspaceCommits {
		items[i] = CommitItem{commit: c}
	}

	historyList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	historyList.Title = "Commit History"
	return historyList
}

func NewCommitTypeList() list.Model {
	itemsAsStrings := commit.GetCommitTypes()
	items := make([]list.Item, len(itemsAsStrings))
	for i, s := range itemsAsStrings {
		items[i] = Item{title: s}
	}

	// Crea y usa el delegado personalizado.
	delegate := commitTypeDelegate{}
	commitList := list.New(items, delegate, 0, 0)
	commitList.Title = "Commit Type"
	commitList.SetShowHelp(false)

	return commitList
}
