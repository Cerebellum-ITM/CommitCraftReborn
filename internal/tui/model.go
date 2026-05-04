package tui

import (
	"fmt"
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/commit"
	configpkg "commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"
	"commit_craft_reborn/internal/tui/styles"
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
	// Release · choose commits sub-view focus targets. Drive the new
	// 6-zone layout (top band: filter/mode/list, middle: msg vp,
	// bottom: file list + per-file diff vp).
	focusReleaseChooseFilter
	focusReleaseChooseModeBar
	focusReleaseChooseCommitList
	focusReleaseChooseMsgVp
	focusReleaseChooseFileList
	focusReleaseChooseDiffVp
	// Output review view focus targets. Tab cycles report (left) ↔
	// content (right). On the right pane, ←/→ swap segment and ↑/↓
	// scroll the viewport.
	focusOutputReport
	focusOutputContent
)

// outputSegment identifies which slice of generated content the right
// panel of the Output view is currently rendering. Cycled with ←/→ when
// focusOutputContent is active.
type outputSegment int

const (
	outSegFinal outputSegment = iota
	outSegSummary
	outSegBody
	outSegTitle
	outSegChangelog
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
	stateConfirming
	stateReleaseChoosingCommits
	stateReleaseBuildingText
	stateReleaseMainMenu
	stateRewordSelectCommit
	stateDone
	statePipeline
	stateOutput
)

