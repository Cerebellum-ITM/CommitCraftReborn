package tui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"
)

// TabID identifies one of the three persistent top-level tabs that are
// always visible in the new layout: Compose (commit creation), History
// (browse past commits/releases), Pipeline (AI pipeline inspector).
type TabID int

const (
	TabCompose TabID = iota
	TabHistory
	TabPipeline
)

// tabOrder is the rendering order of the tabs from left to right and the
// order Ctrl+1/2/3 binds to. History is first because that's the
// entry-point state where the user lands by default.
var tabOrder = [...]TabID{TabHistory, TabCompose, TabPipeline}

// tabLabel returns the human-readable label rendered in the tab bar.
func tabLabel(t TabID) string {
	switch t {
	case TabCompose:
		return "Compose"
	case TabHistory:
		return "History"
	case TabPipeline:
		return "Pipeline"
	}
	return "?"
}

// tabForState maps an appState to the tab it logically belongs to. Used
// to keep model.topTab in sync with state transitions that happen via the
// usual flow (Esc, Enter, etc.) instead of the tab bar.
func tabForState(s appState) TabID {
	switch s {
	case stateChoosingType,
		stateChoosingScope,
		stateWritingMessage,
		stateConfirming:
		return TabCompose
	case statePipeline:
		return TabPipeline
	default:
		// Everything else (history, releases, reword pickers, api key,
		// stateDone) falls under History; that's where the user lands by
		// default and where most flows originate.
		return TabHistory
	}
}

// defaultStateForTab is the state to land on when the user switches to a
// tab that hasn't been visited yet (no entry in lastStatePerTab).
func (model *Model) defaultStateForTab(t TabID) appState {
	switch t {
	case TabCompose:
		return stateWritingMessage
	case TabPipeline:
		return statePipeline
	case TabHistory:
		fallthrough
	default:
		if model.AppMode == ReleaseMode {
			return stateReleaseMainMenu
		}
		return stateChoosingCommit
	}
}

// switchToTab persists the current state under its tab and routes to the
// target tab, restoring the last visited state there or its default.
func (model *Model) switchToTab(target TabID) (*Model, bool, tea.Cmd) {
	current := tabForState(model.state)
	if current == target {
		return model, false, nil
	}
	if model.lastStatePerTab == nil {
		model.lastStatePerTab = map[TabID]appState{}
	}
	model.lastStatePerTab[current] = model.state

	var cmd tea.Cmd
	next, ok := model.lastStatePerTab[target]
	if !ok {
		next = model.defaultStateForTab(target)
		if target == TabCompose && next == stateWritingMessage {
			cmd = model.initFreshCompose()
		}
	}
	model.state = next
	model.topTab = target
	model.keys = keysForState(next, model.AppMode)
	return model, true, cmd
}

// initFreshCompose resets compose-related fields to their initial values
// so a tab switch into Compose lands on a clean draft (mirrors the setup
// performed by the "AddCommit" shortcut on the history list).
func (model *Model) initFreshCompose() tea.Cmd {
	model.currentCommit = storage.Commit{}
	model.keyPoints = nil
	model.resetScopes()
	if len(model.finalCommitTypes) > 0 {
		model.commitType = model.finalCommitTypes[0].Tag
		model.commitTypeColor = model.finalCommitTypes[0].Color
	}
	model.commitsKeysInput.SetValue("")
	model.commitTranslate = ""
	model.iaSummaryOutput = ""
	model.iaCommitRawOutput = ""
	model.iaTitleRawOutput = ""
	model.focusedElement = focusComposeSummary
	model.WritingStatusBar.Content = "Craft your commit"
	return model.commitsKeysInput.Focus()
}

// keysForState picks the keymap that matches a state, used after a tab
// switch teleports us into a new state without going through the regular
// transition handlers.
func keysForState(s appState, mode appMode) KeyMap {
	switch s {
	case stateChoosingCommit:
		return mainListKeys()
	case stateReleaseMainMenu:
		return releaseMainListKeys()
	case stateChoosingType:
		return listKeys()
	case stateChoosingScope:
		return fileListKeys()
	case stateWritingMessage:
		return writingMessageKeys()
	case stateRewordSelectCommit:
		return rewordSelectKeys()
	case stateReleaseChoosingCommits, stateReleaseBuildingText:
		return releaseKeys()
	case stateSettingAPIKey:
		return textInputKeys()
	case statePipeline:
		return pipelineKeys()
	}
	return mainListKeys()
}

// shouldShowTabBar hides the tab bar in states where switching tabs would
// be destructive or nonsensical (API key bootstrap, popups blocking input).
func (model *Model) shouldShowTabBar() bool {
	if model.state == stateSettingAPIKey {
		return false
	}
	return true
}

