package tui

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/statusbar"
)

func updateRewordSelectCommit(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	model.releaseCommitList, cmd = model.releaseCommitList.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Up, model.keys.Down):
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				model.commitLivePreview = item.Preview
			}
		case key.Matches(msg, model.keys.Enter):
			if item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem); ok {
				if model.commitAndReword {
					model.RewordHash = item.Hash
					model.syncRewordIndicator()
					model.commitAndReword = false
					model.usePreloadedDiff = true
					diffCode, err := git.GetCommitDiffSummary(item.Hash, model.globalConfig.Prompts.ChangeAnalyzerMaxDiffSize)
					if err != nil {
						model.log.Error("Error getting commit diff", "error", err)
					}
					model.diffCode = diffCode
					if commitGitData, gErr := git.GetCommitGitStatusData(item.Hash); gErr == nil {
						model.gitStatusData = commitGitData
						model.currentUpdateFileListFn(model.pwd, &model.fileList, model.gitStatusData)
						model.scopeDataStale = false
						model.syncScopeStaleIndicator()
					} else {
						model.log.Error("Error getting commit git status data", "error", gErr)
					}
					loadPipelineFilesFromDb(model, diffCode)
					model.currentCommit = storage.Commit{}
					model.keyPoints = nil
					model.commitTranslate = ""
					model.iaSummaryOutput = ""
					model.iaCommitRawOutput = ""
					model.iaTitleRawOutput = ""
					model.activeTab = 0
					model.activePipelineStage = 0
					model.state = stateWritingMessage
					model.focusedElement = focusComposeSummary
					model.keys = writingMessageKeys()
					model.WritingStatusBar.Content = fmt.Sprintf("Reword %s · compose key points and run AI", item.Hash[:7])
					model.WritingStatusBar.Level = statusbar.LevelInfo
					if cmd := model.commitsKeysInput.Focus(); cmd != nil {
						return model, cmd
					}
					return model, nil
				}
				model.RewordHash = item.Hash
				model.FinalMessage = outputCommitMessageOrFallback(model, model.currentCommit)
				return quitWithAutodraft(model)
			}
		case key.Matches(msg, model.keys.Esc):
			model.state = stateChoosingCommit
			model.keys = mainListKeys()
			model.WritingStatusBar.Content = fmt.Sprintf(
				"choose, create, or edit a commit ::: %s",
				model.Theme.AppStyles().
					Base.Foreground(model.Theme.Tertiary).
					SetString(model.mainList.Title),
			)
			model.WritingStatusBar.Level = statusbar.LevelInfo
			return model, nil
		}
	}

	return model, cmd
}

// openVersionEditor builds the release-version editor popup with the latest
// git tag as the bump base. Used both by the Ctrl+V global shortcut and as a
// pre-flight step before publishing a GitHub release, so the user always
// reviews the tag the upload is about to use.

func setupCommitReword(model *Model) (tea.Model, tea.Cmd) {
	hash := model.pendingRewordHash
	model.pendingRewordHash = ""
	model.RewordHash = hash
	model.syncRewordIndicator()
	model.usePreloadedDiff = true

	diff, dErr := git.GetCommitDiffSummary(
		hash,
		model.globalConfig.Prompts.ChangeAnalyzerMaxDiffSize,
	)
	if dErr != nil {
		model.log.Error("Error getting commit diff for reword", "error", dErr)
	}
	model.diffCode = diff

	if commitGitData, gErr := git.GetCommitGitStatusData(hash); gErr == nil {
		model.gitStatusData = commitGitData
		model.currentUpdateFileListFn(model.pwd, &model.fileList, model.gitStatusData)
		model.scopeDataStale = false
		model.syncScopeStaleIndicator()
	} else {
		model.log.Error("Error getting commit git status data for reword", "error", gErr)
	}

	loadPipelineFilesFromDb(model, diff)
	model.currentCommit = storage.Commit{}
	model.keyPoints = nil
	model.commitTranslate = ""
	model.iaSummaryOutput = ""
	model.iaCommitRawOutput = ""
	model.iaTitleRawOutput = ""
	model.activeTab = 0
	model.activePipelineStage = 0

	model.state = stateWritingMessage
	model.focusedElement = focusComposeSummary
	model.keys = writingMessageKeys()
	model.WritingStatusBar.Content = fmt.Sprintf(
		"Reword %s · compose key points and run AI",
		hash[:7],
	)
	model.WritingStatusBar.Level = statusbar.LevelInfo
	if cmd := model.commitsKeysInput.Focus(); cmd != nil {
		return model, cmd
	}
	return model, nil
}

