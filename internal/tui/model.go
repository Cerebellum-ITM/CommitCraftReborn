package tui

import (
	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// We use iota to create an "enum" for our application states.
type (
	appState      int
	openPopupMsg  struct{}
	closePopupMsg struct{}
	deleteItemMsg struct {
		ID int
	}
)

const (
	stateChoosingType appState = iota
	stateChoosingCommit
	stateShowLogs
	stateChoosingScope
	stateWritingMessage
	stateTranslating
	stateConfirming
	stateDone
)

// model is the main struct that holds the entire application state.
type Model struct {
	pwd              string
	log              *logger.Logger
	db               *storage.DB
	finalCommitTypes []commit.CommitType
	state            appState
	err              error
	mainList         list.Model
	commitTypeList   list.Model
	scopeInput       textinput.Model
	msgInput         textarea.Model
	spinner          spinner.Model
	logViewport      viewport.Model
	logViewVisible   bool
	commitType       string
	commitScope      string
	commitMsg        string
	commitTranslate  string
	FinalMessage     string
	keys             KeyMap
	help             help.Model
	popup            tea.Model
	width, height    int
	globalConfig     config.Config
}

// NewModel is the constructor for our model.
func NewModel(
	log *logger.Logger,
	database *storage.DB,
	config config.Config,
	finalCommitTypes []commit.CommitType,
	pwd string,
) (*Model, error) {
	commitTypesList := NewCommitTypeList(finalCommitTypes, config.CommitFormat.TypeFormat)
	workspaceCommits, err := database.GetCommits(pwd)
	workspaceCommitsList := NewHistoryCommitList(workspaceCommits, pwd)

	// --- Component Initializations ---
	if err != nil {
		// Si hay un error al cargar, lo registramos y podemos devolverlo.
		log.Error("Failed to load recent scopes from database", "error", err)
		return nil, err // O manejarlo de otra forma, como continuar con una lista vacía.
	}
	scopeInput := textinput.New()
	scopeInput.Placeholder = "module, file, etc..."

	msgInput := textarea.New()
	msgInput.Placeholder = "A short description of the changes..."

	spinner := spinner.New()
	spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	// --- End of Initializations ---

	m := &Model{
		log:            log,
		pwd:            pwd,
		db:             database,
		state:          stateChoosingCommit,
		commitTypeList: commitTypesList,
		mainList:       workspaceCommitsList,
		scopeInput:     scopeInput,
		msgInput:       msgInput,
		spinner:        spinner,
		keys:           listKeys(),
		help:           help.New(),
		logViewVisible: false,
		logViewport:    viewport.New(),
		globalConfig:   config,
	}
	return m, nil
}

// Init is the first command that runs when the program starts.
func (model *Model) Init() tea.Cmd {
	// Enter the alternate screen buffer on startup.
	return tea.EnterAltScreen
}
