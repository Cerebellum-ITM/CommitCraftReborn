package model

import (
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
	Tag  string
	Desc string
}

func (ct CommitType) FilterValue() string { return ct.Tag }
func (ct CommitType) Title() string       { return ct.Tag }
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
func NewModel() Model {
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

	// List of commit types
	commitTypes := []list.Item{
		CommitType{Tag: "[FIX]", Desc: "Bug fixes"},
		CommitType{Tag: "[REF]", Desc: "Refactoring"},
		CommitType{Tag: "[ADD]", Desc: "Adding new modules/features"},
		CommitType{Tag: "[REM]", Desc: "Removing resources"},
		CommitType{Tag: "[REV]", Desc: "Reverting commits"},
		CommitType{Tag: "[MOV]", Desc: "Moving files or code"},
		CommitType{Tag: "[REL]", Desc: "Release commits"},
		CommitType{Tag: "[IMP]", Desc: "Incremental improvements"},
		CommitType{Tag: "[MERGE]", Desc: "Merge commits"},
		CommitType{Tag: "[CLA]", Desc: "Signing the Contributor License"},
		CommitType{Tag: "[I18N]", Desc: "Translation changes"},
		CommitType{Tag: "[PERF]", Desc: "Performance patches"},
		CommitType{Tag: "[WIP]", Desc: "Work in progress"},
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
