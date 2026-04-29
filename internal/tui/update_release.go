package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"
)

func updateReleaseMainMenu(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ctrl+f cycles the release filter mode at any time. Mirrors the
		// workspace history flow: empty query → just swap the pill;
		// non-empty → reset+set so DefaultFilter re-runs against the new
		// FilterValue source.
		if msg.String() == "ctrl+f" {
			model.releaseHistoryView.CycleFilterMode()
			val := model.releaseHistoryView.FilterValue()
			if val == "" {
				model.releaseMainList.SetFilterText("")
				model.releaseMainList.SetFilterState(list.Unfiltered)
			} else {
				model.releaseMainList.SetFilterText("")
				model.releaseMainList.SetFilterText(val)
				model.releaseMainList.SetFilterState(list.Filtering)
			}
			syncReleaseHistorySelection(model)
			return model, nil
		}
		// FilterBar focus: route keys to the textinput. Esc clears+blurs;
		// Enter blurs; every other key forwards to the input and the
		// master list filter is kept in sync.
		if model.releaseHistoryView.IsFilterFocused() {
			switch msg.String() {
			case "esc":
				model.releaseHistoryView.ResetFilter()
				model.releaseHistoryView.BlurFilter()
				model.releaseMainList.SetFilterText("")
				model.releaseMainList.SetFilterState(list.Unfiltered)
				return model, nil
			case "enter":
				model.releaseHistoryView.BlurFilter()
				return model, nil
			}
			cmd, changed := model.releaseHistoryView.UpdateFilter(msg)
			if changed {
				val := model.releaseHistoryView.FilterValue()
				model.releaseMainList.SetFilterText(val)
				if val == "" {
					model.releaseMainList.SetFilterState(list.Unfiltered)
				} else {
					model.releaseMainList.SetFilterState(list.Filtering)
				}
				syncReleaseHistorySelection(model)
			}
			return model, cmd
		}
		switch {
		case key.Matches(msg, model.keys.SwapMode):
			model.releaseHistoryView.ToggleMode()
			return model, nil
		case key.Matches(msg, model.keys.CycleNext):
			model.releaseHistoryView.CycleLeftCursor(+1)
			return model, nil
		case key.Matches(msg, model.keys.CyclePrev):
			model.releaseHistoryView.CycleLeftCursor(-1)
			return model, nil
		case key.Matches(msg, model.keys.EditIaCommit):
			// Repurposed on this screen: jump back to the synthetic
			// release entry without holding ctrl+[.
			model.releaseHistoryView.JumpToRelease()
			return model, nil
		case key.Matches(msg, model.keys.Filter):
			return model, model.releaseHistoryView.FocusFilter()
		case key.Matches(msg, model.keys.ReleaseCommit):
			model.WritingStatusBar.Content = "Select the commits to create a release"
			model.state = stateReleaseChoosingCommits
			model.releaseCommitList = NewReleaseCommitList(model.pwd, model.Theme)
			model.releaseCommitList.Select(0)
			model.focusedElement = focusListElement
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
			model.keys = releaseKeys()
			return model, nil
		case key.Matches(msg, model.keys.Enter):
			var menuOptions []itemsOptions
			menu := []string{"Print in console", "Copy to clipboard", "Create release in repository"}
			menuOptions = append(menuOptions, itemsOptions{index: 0, color: model.Theme.Success, icon: model.Theme.AppSymbols().Console})
			menuOptions = append(menuOptions, itemsOptions{index: 1, color: model.ToolsInfo.xclip.textColor, icon: model.ToolsInfo.xclip.icon})
			menuOptions = append(menuOptions, itemsOptions{index: 2, color: model.ToolsInfo.gh.textColor, icon: model.ToolsInfo.gh.icon})
			return model, func() tea.Msg {
				return openListPopup{items: menu, itemsOptions: menuOptions, width: model.width / 2, height: model.height / 2, color: model.Theme.Success}
			}
		case key.Matches(msg, model.keys.Delete):
			return model, func() tea.Msg { return openPopupMsg{Type: Confirmation, Db: releaseDb} }
		case key.Matches(msg, model.keys.SwitchMode):
			model.AppMode = CommitMode
			model.state = stateChoosingCommit
			model.keys = mainListKeys()
			model.WritingStatusBar.Content = fmt.Sprintf(
				"choose, create, or edit a commit ::: %s",
				model.Theme.AppStyles().
					Base.Foreground(model.Theme.Tertiary).
					SetString(model.mainList.Title),
			)
			cmd = model.WritingStatusBar.ShowMessageForDuration(
				"Change app mode: Commit",
				statusbar.LevelWarning,
				2*time.Second,
			)
			return model, cmd

		}
	}

	model.releaseMainList, cmd = model.releaseMainList.Update(msg)
	syncReleaseHistorySelection(model)
	return model, cmd
}

