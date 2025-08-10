package tui

import (
	"commit_craft_reborn/internal/commit"

	"github.com/charmbracelet/bubbles/list"
)

// item is a wrapper to satisfy the list.Item interface.
type item struct {
	title string
}

func (listItem item) Title() string       { return listItem.title }
func (listItem item) Description() string { return "" }
func (listItem item) FilterValue() string { return listItem.title }

func setupList() list.Model {
	// Gets the commit types from our business logic.
	itemsAsStrings := commit.GetCommitTypes()

	// Converts strings into items for the list component.
	items := make([]list.Item, len(itemsAsStrings))
	for index, str := range itemsAsStrings {
		items[index] = item{title: str}
	}

	// Setup the list component.
	listModel := list.New(items, list.NewDefaultDelegate(), 0, 0)
	listModel.Title = "Commit Types"
	listModel.SetShowHelp(false) // We're hiding the help for now.
	return listModel
}