// model is the main struct that holds the entire application state.
type Model struct {
	AppMode             appMode
	Theme               *styles.Theme
	pwd                 string
	log                 *logger.Logger
	db                  *storage.DB
	ToolsInfo           Tools
	finalCommitTypes    []commit.CommitType
	state               appState
	err                 error
	apiKeyInput         textinput.Model
	keyPoints           []string
	commitsKeysInput    textarea.Model
	mainList            list.Model
	historyView         HistoryView
	releaseCommitList   list.Model
	commitsKeysViewport viewport.Model
	releaseViewport     viewport.Model
	// releaseChooseFilterBar is the top-band filter input for the
	// release · choose commits sub-view. Independent instance so its
	// focus/value never bleed into the main release view's filter.
	releaseChooseFilterBar ReleaseFilterBar
	// releaseChooseModeBar is the segmented pill toggle in the same
	// top band ("All / Selected" — when Selected is active the commit
	// list is filtered to only the queued commits).
	releaseChooseModeBar HistoryModeBar
	// releaseFilesList holds the changed files of the currently
	// selected commit; selecting an item swaps the diff vp content.
	releaseFilesList list.Model
	// releaseDiffViewport renders the diff of the file selected in
	// releaseFilesList. Only the per-file hunk is shown — the full
	// commit diff is split file-by-file when the commit changes.
	releaseDiffViewport viewport.Model
	// releaseSelectedCommitHash tracks which commit's per-file data is
	// currently materialized in releaseFilesList / releaseDiffViewport,
	// so navigation that doesn't move the selection skips the rebuild.
	releaseSelectedCommitHash string
	// releaseFileDiffsByCommit caches the parsed per-file diffs of every
	// commit shown in the picker so flipping selection is instant.
	releaseFileDiffsByCommit map[string]map[string]string
	releaseEditText          *textarea.Model
	releaseViewState         *releaseViewState
	releaseText              string
	// releaseBodyOutput / releaseTitleOutput / releaseFinalOutput hold
	// the per-stage AI outputs for the release pipeline. releaseText
	// stays in sync with releaseFinalOutput so legacy view paths keep
	// working unchanged.
	releaseBodyOutput  string
	releaseTitleOutput string
	releaseFinalOutput string
	// releaseMode is the UI-only release/merge toggle; it tags the run
	// in logs but does not branch the prompts. Defaults to "release".
	releaseMode        string
	releaseType        string
	releaseBranch      string
	releaseMainList    list.Model
	releaseHistoryView ReleaseHistoryView
	// releaseLoading is set when an async releaseHistorySync command is
	// in flight. The view renders a centered loading panel for
	// stateReleaseMainMenu while it's true so the user sees a clean
	// "Loading…" frame instead of half-painted chrome on entry.
	releaseLoading          bool
	selectedCommitList      []WorkspaceCommitItem
	commitLivePreview       string
	commitTypeList          list.Model
	fileList                list.Model
	fileListFilter          bool
	pipelineDiffList        list.Model
	currentUpdateFileListFn UpdateFileListFunc
	gitStatusData           git.StatusData
	spinner                 spinner.Model
	iaViewport              viewport.Model
	focusedElement          focusableElement
	WritingStatusBar        statusbar.StatusBar
	logViewport             viewport.Model
	logViewVisible          bool
	logsCh                  <-chan string
	commitType              string
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
	// pipelineModelStageIndex is the cursor inside the pipeline-models
	// row when that section has focus, picking which stage Enter opens
	// the model picker for.
	pipelineModelStageIndex int
	commitMsg               string
	commitTranslate         string
	diffCode                string
	iaSummaryOutput         string
	iaCommitRawOutput       string
	iaTitleRawOutput        string
	// iaChangelogEntry holds the markdown block the refiner produced for the
	// CHANGELOG. Empty when the feature is disabled, the file is missing, or
	// the AI call failed. Persisted to disk in createCommit().
	iaChangelogEntry string
	// iaChangelogTargetPath is the resolved absolute path to the CHANGELOG
	// file the refiner sampled. Reused by the writer so detection and write
	// touch the same file even if pwd changes mid-session.
	iaChangelogTargetPath string
	// iaChangelogSuggestedVersion is the version string the refiner was
	// asked to use; surfaced in the status bar after a successful run.
	iaChangelogSuggestedVersion string
	// iaChangelogMentionLine is the single bullet/sentence the refiner
	// produced to be appended to the commit body. Stage 2's stored body
	// (`iaCommitRawOutput`) is never modified — this line is concatenated
	// only when composing the final commit message.
	iaChangelogMentionLine string
	// changelogActive flips on at pipeline start when CHANGELOG support is
	// enabled and the file was detected. Drives the 4th stage card visibility
	// and the CHANGELOG status-bar pill.
	changelogActive bool
	// changelogFilePresent and changelogWillAutoUpdate back the persistent
	// CHANGELOG indicator pill rendered in WritingStatusBar — they survive
	// across pipeline runs and are refreshed by refreshChangelogState.
	changelogFilePresent    bool
	changelogWillAutoUpdate bool
	activeTab               int
	activePipelineStage     int
	pipelineViewport1       viewport.Model
	pipelineViewport2       viewport.Model
	pipelineViewport3       viewport.Model
	pipelineViewport4       viewport.Model
	// outputReportViewport scrolls the report column of the Output
	// review view (inputs + telemetry + stage outputs) so long content
	// stays reachable on small terminals.
	outputReportViewport viewport.Model
	// outputSegment picks the slice of generated content rendered in
	// the Output view's right viewport: assembled final message or one
	// of the four stage outputs.
	outputSegment    outputSegment
	pipeline         pipelineModel
	usePreloadedDiff bool
	// scopeDataStale flips on when a commit is loaded from the DB without a
	// linked git hash (drafts and history commits committed outside the
	// CLI). In that mode gitStatusData still reflects the live workspace,
	// so the scope picker cannot mark the commit's actual files as
	// modified. Drives the warning pill in WritingStatusBar.
	scopeDataStale bool
	// dbFileDiffs holds per-file diff text parsed out of a DB-loaded
	// commit's Diff_code so the Pipeline tab can render the changed-files
	// list and per-file diff without going to `git diff --staged`. Empty
	// when editing a fresh (non-DB) commit.
	dbFileDiffs  map[string]string
	FinalMessage string
	// AutodraftedID is the row id of the draft the autodraft-on-quit hook
	// just persisted (0 when nothing was saved). Surfaced to main.go so it
	// can print a confirmation line after the TUI tears down.
	AutodraftedID int
	// AutodraftedTab is the human-readable tab label ("Compose" / "Pipeline")
	// the user was on when the autodraft fired. Empty when no autodraft ran.
	AutodraftedTab string
	// hasLocalConfig caches whether a .commitcraft.toml file existed in
	// pwd at startup. Drives the persistent local-config pill in the
	// status bar; we don't re-stat on every render because the file
	// rarely materializes mid-session.
	hasLocalConfig  bool
	RewordHash      string
	OutputDirect    bool
	commitAndReword bool
	// pendingRewordHash holds the resolved hash passed via -w until the user
	// picks a mode in the startup chooser popup. Cleared after the choice.
	pendingRewordHash string
	// releaseRewordHash carries the original commit hash through the
	// release flow when the user picked "Rewrite as release/merge" in the
	// -w startup chooser. RewordHash itself stays empty until createRelease
	// finalises the flow — that way the post-TUI reword in main.go does
	// not fire prematurely if the user cancels mid-pick.
	releaseRewordHash string
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
	currentCommit        storage.Commit
	draftMode            bool
	keys                 KeyMap
	help                 help.Model
	popup                tea.Model
	mentionStart         int
	width, height        int
	globalConfig         configpkg.Config
	Version              string
	themeName            string
	// currentBranch is the git branch the TUI was launched from. Cached
	// so the persistent CWD/branch pill in the top tab bar doesn't shell
	// out on every render.
	currentBranch string
}

