package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
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
		stateEditMessage,
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
		return stateChoosingType
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
func (model *Model) switchToTab(target TabID) (*Model, bool) {
	current := tabForState(model.state)
	if current == target {
		return model, false
	}
	if model.lastStatePerTab == nil {
		model.lastStatePerTab = map[TabID]appState{}
	}
	model.lastStatePerTab[current] = model.state

	next, ok := model.lastStatePerTab[target]
	if !ok {
		next = model.defaultStateForTab(target)
	}
	model.state = next
	model.topTab = target
	model.keys = keysForState(next, model.AppMode)
	return model, true
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
	case stateEditMessage:
		return editingMessageKeys()
	case stateRewordSelectCommit:
		return rewordSelectKeys()
	case stateReleaseChoosingCommits, stateReleaseBuildingText:
		return releaseKeys()
	case stateSettingAPIKey:
		return textInputKeys()
	case statePipeline:
		if mode == ReleaseMode {
			return releaseMainListKeys()
		}
		return mainListKeys()
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
// Left side: the three tab labels with a vertical-bar marker in front of
// the active one. Right side: the keyboard shortcut hints (^1/^2/^3).
//
//	│ Compose  │ History  │ Pipeline                ^1 compose  ^2 history …
func (model *Model) renderTabBar(width int) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	muted := base.Foreground(theme.Muted)
	activeLabel := base.Foreground(theme.FG).Bold(true)
	separator := muted.Render("│")
	marker := base.Foreground(theme.Primary).Render("│")

	tabCells := make([]string, 0, len(tabOrder)*3)
	for i, tab := range tabOrder {
		var cell string
		if tab == model.topTab {
			cell = marker + " " + activeLabel.Render(tabLabel(tab))
		} else {
			cell = "  " + muted.Render(tabLabel(tab))
		}
		tabCells = append(tabCells, cell)
		if i < len(tabOrder)-1 {
			tabCells = append(tabCells, " ", separator)
		}
	}
	leftBar := lipgloss.JoinHorizontal(lipgloss.Top, tabCells...)

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
	spacer := strings.Repeat(" ", pad)

	return leftBar + spacer + rightBar
}

// buildPipelineDummyView is the placeholder content for the Pipeline tab
// until the actual pipeline inspector is rebuilt against the new layout.
func (model *Model) buildPipelineDummyView(width, height int) string {
	theme := model.Theme
	body := lipgloss.JoinVertical(
		lipgloss.Center,
		theme.AppStyles().Base.
			Foreground(theme.Secondary).
			Bold(true).
			Render("Pipeline"),
		"",
		theme.AppStyles().Base.
			Foreground(theme.Muted).
			Render("AI pipeline inspector — coming soon."),
	)
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		body,
	)
}
