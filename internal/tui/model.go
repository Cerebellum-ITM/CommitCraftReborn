package tui

import (
	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/components/statusbar"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/key"
	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/textarea"
	"github.com/charmbracelet/bubbles/v2/textinput"
	"github.com/charmbracelet/bubbles/v2/viewport"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type focusableElement int

const (
	focusMsgInput   focusableElement = iota // 0
	focusAIResponse                         // 1
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

var scopeFilePickerPwd string

const (
	stateChoosingType appState = iota
	stateSettingAPIKey
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
	apiKeyInput      textinput.Model
	mainList         list.Model
	commitTypeList   list.Model
	fileList         list.Model
	scopeInput       textinput.Model
	msgInput         textarea.Model
	spinner          spinner.Model
	iaViewport       viewport.Model
	focusedElement   focusableElement
	WritingStatusBar statusbar.StatusBar
	logViewport      viewport.Model
	logViewVisible   bool
	commitType       string
	commitTypeColor  string
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
	var initalState appState
	var initialKeys KeyMap

	apiKeyInput := textinput.New()
	if config.TUI.IsAPIKeySet {
		initalState = stateChoosingCommit
		initialKeys = mainListKeys()
	} else {
		initalState = stateSettingAPIKey
		apiKeyInput.Placeholder = "gsk_..."
		apiKeyInput.EchoMode = textinput.EchoPassword
		apiKeyInput.Focus()
		initialKeys = textInputKeys()
	}

	commitTypesList := NewCommitTypeList(finalCommitTypes, config.CommitFormat.TypeFormat)
	workspaceCommits, err := database.GetCommits(pwd)
	workspaceCommitsList := NewHistoryCommitList(
		workspaceCommits,
		pwd,
		config.CommitFormat.TypeFormat,
	)

	if err != nil {
		log.Error("Failed to load recent scopes from database", "error", err)
		return nil, err
	}

	fileList, err := NewFileList(pwd, config.TUI.UseNerdFonts)
	if err != nil {
		log.Error("Failed to initialize file list", "error", err)
		return nil, err
	}

	// --- Component Initializations ---
	scopeInput := textinput.New()
	scopeInput.Placeholder = "module, file, etc..."

	msgInput := textarea.New()
	msgInput.Prompt = lipgloss.NewStyle().Foreground(lipgloss.BrightGreen).Render("┃ ")
	msgInput.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+enter"))
	msgInput.Placeholder = "A short description of the changes..."
	msgInput.Styles.Focused.Base.BorderForeground(lipgloss.Blue)
	msgInput.Styles.Blurred.Base.BorderForeground(lipgloss.Black)

	vp := viewport.New()
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Left: "┃"}).
		BorderForeground(lipgloss.Black).
		PaddingRight(2)

	WritingStatusBar := statusbar.New("write your summary of the changes", statusbar.LevelInfo)
	spinner := spinner.New()
	spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	// --- End of Initializations ---

	m := &Model{
		log:              log,
		pwd:              pwd,
		db:               database,
		apiKeyInput:      apiKeyInput,
		state:            initalState,
		mainList:         workspaceCommitsList,
		commitTypeList:   commitTypesList,
		iaViewport:       vp,
		focusedElement:   focusMsgInput,
		fileList:         fileList,
		WritingStatusBar: WritingStatusBar,
		scopeInput:       scopeInput,
		msgInput:         msgInput,
		spinner:          spinner,
		keys:             initialKeys,
		help:             help.New(),
		logViewVisible:   false,
		logViewport:      viewport.New(),
		globalConfig:     config,
	}
	return m, nil
}

// Init is the first command that runs when the program starts.
func (model *Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}