// NewModel is the constructor for our model.
func NewModel(
	log *logger.Logger,
	database *storage.DB,
	config configpkg.Config,
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
	themeName := config.TUI.Theme
	if themeName == "" {
		themeName = styles.DefaultThemeName
	}
	theme := styles.GetTheme(themeName, config.TUI.UseNerdFonts)

	commitsKeysInput := textarea.New()
	commitsKeysInput.SetHeight(4)
	commitsKeysInput.ShowLineNumbers = false
	commitsKeysInput.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("insert", "alt+tab"))
	commitsKeysInput.Placeholder = "Add a key point..."
	kpiStyles := theme.AppStyles().TextArea
	kpiStyles.Focused.Placeholder = theme.AppStyles().Base.Foreground(theme.FgMuted)
	kpiStyles.Cursor.Blink = true
	kpiStyles.Cursor.Color = theme.Primary
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

	commitTypesList := NewCommitTypeList(finalCommitTypes, config.CommitFormat.TypeFormat, theme)
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
	ckiVp := viewport.New()
	ckiVp.Style = lipgloss.NewStyle().
		BorderForeground(lipgloss.BrightWhite).
		PaddingRight(2)

	vp := viewport.New()
	vp.Style = lipgloss.NewStyle().
		BorderForeground(lipgloss.BrightWhite).
		PaddingRight(2)

	releaseViewport := viewport.New()
	releaseViewport.Style = lipgloss.NewStyle().
		BorderForeground(theme.FocusableElement).
		PaddingRight(2)

	// Release · choose-commits sub-view extras: a dedicated filter bar
	// + mode bar (so the top band mirrors the main release view chrome)
	// plus the per-file diff vp and a placeholder files list. Real items
	// land when the user enters the state and a commit is selected.
	releaseChooseFilterBar := NewReleasePickerFilterBar(theme)
	releaseChooseModeBar := NewHistoryModeBar(theme)
	releaseChooseModeBar.SetLabels("All commits", "Selected only")

	releaseDiffViewport := viewport.New()
	releaseDiffViewport.Style = lipgloss.NewStyle().
		BorderForeground(theme.FocusableElement).
		PaddingRight(2)

	releaseFilesList := list.New(
		[]list.Item{},
		diffFileDelegate{useNerdFonts: config.TUI.UseNerdFonts},
		0, 0,
	)
	releaseFilesList.SetShowHelp(false)
	releaseFilesList.SetShowPagination(false)
	releaseFilesList.SetShowTitle(false)
	releaseFilesList.SetShowStatusBar(false)
	releaseFilesList.SetFilteringEnabled(false)

	pipelineVpStyle := lipgloss.NewStyle().
		BorderForeground(theme.FocusableElement).
		PaddingRight(2)
	pvp1 := viewport.New()
	pvp1.Style = pipelineVpStyle
	pvp2 := viewport.New()
	pvp2.Style = pipelineVpStyle
	pvp3 := viewport.New()
	pvp3.Style = pipelineVpStyle
	pvp4 := viewport.New()
	pvp4.Style = pipelineVpStyle

	spinner := spinner.New()
	spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	if config.TUI.IsAPIKeySet {
		initalState = stateChoosingCommit
		initialKeys = mainListKeys()
		statusBarInitialMessage = "choose, create, or edit a commit"

		if appMode == ReleaseMode {
			initalState = stateReleaseMainMenu
			initialKeys = releaseMainListKeys()
			statusBarInitialMessage = "choose, create, or edit a release"
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

	branch, branchErr := git.GetCurrentGitBranch()
	if branchErr != nil {
		// Non-fatal: the pill just stays empty and we keep going.
		log.Debug("could not resolve current git branch", "err", branchErr)
		branch = ""
	}

	m := &Model{
		AppMode:                  appMode,
		ToolsInfo:                toolInfo,
		finalCommitTypes:         finalCommitTypes,
		log:                      log,
		pwd:                      pwd,
		currentBranch:            branch,
		db:                       database,
		apiKeyInput:              apiKeyInput,
		state:                    initalState,
		pendingRewordHash:        pendingRewordHash,
		mainList:                 workspaceCommitsList,
		historyView:              NewHistoryView(theme),
		releaseMainList:          releaseList,
		releaseHistoryView:       NewReleaseHistoryView(theme),
		releaseViewport:          releaseViewport,
		releaseChooseFilterBar:   releaseChooseFilterBar,
		releaseChooseModeBar:     releaseChooseModeBar,
		releaseFilesList:         releaseFilesList,
		releaseDiffViewport:      releaseDiffViewport,
		releaseFileDiffsByCommit: map[string]map[string]string{},
		releaseViewState:         &releaseViewState{selecting: false, releaseCreated: false},
		commitTypeList:           commitTypesList,
		iaViewport:               vp,
		outputReportViewport:     viewport.New(),
		focusedElement:           focusMsgInput,
		fileList:                 fileList,
		fileListFilter:           false,
		pipelineDiffList:         NewDiffFileList(gitStatusData, config.TUI.UseNerdFonts),
		currentUpdateFileListFn:  ChooseUpdateFileListFunction(false),
		gitStatusData:            gitStatusData,
		WritingStatusBar:         WritingStatusBar,
		keyPoints:                []string{},
		commitsKeysInput:         commitsKeysInput,
		spinner:                  spinner,
		keys:                     initialKeys,
		help:                     help,
		logViewVisible:           false,
		logViewport:              viewport.New(),
		globalConfig:             config,
		Theme:                    theme,
		commitsKeysViewport:      ckiVp,
		pipelineViewport1:        pvp1,
		pipelineViewport2:        pvp2,
		pipelineViewport3:        pvp3,
		pipelineViewport4:        pvp4,
		pipeline:                 newPipelineModel(),
		releaseMode:              "release",
		usePreloadedDiff:         false,
		OutputDirect:             outputDirect,
		Version:                  version,
		topTab:                   tabForState(initalState),
		lastStatePerTab:          map[TabID]appState{},
		themeName:                themeName,
		hasLocalConfig:           configpkg.HasLocalConfig(pwd),
	}
	if len(finalCommitTypes) > 0 {
		m.commitType = finalCommitTypes[0].Tag
	}
	m.pipeline.stages[stageSummary].Model = config.Prompts.ChangeAnalyzerPromptModel
	m.pipeline.stages[stageBody].Model = config.Prompts.CommitBodyGeneratorPromptModel
	m.pipeline.stages[stageTitle].Model = config.Prompts.CommitTitleGeneratorPromptModel
	m.pipeline.stages[stageChangelog].Model = config.Changelog.PromptModel
	refreshPipelineNumstat(m)
	applyPipelineFilesDelegate(m)
	setDiffFromSelectedFile(m)
	m.refreshChangelogState()
	m.syncRewordIndicator()
	m.syncLocalConfigIndicator()
	m.historyView.SetBodyRenderer(func(text string, width int) string {
		return m.renderCommitMessage(text, width)
	})
	m.releaseHistoryView.SetBodyRenderer(func(text string, width int) string {
		return m.renderCommitMessage(text, width)
	})
	// When the user boots straight into Release mode, flag the loading
	// state here so the very first frame is the loading panel; Init()
	// then dispatches the async sync. The CommitMode startup path
	// doesn't need this — the workspace history is fast to render.
	if m.state == stateReleaseMainMenu {
		m.releaseLoading = true
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
	// If the app booted straight into Release mode, kick off the async
	// release-history sync (and the spinner that animates the loading
	// frame) before any user input arrives.
	if model.releaseLoading {
		cmds = append(cmds, startReleaseHistorySync(model), model.spinner.Tick)
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
	rewordChooseAsCommit  = "Reword this commit"
	rewordChooseAsRelease = "Rewrite as release/merge"
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
