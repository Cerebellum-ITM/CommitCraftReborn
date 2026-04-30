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

// initFreshReleaseCompose seeds the release commit-picker (Compose tab in
// ReleaseMode) the same way the `r` shortcut does from the release main
// menu: rebuild the workspace list, focus the list, prime the live
// preview / per-file cache, and reset the filter bar.
func (model *Model) initFreshReleaseCompose() tea.Cmd {
	model.releaseCommitList = NewReleaseCommitList(model.pwd, model.Theme)
	model.releaseCommitList.Select(0)
	model.focusedElement = focusReleaseChooseCommitList
	if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
		model.commitLivePreview = item.Preview
	}
	model.releaseSelectedCommitHash = ""
	model.releaseChooseFilterBar.Reset()
	model.releaseChooseFilterBar.Blur()
	model.loadReleaseCommitFiles()
	model.WritingStatusBar.Content = "Select the commits to create a release"
	model.WritingStatusBar.Level = statusbar.LevelInfo
	return nil
}

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
			syncCmd := syncReleaseHistorySelection(model)
			return model, syncCmd
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
				cmd = tea.Batch(cmd, syncReleaseHistorySelection(model))
			}
			return model, cmd
		}
		switch msg.String() {
		case "pgup", "pgdown", "ctrl+up", "ctrl+down":
			panelCmd := model.releaseHistoryView.UpdatePanel(msg)
			return model, panelCmd
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
			model.focusedElement = focusReleaseChooseCommitList
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
			model.releaseSelectedCommitHash = ""
			model.releaseChooseFilterBar.Reset()
			model.releaseChooseFilterBar.Blur()
			model.loadReleaseCommitFiles()
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
	syncCmd := syncReleaseHistorySelection(model)
	return model, tea.Batch(cmd, syncCmd)
}

// releaseCommitsResolvedMsg is the result of a single async git lookup
// for one release's commit list — both the on-demand path triggered by
// a cache miss on the selected entry and the prefetch path that warms
// the cache for the cursor's ±N neighbours. The handler in update.go
// always writes the messages to the view's cache; only the on-demand
// kind (fromSelected == true) goes on to refresh the dual panel and
// clear the inline spinner.
type releaseCommitsResolvedMsg struct {
	releaseID    int
	release      storage.Release
	messages     map[string]git.CommitMessage
	calls        []storage.AICall
	fromSelected bool
}

// releasePrefetchRadius is how many neighbours of the selected release
// get their commit messages fetched in the background. With ±2 the
// cursor can scroll five entries deep without a single cache miss; any
// wider just multiplies the git-show fork+exec cost the user will not
// see anyway because most navigation is short-range.
const releasePrefetchRadius = 2

// syncReleaseHistorySelection mirrors the master list's current selection
// into the ReleaseHistoryView's DualPanel. Cache-aware: a hit applies
// synchronously; a miss applies whatever was already on screen, starts
// the inline spinner on the WritingStatusBar, and dispatches an async
// git lookup for the selected entry. Either way it kicks off prefetch
// commands for the ±releasePrefetchRadius neighbours so the next cursor
// step is also a hit. The returned tea.Cmd is the batch of async work
// the caller has to thread back into Update.
func syncReleaseHistorySelection(model *Model) tea.Cmd {
	item, ok := model.releaseMainList.SelectedItem().(HistoryReleaseItem)
	if !ok {
		model.releaseHistoryView.ClearRelease()
		model.releaseHistoryView.SetCurrentReleaseID(0)
		model.WritingStatusBar.StopSpinner()
		return nil
	}
	r := item.release
	model.releaseHistoryView.SetCurrentReleaseID(r.ID)
	calls := loadReleaseAICalls(model, r.ID)

	if cached, hit := model.releaseHistoryView.CachedCommits(r.ID); hit {
		model.releaseHistoryView.SetRelease(r, cached, calls)
		model.WritingStatusBar.StopSpinner()
		return prefetchReleaseNeighbors(model)
	}

	// Cache miss: keep whatever the dual panel already shows so the user
	// doesn't see a flash of empty chrome, light the spinner, and run
	// the lookup in the background. Prefetch goes on the same batch so
	// neighbours warm in parallel.
	cmds := []tea.Cmd{model.WritingStatusBar.StartSpinner()}
	if model.releaseHistoryView.BeginFetch(r.ID) {
		cmds = append(cmds, fetchReleaseCommits(r, calls, true))
	}
	if pre := prefetchReleaseNeighbors(model); pre != nil {
		cmds = append(cmds, pre)
	}
	return tea.Batch(cmds...)
}