// setupReleaseReword routes the user from the -w startup popup into the
// regular release creation flow (-r) while preserving the original commit
// hash so the eventual createRelease call can rewrite that commit's
// message with the release-formatted output. RewordHash itself stays
// empty until createRelease moves the hash back into it next to the
// composed FinalMessage — otherwise an early quit (Esc, ^X) would fire
// the post-TUI reword hook against an empty message.
func setupReleaseReword(model *Model) (tea.Model, tea.Cmd) {
	model.releaseRewordHash = model.pendingRewordHash
	model.pendingRewordHash = ""
	model.RewordHash = ""
	model.syncRewordIndicator()

	model.AppMode = ReleaseMode
	model.state = stateReleaseChoosingCommits
	model.topTab = TabCompose
	model.keys = releaseKeys()
	cmd := model.initFreshReleaseCompose()
	if model.releaseRewordHash != "" {
		short := model.releaseRewordHash
		if len(short) > 7 {
			short = short[:7]
		}
		model.WritingStatusBar.Content = fmt.Sprintf(
			"Reword %s as release/merge · pick commits to compose the message",
			short,
		)
		model.WritingStatusBar.Level = statusbar.LevelInfo
	}
	return model, cmd
}

// setupReleaseFromDbChooser handles the third item of the `-w` startup
// chooser ("Rewrite using existing release"). It loads the workspace's
// release rows, caches them on the model so the dispatcher can resolve
// the picked entry, and opens a second list popup with one row per
// release (RELDB#<id> sentinel + a human-readable summary). The
// pendingRewordHash is preserved through the popup interaction; the
// reword fires from finalizeReleaseFromDbPick once the user confirms a
// row.
func setupReleaseFromDbChooser(model *Model) (tea.Model, tea.Cmd) {
	releases, err := model.db.GetReleases(model.pwd)
	if err != nil {
		model.log.Error("could not load releases for reword chooser", "error", err)
		model.WritingStatusBar.Content = fmt.Sprintf("Could not load releases: %s", err)
		model.WritingStatusBar.Level = statusbar.LevelError
		return model, nil
	}
	if len(releases) == 0 {
		model.WritingStatusBar.Content = "No release entries in this workspace · pick a different option"
		model.WritingStatusBar.Level = statusbar.LevelWarning
		// Re-open the original chooser so the user can pick a different
		// strategy without restarting the TUI. pendingRewordHash is
		// still set, so the chooser knows which commit it is reworking.
		return model, openRewordChooserCmd(model)
	}

	model.pendingDbReleases = releases
	items := make([]string, 0, len(releases))
	options := make([]itemsOptions, 0, len(releases))
	syms := model.Theme.AppSymbols()
	for i, r := range releases {
		date := r.CreatedAt.Format("2006-01-02")
		title := strings.TrimSpace(r.Title)
		if title == "" {
			title = "(no title)"
		}
		entry := fmt.Sprintf(
			"%s%d · %s [%s] %s · %s",
			dbReleasePickPrefix, r.ID, date, r.Type, r.Branch, title,
		)
		items = append(items, entry)
		options = append(options, itemsOptions{
			index: i,
			color: model.Theme.Secondary,
			icon:  syms.RewordChooserDb,
		})
	}

	w := model.width * 3 / 4
	if w < 60 {
		w = 70
	}
	h := model.height * 3 / 4
	if h < 12 {
		h = 14
	}
	short := model.pendingRewordHash
	if len(short) > 7 {
		short = short[:7]
	}
	return model, func() tea.Msg {
		return openListPopup{
			title:        fmt.Sprintf("Reword %s · pick a release", short),
			color:        model.Theme.Primary,
			items:        items,
			itemsOptions: options,
			width:        w,
			height:       h,
		}
	}
}

// finalizeReleaseFromDbPick is the dispatcher hook for selections coming
// out of the second popup of the "Rewrite using existing release" flow.
// It parses the RELDB#<id> sentinel, resolves the row from the cached
// snapshot, composes the same `[TYPE] <branch>: <title>\n\n<body>` shape
// the live release pipeline produces, copies the original hash into
// RewordHash, sets FinalMessage, and quits so main.go's post-TUI hook
// runs git.RewordCommit.
func finalizeReleaseFromDbPick(model *Model, action string) (tea.Model, tea.Cmd) {
	idStr := strings.SplitN(strings.TrimPrefix(action, dbReleasePickPrefix), " ", 2)[0]
	idStr = strings.TrimSuffix(idStr, "·")
	idStr = strings.TrimSpace(idStr)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		model.log.Error("could not parse release id from pick", "raw", action, "error", err)
		model.WritingStatusBar.Content = "Could not resolve the picked release"
		model.WritingStatusBar.Level = statusbar.LevelError
		return model, nil
	}
	var picked *storage.Release
	for i := range model.pendingDbReleases {
		if model.pendingDbReleases[i].ID == id {
			picked = &model.pendingDbReleases[i]
			break
		}
	}
	if picked == nil {
		model.log.Error("picked release id not found in cache", "id", id)
		model.WritingStatusBar.Content = "Picked release vanished from cache"
		model.WritingStatusBar.Level = statusbar.LevelError
		return model, nil
	}

	formattedType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, picked.Type)
	final := fmt.Sprintf("%s %s: %s", formattedType, picked.Branch, strings.TrimSpace(picked.Title))
	body := strings.TrimSpace(picked.Body)
	if body != "" {
		final = final + "\n\n" + body
	}

	model.RewordHash = model.pendingRewordHash
	model.pendingRewordHash = ""
	model.pendingDbReleases = nil
	model.syncRewordIndicator()
	model.FinalMessage = final
	return quitWithAutodraft(model)
}
