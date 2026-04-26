package tui

import (
	"fmt"
	"image/color"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/git"
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
)

const (
	focusMsgInput   focusableElement = iota // 0
	focusAIResponse                         // 1
	focusListElement
	focusViewportElement
	focusPipelineViewport // 4 — active viewport in pipeline tab
	focusPipelineDiffList // 5 — left file list in pipeline tab
	// New compose sections (Tab cycles through these in stateWritingMessage)
	focusComposeType
	focusComposeScope
	focusComposeSummary
	focusComposeKeypoints
	focusComposePipelineModels
	focusComposeAISuggestion
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
	statePipeline
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
	gitStatusData           git.StatusData
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
	// commitScopes is the multi-value scope list shown as chips in the
	// compose view. commitScope is the joined representation kept in sync
	// for db persistence and AI prompts.
	commitScopes []string
	// scopeChipIndex is the cursor inside the scope chip row when the
	// scope section has focus. Used so x/delete removes the right chip.
	scopeChipIndex int
	// keypointIndex is the cursor inside the key-points list when the
	// keypoints section has focus, used by the per-section delete keys.
	keypointIndex int
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
	// pendingRewordHash holds the resolved hash passed via -w until the user
	// picks a mode in the startup chooser popup. Cleared after the choice.
	pendingRewordHash string
	// topTab is the persistent top-level tab the user is on. Different from
	// model.activeTab (which is the inner editor/pipeline tab inside the
	// writing-message view).
	topTab TabID
	// lastStatePerTab remembers the last appState the user was on inside
	// each top-level tab so switching back resumes there instead of always
	// landing on the tab's default state.
	lastStatePerTab map[TabID]appState
	// pendingReleaseUpload, when non-nil, is the release the user has just
	// asked to publish on GitHub. We pop the version editor first and only
	// fire execUploadRelease after the user confirms the tag.
	pendingReleaseUpload *HistoryReleaseItem
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

	gitStatusData, err := git.GetAllGitStatusData()
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

	// When the CLI is invoked with a hash to reword we don't preconfigure the
	// state machine here; instead we resolve the hash and store it as
	// pendingRewordHash so Init() can pop a chooser asking the user whether
	// to reword as a regular commit or as a release-style commit.
	var pendingRewordHash string
	if rewordHash != "" && config.TUI.IsAPIKeySet {
		full, rerr := git.ResolveCommitHash(rewordHash)
		if rerr != nil {
			log.Error("Cannot resolve reword hash", "hash", rewordHash, "error", rerr)
			return nil, fmt.Errorf("cannot resolve commit %s: %w", rewordHash, rerr)
		}
		pendingRewordHash = full
	}

	m := &Model{
		AppMode:                 appMode,
		ToolsInfo:               toolInfo,
		finalCommitTypes:        finalCommitTypes,
		log:                     log,
		pwd:                     pwd,
		db:                      database,
		apiKeyInput:             apiKeyInput,
		state:                   initalState,
		pendingRewordHash:       pendingRewordHash,
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
		useDbCommmit:            false,
		OutputDirect:            outputDirect,
		Version:                 version,
		topTab:                  tabForState(initalState),
		lastStatePerTab:         map[TabID]appState{},
	}
	if len(finalCommitTypes) > 0 {
		m.commitType = finalCommitTypes[0].Tag
		m.commitTypeColor = finalCommitTypes[0].Color
	}
	return m, nil
}

// Init is the first command that runs when the program starts.
func (model *Model) Init() tea.Cmd {
	model.logsCh = model.log.Subscribe()
	cmds := []tea.Cmd{waitForLogLineCmd(model.logsCh)}
	if model.pendingRewordHash != "" {
		cmds = append(cmds, openRewordChooserCmd(model))
	}
	return tea.Batch(cmds...)
}

// openRewordChooserCmd builds the startup chooser popup that asks the user
// whether to reword the target commit using the regular commit AI pipeline or
// the release AI pipeline. The selection is dispatched as releaseAction with
// one of the rewordChooseAs* labels.
func openRewordChooserCmd(model *Model) tea.Cmd {
	short := model.pendingRewordHash
	if len(short) > 7 {
		short = short[:7]
	}
	w := model.width / 2
	if w < 40 {
		w = 60
	}
	h := model.height / 2
	if h < 8 {
		h = 10
	}
	return func() tea.Msg {
		return openListPopup{
			title:  fmt.Sprintf("Reword %s", short),
			color:  model.Theme.Primary,
			items:  []string{rewordChooseAsCommit, rewordChooseAsRelease},
			width:  w,
			height: h,
			itemsOptions: []itemsOptions{
				{index: 0, color: model.Theme.Primary, icon: model.Theme.AppSymbols().CommitCraft},
				{index: 1, color: model.Theme.Secondary, icon: model.Theme.AppSymbols().Rewrite},
			},
		}
	}
}

const (
	rewordChooseAsCommit  = "Reword as commit"
	rewordChooseAsRelease = "Reword as release"
)

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