// prefetchReleaseNeighbors returns a batch of background git lookups
// for the ±releasePrefetchRadius releases around the master list's
// current cursor. Releases that are already cached (or already being
// fetched) are skipped, so a steady scroll only enqueues the freshly
// uncovered edge and reuses everything else.
func prefetchReleaseNeighbors(model *Model) tea.Cmd {
	items := model.releaseMainList.VisibleItems()
	if len(items) == 0 {
		return nil
	}
	center := model.releaseMainList.Index()
	var cmds []tea.Cmd
	for delta := -releasePrefetchRadius; delta <= releasePrefetchRadius; delta++ {
		if delta == 0 {
			continue
		}
		idx := center + delta
		if idx < 0 || idx >= len(items) {
			continue
		}
		hri, ok := items[idx].(HistoryReleaseItem)
		if !ok {
			continue
		}
		if !model.releaseHistoryView.BeginFetch(hri.release.ID) {
			continue
		}
		cmds = append(cmds, fetchReleaseCommits(hri.release, nil, false))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// fetchReleaseCommits is the actual goroutine body — a tea.Cmd that runs
// git.LookupCommitMessages off the UI thread and emits a
// releaseCommitsResolvedMsg the dispatch loop will route back to the
// view's cache. `calls` is precomputed by the caller (only the on-demand
// path needs it; prefetch passes nil to skip the DB roundtrip).
func fetchReleaseCommits(r storage.Release, calls []storage.AICall, fromSelected bool) tea.Cmd {
	hashes := strings.Split(r.CommitList, ",")
	return func() tea.Msg {
		messages, _ := git.LookupCommitMessages(hashes)
		return releaseCommitsResolvedMsg{
			releaseID:    r.ID,
			release:      r,
			messages:     messages,
			calls:        calls,
			fromSelected: fromSelected,
		}
	}
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
			// Tab from the build-text view goes back to the picker. Reset
			// to a focus value that the picker dispatch recognises —
			// otherwise focusedElement stays at focusListElement /
			// focusViewportElement (legacy enum from switchFocusElement)
			// and updateReleaseChoosingCommits' focus switch matches no
			// case, leaving the commit list permanently unresponsive.
			model.focusedElement = focusReleaseChooseCommitList
			model.state = stateReleaseChoosingCommits
			return model, nil
		case key.Matches(msg, model.keys.PrevField):
			model.focusedElement = focusReleaseChooseCommitList
			model.state = stateReleaseChoosingCommits
			return model, nil
		}
	}

	return model, cmd
}

// releaseChooseFocusOrder is the canonical Tab cycle for the
// release · choose commits sub-view. Filter → CommitList →
// CommitMessage vp → Files list → Diff vp. The "All / Selected"
// indicator is no longer in the cycle — it now toggles via ctrl+e
// from any focus.
var releaseChooseFocusOrder = []focusableElement{
	focusReleaseChooseFilter,
	focusReleaseChooseCommitList,
	focusReleaseChooseMsgVp,
	focusReleaseChooseFileList,
	focusReleaseChooseDiffVp,
}

// isReleaseChooseFocus reports whether the given focus token belongs to
// the release commit picker. Used by updateReleaseChoosingCommits as a
// defensive guard against state transitions that leave focusedElement
// pointing at a legacy zone.
func isReleaseChooseFocus(f focusableElement) bool {
	for _, v := range releaseChooseFocusOrder {
		if v == f {
			return true
		}
	}
	return false
}

