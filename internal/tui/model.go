package tui

import (
	"commit_craft_reborn/internal/storage"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
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
type Model struct {
	log             *log.Logger
	db              *storage.DB
	state           appState
	err             error
	list            list.Model
	scopeInput      textinput.Model
	msgInput        textarea.Model
	spinner         spinner.Model
	commitType      string
	commitScope     string
	commitMsg       string
	commitTranslate string // Field for the selected value
	FinalMessage    string // Exported to be read by main.go
	keys            KeyMap
	help            help.Model
}

// NewModel is the constructor for our model.
func NewModel(logger *log.Logger, database *storage.DB) (*Model, error) {
	// commitTypesList := NewCommitTypeList()
	workspaceCommits, err := database.GetCommits()
	workspaceCommitsList := setupList(workspaceCommits)

	// --- Component Initializations ---
	if err != nil {
		// Si hay un error al cargar, lo registramos y podemos devolverlo.
		logger.Error("Failed to load recent scopes from database", "error", err)
		return nil, err // O manejarlo de otra forma, como continuar con una lista vacía.
	}
	scopeInput := textinput.New()
	scopeInput.Placeholder = "module, file, etc..."

	msgInput := textarea.New()
	msgInput.Placeholder = "A short description of the changes..."

	spinner := spinner.New()
	spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	// --- End of Initializations ---

	viewModel := &Model{
		log:   logger,
		db:    database,
		state: stateChoosingType,
		// list:       commitTypesList,
		list:       workspaceCommitsList,
		scopeInput: scopeInput,
		msgInput:   msgInput,
		spinner:    spinner,
		keys:       listKeys(),
		help:       help.New(),
	}
	return viewModel, nil
}

// Init is the first command that runs when the program starts.
func (model *Model) Init() tea.Cmd {
	// Enter the alternate screen buffer on startup.
	return tea.EnterAltScreen
}