// syncReleaseHistorySelection mirrors the master list's current selection
// into the ReleaseHistoryView's DualPanel. Synchronous — used after every
// cursor move where the lookup is fast enough to feel instant. The
// initial-entry path uses startReleaseHistorySync (async) so the UI
// stays responsive while the first git lookup runs.
func syncReleaseHistorySelection(model *Model) {
	item, ok := model.releaseMainList.SelectedItem().(HistoryReleaseItem)
	if !ok {
		model.releaseHistoryView.ClearRelease()
		return
	}
	r := item.release
	hashes := strings.Split(r.CommitList, ",")
	messages, err := git.LookupCommitMessages(hashes)
	if err != nil && model.log != nil {
		model.log.Warn("git commit lookup failed", "release_id", r.ID, "error", err)
	}
	calls := loadReleaseAICalls(model, r.ID)
	model.releaseHistoryView.SetRelease(r, messages, calls)
}

// releaseHistorySyncMsg is the result of an async release-history load.
// `cleared` is true when the master list had no selection; the
// dual panel is cleared instead of hydrated in that case.
type releaseHistorySyncMsg struct {
	cleared  bool
	release  storage.Release
	messages map[string]git.CommitMessage
	calls    []storage.AICall
}

// enterReleaseHistoryLoading flips the model into the loading screen
// state and returns the batch of commands that drive it: the async
// release-history sync (which produces the data the dual panel needs)
// plus the spinner tick that animates the "Loading…" frame. Used at
// every entry point into stateReleaseMainMenu where the lookup might
// be slow enough to flash the chrome.
func enterReleaseHistoryLoading(model *Model) tea.Cmd {
	model.releaseLoading = true
	return tea.Batch(
		startReleaseHistorySync(model),
		model.spinner.Tick,
	)
}

// startReleaseHistorySync kicks off the slow git+db lookups for the
// currently selected release on a background command. The returned cmd
// resolves to a releaseHistorySyncMsg that the global update handler
// applies to the dual panel.
func startReleaseHistorySync(model *Model) tea.Cmd {
	item, ok := model.releaseMainList.SelectedItem().(HistoryReleaseItem)
	if !ok {
		return func() tea.Msg { return releaseHistorySyncMsg{cleared: true} }
	}
	r := item.release
	// Capture only what we need so the closure doesn't pin the whole
	// model into the goroutine spawned by tea.
	logger := model.log
	calls := loadReleaseAICalls(model, r.ID)
	hashes := strings.Split(r.CommitList, ",")
	return func() tea.Msg {
		messages, err := git.LookupCommitMessages(hashes)
		if err != nil && logger != nil {
			logger.Warn("git commit lookup failed", "release_id", r.ID, "error", err)
		}
		return releaseHistorySyncMsg{
			release:  r,
			messages: messages,
			calls:    calls,
		}
	}
}

