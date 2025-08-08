package model

import (
	"commit_craft_reborn/pkg/config"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
)

// ViewMode defines the current view of the application.
type ViewMode int

const (
	ListView ViewMode = iota
	SelectCommitTypeView
	EnterScopeView
	EnterMessageView
)

// CommitType represents a type of commit (e.g., FIX, ADD).
type CommitType struct {
	Tag         string
	Desc        string
	Color       string // Added for styling from config
}

func (ct CommitType) FilterValue() string { return ct.Tag }
func (ct CommitType) Title() string       { return fmt.Sprintf("%s", ct.Tag) }
func (ct CommitType) Description() string { return ct.Desc }


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

func (c Commit) FilterValue() string {
	return fmt.Sprintf("%s %s %s", c.Type, c.Scope, c.MessageES)
}
func (c Commit) Title() string {
	return fmt.Sprintf("%d (%s) - %s", c.ID, c.CreatedAt.Format("02/01/06 15:04"), c.MessageES)
}
func (c Commit) Description() string { return fmt.Sprintf("Translated: %s", c.MessageEN) }

// Model is the main application state.
type Model struct {
	Mode           ViewMode
	CommitList     list.Model
	CommitTypeList list.Model
	Inputs         []textinput.Model
	NewCommit      Commit
	SelectedMessage string
	Err            error
}

// NewModel initializes and returns a new Model.
func NewModel(cfg *config.Config) Model {
	// Inputs for scope and message
	inputs := make([]textinput.Model, 2)
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Scope (e.g., folder/file)"
	inputs[0].CharLimit = 64
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Commit message in Spanish"
	inputs[1].CharLimit = 128

	// List of existing commits
	commitList := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	commitList.Title = "Commit History"

	// List of commit types from config
	var commitTypes []list.Item
	for _, t := range cfg.CommitTypes.Types {
		commitTypes = append(commitTypes, CommitType{Tag: t.Name, Desc: t.Description, Color: t.Color})
	}
	
	typeList := list.New(commitTypes, list.NewDefaultDelegate(), 0, 0)
	typeList.Title = "Select Commit Type"

	return Model{
		Mode:           ListView,
		CommitList:     commitList,
		CommitTypeList: typeList,
		Inputs:         inputs,
	}
}
