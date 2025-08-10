package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// We use iota to create an "enum" for our application states.
type appState int

const (
	stateChoosingType appState = iota
	stateChoosingScope
	stateWritingMessage
	stateTranslating
	stateConfirming
	stateDone
)

// model is the main struct that holds the entire application state.
type model struct {
	state       appState
	err         error
	list        list.Model
	scopeInput  textinput.Model
	msgInput    textarea.Model
	spinner     spinner.Model
	commitType  string
	commitScope string
	commitMsg   string
}

// NewModel is the constructor for our model.
func NewModel() (*model, error) {
	commitTypesList := setupList()

	viewModel := &model{
		state: stateChoosingType,
		list:  commitTypesList,
	}
	return viewModel, nil
}

// Init is the first command that runs when the program starts.
func (model *model) Init() tea.Cmd {
	// Enter the alternate screen buffer on startup.
	return tea.EnterAltScreen
}