// loadReleaseAICalls is the release counterpart of loadCommitAICalls. It
// reads the release-side telemetry rows so the dual panel can render the
// stages telemetry strip. Today we don't have a release_ai_calls table
// yet, so this returns nil — a follow-up phase will add the table and
// flush the create-release pipeline through it. The hook lives here now
// so the dual panel doesn't have to know whether persistence is wired.
func loadReleaseAICalls(model *Model, releaseID int) []storage.AICall {
	_ = model
	_ = releaseID
	return nil
}

func updateReleaseBuildingText(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch model.focusedElement {
	case focusViewportElement:
		model.releaseViewport, cmd = model.releaseViewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Enter):
			var menuOptions []itemsOptions
			menu := []string{"Create item in CommitCraft", "Create release in Github"}
			menuOptions = append(menuOptions, itemsOptions{index: 0, color: model.Theme.Success, icon: model.Theme.AppSymbols().CommitCraft})
			menuOptions = append(menuOptions, itemsOptions{index: 1, color: model.ToolsInfo.gh.textColor, icon: model.ToolsInfo.gh.icon})
			return model, func() tea.Msg {
				return openListPopup{items: menu, width: model.width / 2, height: model.height / 2, itemsOptions: menuOptions}
			}
		case key.Matches(msg, model.keys.NextField):
			cmd = switchFocusElement(model)
			model.state = stateReleaseChoosingCommits
			return model, cmd
		case key.Matches(msg, model.keys.PrevField):
			cmd = switchFocusElement(model)
			model.state = stateReleaseChoosingCommits
			return model, cmd
		}
	}

	return model, cmd
}

func updateReleaseChoosingCommits(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch model.focusedElement {
	case focusListElement:
		model.releaseCommitList, cmd = model.releaseCommitList.Update(msg)
	case focusViewportElement:
		model.releaseViewport, cmd = model.releaseViewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.NextViewPort):
			if model.releaseViewState.releaseCreated {
				model.state = stateReleaseBuildingText
				model.focusedElement = focusViewportElement
				model.WritingStatusBar.Content = "Release creation"
				model.WritingStatusBar.Level = statusbar.LevelInfo
				model.commitLivePreview = model.releaseText
			}
			return model, nil
		case key.Matches(msg, model.keys.Enter):
			model.state = stateReleaseBuildingText
			model.focusedElement = focusViewportElement
			model.WritingStatusBar.Level = statusbar.LevelWarning
			model.WritingStatusBar.Content = "Making a request to the AI. Please wait ..."
			spinnerCmd := model.WritingStatusBar.StartSpinner()
			iaBuilderCmd := callIaReleaseBuilderCmd(model)
			model.releaseViewState.releaseCreated = true
			return model, tea.Batch(spinnerCmd, iaBuilderCmd)
		case key.Matches(msg, model.keys.AddCommit):
			item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem)
			if !ok {
				return model, nil
			}
			if item.Selected {
				item.Selected = false
				foundIndex := -1
				for i, r := range model.selectedCommitList {
					if r.Hash == item.Hash {
						foundIndex = i
						break
					}
				}
				model.selectedCommitList = append(model.selectedCommitList[:foundIndex], model.selectedCommitList[foundIndex+1:]...)
			} else {
				item.Selected = true
				model.selectedCommitList = append(model.selectedCommitList, item)
			}
			index := model.releaseCommitList.Index()
			cmd = model.releaseCommitList.SetItem(index, item)
			return model, cmd
		case key.Matches(msg, model.keys.Up, model.keys.Down):
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
		case key.Matches(msg, model.keys.NextField):
			cmd = switchFocusElement(model)
			return model, cmd
		case key.Matches(msg, model.keys.PrevField):
			cmd = switchFocusElement(model)
			return model, cmd
		case key.Matches(msg, model.keys.Esc):
			switch model.AppMode {
			case CommitMode:
				model.state = stateChoosingCommit
				model.keys = mainListKeys()
			case ReleaseMode:
				model.state = stateReleaseMainMenu
				model.keys = releaseMainListKeys()
				syncReleaseHistorySelection(model)
			}
			return model, nil
		}
	}

	return model, cmd
}
