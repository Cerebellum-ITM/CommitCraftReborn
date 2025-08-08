package model

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
)

// Commit holds the data for a single git commit.
type Commit struct {
	ID        int
	Type      string
	Scope     string
	MessageES string
	MessageEN string
	Workspace string
	CreatedAt time.Time
}

// Implement the list.Item interface for Commit.
func (c Commit) FilterValue() string {
	return fmt.Sprintf("%d %s %s %s", c.ID, c.Scope, c.MessageES, c.MessageEN)
}
func (c Commit) Title() string {
	return fmt.Sprintf("%d (%s) - Original: %s", c.ID, c.CreatedAt.Format("02/01/2006 15:04"), c.MessageES)
}
func (c Commit) Description() string { return fmt.Sprintf("Translated: %s", c.MessageEN) }

// State is the main application state.
type State struct {
	List                  list.Model
	Inputs                []textinput.Model
	FocusIndex            int
	CurrentWorkspace      string
	StatusMessage         string
	Quitting              bool
	SelectedCommitMessage string
	Err                   error
}

// NewState initializes and returns a new State.
func NewState(workspace string) State {
	inputs := make([]textinput.Model, 2)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Scope (e.g., crm_customs)"
	inputs[0].Focus()
	inputs[0].CharLimit = 64
	inputs[0].Cursor.Blink = true
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Message in Spanish"
	inputs[1].CharLimit = 128

	commitList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	commitList.Title = "Commit History"
	commitList.SetShowHelp(true)
	commitList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithHelp("n", "new")),
			key.NewBinding(key.WithHelp("d", "delete")),
		}
	}

	return State{
		List:             commitList,
		Inputs:           inputs,
		CurrentWorkspace: workspace,
	}
}