// cycleReleaseChooseFocus advances/retreats the picker focus and keeps
// the filter bar's textinput state coherent with the new focus. Returns
// any cmd produced by Focus()ing the textinput (textinput.Blink) so the
// cursor blinks while the user types.
func cycleReleaseChooseFocus(model *Model, forward bool) tea.Cmd {
	cur := -1
	for i, f := range releaseChooseFocusOrder {
		if f == model.focusedElement {
			cur = i
			break
		}
	}
	if cur == -1 {
		model.focusedElement = releaseChooseFocusOrder[0]
		return model.releaseChooseFilterBar.Focus()
	}
	step := 1
	if !forward {
		step = -1
	}
	next := (cur + step + len(releaseChooseFocusOrder)) % len(releaseChooseFocusOrder)
	model.focusedElement = releaseChooseFocusOrder[next]
	if model.focusedElement == focusReleaseChooseFilter {
		// Cycling INTO the filter must focus the textinput too, otherwise
		// typing does nothing because textinput.Update no-ops on !focus.
		return model.releaseChooseFilterBar.Focus()
	}
	model.releaseChooseFilterBar.Blur()
	return nil
}

func updateReleaseChoosingCommits(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Defensive coercion: any path that lands here with focusedElement
	// outside the picker's focus set (typical example: switchFocusElement
	// leaving us on focusListElement / focusViewportElement) would make
	// the focus switch below match no case, so the commit list never
	// receives the msg and looks frozen. Snap to the commit list when
	// that happens.
	if !isReleaseChooseFocus(model.focusedElement) {
		model.focusedElement = focusReleaseChooseCommitList
	}

	// Filter focus: the textinput owns most key events. Esc clears+blurs,
	// Enter blurs, Tab/Shift-Tab keep the global focus cycling so the
	// user can leave the filter without breaking the rotation.
	if model.focusedElement == focusReleaseChooseFilter {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
			case "esc":
				model.releaseChooseFilterBar.Reset()
				model.releaseChooseFilterBar.Blur()
				applyReleaseChooseModeFilter(model)
				model.focusedElement = focusReleaseChooseCommitList
				return model, nil
			case "enter":
				model.releaseChooseFilterBar.Blur()
				model.focusedElement = focusReleaseChooseCommitList
				return model, nil
			case "tab":
				return model, cycleReleaseChooseFocus(model, true)
			case "shift+tab":
				return model, cycleReleaseChooseFocus(model, false)
			}
		}
		prev := model.releaseChooseFilterBar.Value()
		var fcmd tea.Cmd
		model.releaseChooseFilterBar, fcmd = model.releaseChooseFilterBar.Update(msg)
		if model.releaseChooseFilterBar.Value() != prev {
			// Re-derive list filter state from both the typed value and
			// the segmented "All / Selected only" flag so the two never
			// fight each other.
			applyReleaseChooseModeFilter(model)
			// The cursor may have jumped to a freshly-filtered row, so
			// rebind the message + per-file cache to whatever is now
			// selected (no-op if it didn't actually move).
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
			model.loadReleaseCommitFiles()
		}
		return model, fcmd
	}

	switch model.focusedElement {
	case focusReleaseChooseCommitList:
		model.releaseCommitList, cmd = model.releaseCommitList.Update(msg)
	case focusReleaseChooseMsgVp:
		model.releaseViewport, cmd = model.releaseViewport.Update(msg)
	case focusReleaseChooseFileList:
		model.releaseFilesList, cmd = model.releaseFilesList.Update(msg)
	case focusReleaseChooseDiffVp:
		model.releaseDiffViewport, cmd = model.releaseDiffViewport.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ctrl+f cycles the filter mode pill regardless of focus, like
		// the main release view does. Re-apply the filter so the new
		// FilterValue (Subject/Hash/Type/Tag) is evaluated against the
		// existing items immediately — without it, the visible set
		// would only refresh on the next keystroke.
		if msg.String() == "ctrl+f" {
			model.releaseChooseFilterBar.CycleMode()
			applyReleaseChooseModeFilter(model)
			return model, nil
		}
		// ctrl+e is context-aware:
		//   - on the files list: swap between filename+dim-dir and the
		//     full relative path render mode.
		//   - elsewhere: swap the "All commits / Selected only"
		//     indicator on the top panel border.
		if key.Matches(msg, model.keys.SwapMode) {
			if model.focusedElement == focusReleaseChooseFileList {
				diffFileShowFullPath = !diffFileShowFullPath
				return model, nil
			}
			model.releaseChooseModeBar.Toggle()
			applyReleaseChooseModeFilter(model)
			return model, nil
		}
		switch {
		case key.Matches(msg, model.keys.Filter):
			model.focusedElement = focusReleaseChooseFilter
			return model, model.releaseChooseFilterBar.Focus()
		case key.Matches(msg, model.keys.NextField):
			return model, cycleReleaseChooseFocus(model, true)
		case key.Matches(msg, model.keys.PrevField):
			return model, cycleReleaseChooseFocus(model, false)
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
			// Enter kicks off the AI release builder. The mode-bar
			// toggle moved to ctrl+e (handled above) since the bar
			// is no longer a focus target.
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
			// SetItem returns a cmd that re-runs the filter in any state
			// other than Unfiltered, so when "Selected only" is active
			// the toggled row is added/removed from the visible set
			// automatically. Re-applying the filter here would call
			// SetFilterText/SetFilterState, both of which GoToStart()
			// and so reset the cursor to row 0 — exactly the bug the
			// user reported when adding a commit.
			cmd = model.releaseCommitList.SetItem(index, item)
			return model, cmd
		case key.Matches(msg, model.keys.Up, model.keys.Down):
			switch model.focusedElement {
			case focusReleaseChooseCommitList:
				if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
					model.commitLivePreview = item.Preview
				}
				model.loadReleaseCommitFiles()
			case focusReleaseChooseFileList:
				model.refreshReleaseDiffForSelectedFile()
			}
		case key.Matches(msg, model.keys.Esc):
			switch model.AppMode {
			case CommitMode:
				model.state = stateChoosingCommit
				model.keys = mainListKeys()
			case ReleaseMode:
				model.state = stateReleaseMainMenu
				model.keys = releaseMainListKeys()
				return model, syncReleaseHistorySelection(model)
			}
			return model, nil
		}
	}

	return model, cmd
}

