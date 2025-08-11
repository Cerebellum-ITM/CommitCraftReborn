package tui

import (
	"commit_craft_reborn/internal/commit"
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

func (listItem Item) Title() string       { return listItem.title }
func (listItem Item) Description() string { return "" }
func (listItem Item) FilterValue() string { return listItem.title }

func setupList() list.Model {
	// Gets the commit types from our business logic.
	itemsAsStrings := commit.GetCommitTypes()

	// Converts strings into items for the list component.
	items := make([]list.Item, len(itemsAsStrings))
	for index, str := range itemsAsStrings {
		items[index] = Item{title: str}
	}

	// Setup the list component.
	listModel := list.New(items, list.NewDefaultDelegate(), 0, 0)
	listModel.Title = "Commit Types"
	listModel.SetShowHelp(false) // We're hiding the help for now.
	return listModel
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
