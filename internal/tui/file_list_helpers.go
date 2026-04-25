package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"

	"charm.land/bubbles/v2/list"
)

// UpdateFileListFunc is the shape of a function that repopulates a
// bubbles list with file entries from the current workspace, possibly
// filtered. ChooseUpdateFileListFunction picks the right implementation
// for the user's filter setting.
type UpdateFileListFunc func(pwd string, l *list.Model, gitData git.StatusData) error

// UpdateCommitList reloads the commit/release picker from the SQLite
// store so the UI reflects new entries created in this session.
func UpdateCommitList(
	pwd string,
	db *storage.DB,
	log *logger.Logger,
	l *list.Model,
	action CommitCraftTables,
) error {
	var items []list.Item

	switch action {
	case commitDb:
		workspaceCommits, err := db.GetCommits(pwd, "completed")
		if err != nil {
			log.Error("Error reloading the list of commits", "error", err)
			return err
		}

		items = make([]list.Item, len(workspaceCommits))
		for i, c := range workspaceCommits {
			items[i] = HistoryCommitItem{commit: c}
		}
	case releaseDb:
		workspaceReleases, err := db.GetReleases(pwd)
		if err != nil {
			log.Error("Error reloading the list of releases", "error", err)
			return err
		}
		items = make([]list.Item, len(workspaceReleases))
		for i, r := range workspaceReleases {
			items[i] = HistoryReleaseItem{release: r}
		}
	}

	l.SetItems(items)

	return nil
}

// CreateFileItemsList reads pwd and joins each entry with its git status
// (and "M" for directories that contain modified files) so the file picker
// can highlight changed paths.
func CreateFileItemsList(pwd string, gitData git.StatusData) ([]list.Item, error) {
	dirEntries, err := os.ReadDir(pwd)
	if err != nil {
		return make([]list.Item, 0), err
	}

	items := make([]list.Item, len(dirEntries))
	for i, entry := range dirEntries {
		status := ""
		var fullPathRelativeToGitRoot string

		absPath, err := filepath.Abs(filepath.Join(pwd, entry.Name()))
		if err == nil {
			if gitData.Root != "" {
				fullPathRelativeToGitRoot, err = filepath.Rel(
					gitData.Root,
					absPath,
				)
				if err != nil {
					fullPathRelativeToGitRoot = absPath
				}
			} else {
				fullPathRelativeToGitRoot = absPath
			}
		} else {
			fullPathRelativeToGitRoot = filepath.Join(pwd, entry.Name())
		}

		if s, ok := gitData.FileStatus[fullPathRelativeToGitRoot]; ok {
			status = s
		} else if entry.IsDir() {
			if _, ok := gitData.AffectedDirectories[fullPathRelativeToGitRoot]; ok {
				status = "M"
			}
		}

		items[i] = FileItem{Entry: entry, Status: status}
	}
	return items, nil
}

// UpdateFileListWithFilterItems repopulates the picker with only the entries
// that have a non-empty git status (modified, added, deleted…).
func UpdateFileListWithFilterItems(pwd string, l *list.Model, gitData git.StatusData) error {
	items, err := CreateFileItemsList(pwd, gitData)
	if err != nil {
		return fmt.Errorf("Error changing list items %s", err)
	}

	var filteredItems []list.Item
	for _, item := range items {
		if fileItem, ok := item.(FileItem); ok {
			if fileItem.Status != "" {
				filteredItems = append(filteredItems, fileItem)
			}
		}
	}
	l.SetItems(filteredItems)
	return nil
}

// UpdateFileList repopulates the picker with every directory entry,
// regardless of git status.
func UpdateFileList(pwd string, l *list.Model, gitData git.StatusData) error {
	items, err := CreateFileItemsList(pwd, gitData)
	if err != nil {
		return fmt.Errorf("Error changing list items %s", err)
	}

	l.SetItems(items)
	return nil
}

// ChooseUpdateFileListFunction returns the filter-aware variant when the
// user has the "modified only" toggle on, otherwise the unfiltered one.
func ChooseUpdateFileListFunction(showOnlyModified bool) UpdateFileListFunc {
	if showOnlyModified {
		return UpdateFileListWithFilterItems
	}
	return UpdateFileList
}
