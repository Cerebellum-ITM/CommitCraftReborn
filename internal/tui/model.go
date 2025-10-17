package tui

import (
	"fmt"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"
	"commit_craft_reborn/internal/tui/styles"

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
	stateEditMessage
	stateConfirming
	stateDone
)

// model is the main struct that holds the entire application state.
type Model struct {
	Theme                   *styles.Theme
	pwd                     string
	log                     *logger.Logger
	db                      *storage.DB
	finalCommitTypes        []commit.CommitType
	state                   appState
	err                     error
	apiKeyInput             textinput.Model
	mainList                list.Model
	commitTypeList          list.Model
	fileList                list.Model
	fileListFilter          bool
	currentUpdateFileListFn UpdateFileListFunc
	gitStatusData           GitStatusData
	msgInput                *textarea.Model
	msgEdit                 *textarea.Model
	spinner                 spinner.Model
	iaViewport              viewport.Model
	focusedElement          focusableElement
	WritingStatusBar        statusbar.StatusBar
	logViewport             viewport.Model
	logViewVisible          bool
	commitType              string
	commitTypeColor         string
	commitScope             string
	commitMsg               string
	commitTranslate         string
	diffCode                string
	useDbCommmit            bool
	FinalMessage            string
	keys                    KeyMap
	help                    help.Model
	popup                   tea.Model
	width, height           int
	globalConfig            config.Config
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
	var statusBarInitialMessage string
	var WritingStatusBar statusbar.StatusBar
	theme := styles.NewCharmtoneTheme()

	apiKeyInput := textinput.New()
	gitStatusData, err := GetAllGitStatusData()
	if err != nil {
		log.Error("Failed to initialize git Status Data", "error", err)
		return nil, err
	}

	commitTypesList := NewCommitTypeList(finalCommitTypes, config.CommitFormat.TypeFormat)
	workspaceCommits, err := database.GetCommits(pwd)
	workspaceCommitsList := NewHistoryCommitList(
		workspaceCommits,
		pwd,
		config,
		theme,
	)

	if err != nil {
		log.Error("Failed to load recent scopes from database", "error", err)
		return nil, err
	}

	fileList, err := NewFileList(pwd, config.TUI.UseNerdFonts, gitStatusData)
	if err != nil {
		log.Error("Failed to initialize file list", "error", err)
		return nil, err
	}

	// --- Component Initializations ---
	msgInput := textarea.New()
	msgInput.SetStyles(theme.AppStyles().TextArea)
	msgInput.Prompt = "┃ "
	msgInput.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+enter"))
	msgInput.Placeholder = "A short description of the changes..."
	//
	msgEdit := textarea.New()
	msgEdit.SetStyles(theme.AppStyles().TextArea)
	msgEdit.Prompt = "┃ "
	msgEdit.KeyMap.DeleteAfterCursor = key.NewBinding(key.WithKeys("ctrl+c"))
	msgEdit.KeyMap.DeleteBeforeCursor = key.NewBinding(key.WithKeys("ctrl+z"))
	msgEdit.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+enter"))
	msgEdit.Placeholder = "A short description of the changes..."

	vp := viewport.New()
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Left: "┃"}).
		BorderForeground(lipgloss.BrightWhite).
		PaddingRight(2)

	spinner := spinner.New()
	spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	if config.TUI.IsAPIKeySet {
		initalState = stateChoosingCommit
		initialKeys = mainListKeys()
		statusBarInitialMessage = fmt.Sprintf(
			"choose, create, or edit a commit ::: %s",
			theme.AppStyles().Base.Foreground(theme.Tertiary).SetString(workspaceCommitsList.Title),
		)

		WritingStatusBar = statusbar.New(
			statusBarInitialMessage,
			statusbar.LevelInfo,
			50,
			theme,
		)
	} else {
		initalState = stateSettingAPIKey
		apiKeyInput.Placeholder = "gsk_..."
		apiKeyInput.EchoMode = textinput.EchoPassword
		apiKeyInput.Focus()
		initialKeys = textInputKeys()
		statusBarInitialMessage = "It is necessary to add a Groq API key"
		WritingStatusBar = statusbar.New(
			statusBarInitialMessage,
			statusbar.LevelFatal,
			50,
			theme,
		)
	}

	help := help.New()
	help.Styles = theme.AppStyles().Help
	// --- End of Initializations ---

	m := &Model{
		log:                     log,
		pwd:                     pwd,
		db:                      database,
		apiKeyInput:             apiKeyInput,
		state:                   initalState,
		mainList:                workspaceCommitsList,
		commitTypeList:          commitTypesList,
		iaViewport:              vp,
		focusedElement:          focusMsgInput,
		fileList:                fileList,
		fileListFilter:          false,
		currentUpdateFileListFn: ChooseUpdateFileListFunction(false),
		gitStatusData:           gitStatusData,
		WritingStatusBar:        WritingStatusBar,
		msgInput:                msgInput,
		msgEdit:                 msgEdit,
		spinner:                 spinner,
		keys:                    initialKeys,
		help:                    help,
		logViewVisible:          false,
		logViewport:             viewport.New(),
		globalConfig:            config,
		Theme:                   theme,
		useDbCommmit:            false,
	}
	return m, nil
}

// Init is the first command that runs when the program starts.
func (model *Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}