// renderTabBar draws the persistent tab strip across the top of the TUI.
// Each tab is wrapped by `│` separators. The two `│`s flanking the active
// tab and its label render in `theme.Primary` + bold to mark the
// selection; the remaining separators stay in `theme.Subtle` and inactive
// labels in `theme.Muted`. The right side keeps the keyboard shortcut
// hints (^1/^2/^3).
//
//	│ History │ Compose │ Pipeline │           ^1 history  ^2 compose …
func (model *Model) renderTabBar(width int) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	muted := base.Foreground(theme.Muted)
	activeLabel := base.Foreground(theme.FG).Bold(true)
	sepActive := base.Foreground(theme.Primary).Bold(true).Render("│")
	sepInactive := base.Foreground(theme.Subtle).Render("│")

	// Each slot between adjacent tabs is "active" if either of its two
	// neighbours is the selected tab. The leading separator (before the
	// first tab) and the trailing one (after the last tab) follow the
	// same rule against their single neighbour.
	sepFor := func(leftIdx, rightIdx int) string {
		var leftTab, rightTab TabID
		if leftIdx >= 0 {
			leftTab = tabOrder[leftIdx]
		}
		if rightIdx < len(tabOrder) {
			rightTab = tabOrder[rightIdx]
		}
		if (leftIdx >= 0 && leftTab == model.topTab) ||
			(rightIdx < len(tabOrder) && rightTab == model.topTab) {
			return sepActive
		}
		return sepInactive
	}

	parts := make([]string, 0, len(tabOrder)*4+1)
	parts = append(parts, sepFor(-1, 0))
	for i, tab := range tabOrder {
		var label string
		if tab == model.topTab {
			label = activeLabel.Render(tabLabel(tab))
		} else {
			label = muted.Render(tabLabel(tab))
		}
		parts = append(parts, " ", label, " ", sepFor(i, i+1))
	}
	leftBar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	// Right-aligned shortcut hints: "^1 compose · ^2 history · ^3 pipeline"
	hintCells := make([]string, 0, len(tabOrder)*3)
	for i, tab := range tabOrder {
		key := base.Foreground(theme.Primary).Render(fmt.Sprintf("^%d", i+1))
		name := muted.Render(strings.ToLower(tabLabel(tab)))
		hintCells = append(hintCells, key+" "+name)
		if i < len(tabOrder)-1 {
			hintCells = append(hintCells, "  ")
		}
	}
	rightBar := lipgloss.JoinHorizontal(lipgloss.Top, hintCells...)

	leftW := lipgloss.Width(leftBar)
	rightW := lipgloss.Width(rightBar)
	pad := max(2, width-leftW-rightW)

	// Render the persistent CWD + branch pills centered inside the
	// spacer between the tabs (left) and the keyboard-shortcut hints
	// (right). Pills only appear when there's enough breathing room —
	// at least 2 spaces of margin on each side and a 1-cell gap between
	// pills — otherwise we keep the original plain spacer so narrow
	// terminals don't crush the layout. The branch pill is dropped
	// first when space is tight; the CWD pill is dropped next.
	cwd := cwdDisplayPath(model.pwd)
	branch := model.currentBranch
	const minMargin = 2
	const pillsGap = " "
	if pad >= minMargin*2+5 {
		budget := pad - minMargin*2
		var combined string
		switch {
		case branch != "" && budget >= 14:
			// Split the budget between the two pills, biased toward the
			// CWD which is usually the longer string. Reserve room for
			// the 1-cell gap.
			gapW := lipgloss.Width(pillsGap)
			cwdBudget := (budget - gapW) * 6 / 10
			branchBudget := budget - gapW - cwdBudget
			cwdPill := statusbar.RenderCwdPill(cwd, cwdBudget)
			branchPill := statusbar.RenderBranchPill(branch, branchBudget)
			combined = cwdPill + pillsGap + branchPill
		default:
			combined = statusbar.RenderCwdPill(cwd, budget)
		}
		pillW := lipgloss.Width(combined)
		leftGap := (pad - pillW) / 2
		rightGap := pad - pillW - leftGap
		spacer := strings.Repeat(" ", leftGap) + combined + strings.Repeat(" ", rightGap)
		return leftBar + spacer + rightBar
	}

	spacer := strings.Repeat(" ", pad)
	return leftBar + spacer + rightBar
}

// cwdDisplayPath collapses $HOME to "~" so the CWD pill stays compact on
// the most common paths. Falls back to the raw path when $HOME can't be
// resolved or doesn't prefix pwd.
func cwdDisplayPath(pwd string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return pwd
	}
	switch {
	case pwd == home:
		return "~"
	case strings.HasPrefix(pwd, home+string(os.PathSeparator)):
		return "~" + strings.TrimPrefix(pwd, home)
	}
	return pwd
}
