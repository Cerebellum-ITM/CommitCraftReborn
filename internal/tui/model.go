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

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	focusPipelineViewport  // 4 — active viewport in pipeline tab
	focusPipelineDiffList  // 5 — left file list in pipeline tab
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
	stateRewordSelectCommit
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
	keyPoints               []string
	commitsKeysInput        textarea.Model
	mainList                list.Model
	releaseCommitList       list.Model
	commitsKeysViewport     viewport.Model
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
	pipelineDiffList        list.Model
	currentUpdateFileListFn UpdateFileListFunc
	gitStatusData           GitStatusData
	msgEdit                 textarea.Model
	spinner                 spinner.Model
	iaViewport              viewport.Model
	focusedElement          focusableElement
	WritingStatusBar        statusbar.StatusBar
	logViewport             viewport.Model
	logViewVisible          bool
	logsCh                  <-chan string
	commitType              string
	commitTypeColor         string
	commitScope             string
	commitMsg               string
	commitTranslate         string
	diffCode                string
	iaSummaryOutput         string
	iaCommitRawOutput       string
	iaTitleRawOutput        string
	activeTab               int
	activePipelineStage     int
	pipelineViewport1       viewport.Model
	pipelineViewport2       viewport.Model
	pipelineViewport3       viewport.Model
	useDbCommmit            bool
	FinalMessage            string
	RewordHash              string
	OutputDirect            bool
	commitAndReword         bool
	currentCommit           storage.Commit
	draftMode               bool
	keys                    KeyMap
	help                    help.Model
	popup                   tea.Model
	mentionStart            int
	width, height           int
	globalConfig            config.Config
	Version                 string
}

