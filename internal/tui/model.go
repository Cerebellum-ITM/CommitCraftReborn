package tui

import (
	"fmt"
	"image/color"

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

type (
	focusableElement int
	itemsOptions     struct {
		index int
		color color.Color
		icon  string
	}
	ToolInfo struct {
		name      string
		available bool
		textColor color.Color
		icon      string
	}
)

type Tools struct {
	xclip ToolInfo
	gh    ToolInfo
}

const (
	focusMsgInput   focusableElement = iota // 0
	focusAIResponse                         // 1
	focusListElement
	focusViewportElement
)

// We use iota to create an "enum" for our application states.
type (
	appState      int
	appMode       int
	popupType     int
	openListPopup struct {
		title         string
		color         color.Color
		items         []string
		itemsOptions  []itemsOptions
		width, height int
	}
	releaseAction struct {
		action string
	}
	CommitCraftTables int
	closeListPopup    struct{}
	openPopupMsg      struct {
		Type popupType
		Db   CommitCraftTables
	}
	closePopupMsg struct{}
	deleteItemMsg struct {
		ID int
		Db CommitCraftTables
	}
)

const (
	Confirmation popupType = iota
)

const (
	CommitMode appMode = iota
	ReleaseMode
)

const (
	commitDb CommitCraftTables = iota
	releaseDb
)

var scopeFilePickerPwd string

type releaseViewState struct {
	selecting      bool
	releaseCreated bool
}

const (
	stateChoosingType appState = iota
	stateSettingAPIKey
	stateChoosingCommit
	stateShowLogs
	stateChoosingScope
	stateWritingMessage
	stateEditMessage
	stateConfirming
	stateReleaseChoosingCommits
	stateReleaseBuildingText
	stateReleaseMainMenu
	stateDone
)

// model is the main struct that holds the entire application state.
type Model struct {
	AppMode                 appMode
	Theme                   *styles.Theme
	pwd                     string
	log                     *logger.Logger
	db                      *storage.DB
	ToolsInfo               Tools
	finalCommitTypes        []commit.CommitType
	state                   appState
	err                     error
	apiKeyInput             textinput.Model
	mainList                list.Model
	releaseCommitList       list.Model
	releaseViewport         viewport.Model
	releaseEditText         *textarea.Model
	releaseViewState        *releaseViewState
	releaseText             string
	releaseType             string
	releaseBranch           string
	releaseMainList         list.Model
	selectedCommitList      []WorkspaceCommitItem
	commitLivePreview       string
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
	appMode appMode,
) (*Model, error) {
	var initalState appState
	var initialKeys KeyMap
	var statusBarInitialMessage string
	var WritingStatusBar statusbar.StatusBar

	apiKeyInput := textinput.New()
	theme := styles.NewCharmtoneTheme(config.TUI.UseNerdFonts)

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

	workspaceReleses, err := database.GetReleases(pwd)
	if err != nil {
		log.Error("Failed to load recent releases from database", "error", err)
		return nil, err
	}
	releaseList := NewHistoryReleaseList(workspaceReleses, pwd, config, theme)

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

	releaseViewport := viewport.New()
	releaseViewport.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Left: "┃"}).
		BorderForeground(theme.FocusableElement).
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

		if appMode == ReleaseMode {
			initalState = stateReleaseMainMenu
			initialKeys = releaseMainListKeys()
			statusBarInitialMessage = fmt.Sprintf(
				"choose, create, or edit a release ::: %s",
				theme.AppStyles().
					Base.Foreground(theme.Tertiary).
					SetString(workspaceCommitsList.Title),
			)
		}

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
	toolInfo := CheckTools(*theme)
	// --- End of Initializations ---

	m := &Model{
		AppMode:                 appMode,
		ToolsInfo:               toolInfo,
		log:                     log,
		pwd:                     pwd,
		db:                      database,
		apiKeyInput:             apiKeyInput,
		state:                   initalState,
		mainList:                workspaceCommitsList,
		releaseMainList:         releaseList,
		releaseViewport:         releaseViewport,
		releaseViewState:        &releaseViewState{selecting: false, releaseCreated: false},
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