// applyReleaseChooseModeFilter projects the segmented "All / Selected"
// toggle onto the workspace commit list by flipping the package-level
// `releaseChooseSelectedOnly` flag — `releaseChooseListFilter` and
// `WorkspaceCommitItem.FilterValue` both consult it. The list filter
// state is forced to Filtering whenever the user has typed a query OR
// the Selected-only mode is active, so the custom filter func always
// has a chance to run.
func applyReleaseChooseModeFilter(model *Model) {
	selectedOnly := model.releaseChooseModeBar.Mode() == ModeStagesResponse
	releaseChooseSelectedOnly = selectedOnly

	val := model.releaseChooseFilterBar.Value()
	if val == "" && !selectedOnly {
		// Nothing constrains the visible set — drop into Unfiltered so
		// the list serves all items and our custom Filter never runs.
		model.releaseCommitList.ResetFilter()
		return
	}
	filterText := val
	if filterText == "" {
		// list.filterItems short-circuits to "show all" when
		// FilterInput.Value() is empty, never invoking our Filter func.
		// Hand it a sentinel that releaseChooseListFilter recognises so
		// the Selected-only pass actually gets a chance to run.
		filterText = releaseChooseSentinel
	}
	// FilterApplied is the right state when "a filter is applied and
	// the user is not editing the filter": handleBrowsing routes the
	// up/down keys to the cursor, and the title bar stays clean.
	// Filtering would route everything into the bubble's internal
	// FilterInput and the list would stop responding to navigation.
	model.releaseCommitList.SetFilterText(filterText)
	model.releaseCommitList.SetFilterState(list.FilterApplied)
}