// NewModel is the constructor for our model.
func NewModel(
	log *logger.Logger,
	database *storage.DB,
	config config.Config,
	finalCommitTypes []commit.CommitType,
	pwd string,
	appMode appMode,
	version string,
	outputDirect bool,
	rewordHash string,
) (*Model, error) {
	var initalState appState
	var initialKeys KeyMap
	var statusBarInitialMessage string
	var WritingStatusBar statusbar.StatusBar

	apiKeyInput := textinput.New()
	theme := styles.NewCharmtoneTheme(config.TUI.UseNerdFonts)

	commitsKeysInput := textarea.New()
	commitsKeysInput.SetHeight(4)
	commitsKeysInput.ShowLineNumbers = false
	commitsKeysInput.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("insert", "alt+tab"))
	commitsKeysInput.Placeholder = "Add a key point..."
	kpiStyles := theme.AppStyles().TextArea
	kpiStyles.Focused.Placeholder = theme.AppStyles().Base.Foreground(theme.FgMuted)
	kpiStyles.Cursor.Blink = true
	commitsKeysInput.SetStyles(kpiStyles)
	commitsKeysInput.SetPromptFunc(4, func(info textarea.PromptInfo) string {
		s := theme.AppStyles().KeyPointsInput
		if info.LineNumber == 0 {
			if info.Focused {
				return s.PromptFocused.Render()
			}
			return s.PromptBlurred.Render()
		}
		if info.Focused {
			return s.DotsFocused.Render()
		}
		return s.DotsBlurred.Render()
	})

	gitStatusData, err := GetAllGitStatusData()
	if err != nil {
		log.Error("Failed to initialize git Status Data", "error", err)
		return nil, err
	}

	commitTypesList := NewCommitTypeList(finalCommitTypes, config.CommitFormat.TypeFormat)
	workspaceCommits, err := database.GetCommits(pwd, "completed")
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
	msgEdit := textarea.New()
	msgEdit.SetStyles(theme.AppStyles().TextArea)
	msgEdit.Prompt = "┃ "
	msgEdit.KeyMap.DeleteAfterCursor = key.NewBinding(key.WithKeys("ctrl+c"))
	msgEdit.KeyMap.DeleteBeforeCursor = key.NewBinding(key.WithKeys("ctrl+z"))
	msgEdit.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("insert", "alt+tab"))
	msgEdit.Placeholder = "A short description of the changes..."

	ckiVp := viewport.New()
	ckiVp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Left: "┃"}).
		BorderForeground(lipgloss.BrightWhite).
		PaddingRight(2)

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

	pipelineVpStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.Border{Left: "┃"}).
		BorderForeground(theme.FocusableElement).
		PaddingRight(2)
	pvp1 := viewport.New()
	pvp1.Style = pipelineVpStyle
	pvp2 := viewport.New()
	pvp2.Style = pipelineVpStyle
	pvp3 := viewport.New()
	pvp3.Style = pipelineVpStyle

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
			version,
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
			version,
		)
	}

	help := help.New()
	help.Styles = theme.AppStyles().Help
	toolInfo := CheckTools(*theme)
	// --- End of Initializations ---

	// When the CLI is invoked with a hash to reword, jump straight into the
	// reword pipeline (commit type → scope → AI message) instead of the
	// commit history list. We resolve the partial hash to a full one so the
	// downstream git calls don't have to.
	var initialRewordHash, initialDiffCode string
	initialUseDbCommit := false
	if rewordHash != "" && config.TUI.IsAPIKeySet {
		full, rerr := ResolveCommitHash(rewordHash)
		if rerr != nil {
			log.Error("Cannot resolve reword hash", "hash", rewordHash, "error", rerr)
			return nil, fmt.Errorf("cannot resolve commit %s: %w", rewordHash, rerr)
		}
		initialRewordHash = full
		diff, dErr := GetCommitDiffSummary(full, config.Prompts.ChangeAnalyzerMaxDiffSize)
		if dErr != nil {
			log.Error("Error getting commit diff for reword", "error", dErr)
		}
		initialDiffCode = diff
		if commitGitData, gErr := GetCommitGitStatusData(full); gErr == nil {
			gitStatusData = commitGitData
		} else {
			log.Error("Error getting commit git status data for reword", "error", gErr)
		}
		initialUseDbCommit = true
		initalState = stateChoosingType
		initialKeys = listKeys()
		statusBarInitialMessage = fmt.Sprintf("Reword %s · select a prefix", full[:7])
		WritingStatusBar = statusbar.New(
			statusBarInitialMessage,
			statusbar.LevelInfo,
			50,
			theme,
			version,
		)
		ResetAndActiveFilterOnList(&commitTypesList)
	}

	m := &Model{
		AppMode:                 appMode,
		ToolsInfo:               toolInfo,
		log:                     log,
		pwd:                     pwd,
		db:                      database,
		apiKeyInput:             apiKeyInput,
		state:                   initalState,
		RewordHash:              initialRewordHash,
		diffCode:                initialDiffCode,
		mainList:                workspaceCommitsList,
		releaseMainList:         releaseList,
		releaseViewport:         releaseViewport,
		releaseViewState:        &releaseViewState{selecting: false, releaseCreated: false},
		commitTypeList:          commitTypesList,
		iaViewport:              vp,
		focusedElement:          focusMsgInput,
		fileList:                fileList,
		fileListFilter:          false,
		pipelineDiffList:        NewDiffFileList(gitStatusData, config.TUI.UseNerdFonts),
		currentUpdateFileListFn: ChooseUpdateFileListFunction(false),
		gitStatusData:           gitStatusData,
		WritingStatusBar:        WritingStatusBar,
		keyPoints:               []string{},
		commitsKeysInput:        commitsKeysInput,
		msgEdit:                 msgEdit,
		spinner:                 spinner,
		keys:                    initialKeys,
		help:                    help,
		logViewVisible:          false,
		logViewport:             viewport.New(),
		globalConfig:            config,
		Theme:                   theme,
		commitsKeysViewport:     ckiVp,
		pipelineViewport1:       pvp1,
		pipelineViewport2:       pvp2,
		pipelineViewport3:       pvp3,
		useDbCommmit:            initialUseDbCommit,
		OutputDirect:            outputDirect,
		Version:                 version,
	}
	return m, nil
}

// Init is the first command that runs when the program starts.
func (model *Model) Init() tea.Cmd {
	model.logsCh = model.log.Subscribe()
	return waitForLogLineCmd(model.logsCh)
}

// waitForLogLineCmd reads the next line from the logs subscription channel and
// turns it into a logLineMsg. When the channel is closed it emits
// logsChannelClosedMsg so we stop re-subscribing.
func waitForLogLineCmd(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logsChannelClosedMsg{}
		}
		return logLineMsg{line: line}
	}
}
