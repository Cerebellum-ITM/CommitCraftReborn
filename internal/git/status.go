package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// StatusData captures the diff state of a git repository at a given moment:
// the absolute repo root, the changed files keyed by their repo-relative path,
// and every parent directory of those files. It's used by the file-picker UI
// so it can highlight modified entries without rerunning git per-frame.
type StatusData struct {
	Root                string
	FileStatus          map[string]string
	AffectedDirectories map[string]bool
}

// GetAllGitStatusData reads the staged diff plus the repo root and builds the
// directory closure used by the scope picker.
func GetAllGitStatusData() (StatusData, error) {
	var data StatusData
	data.FileStatus = make(map[string]string)
	data.AffectedDirectories = make(map[string]bool)

	gitRootBytes, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		data.Root = strings.TrimSpace(string(gitRootBytes))
	} else {
		return data, fmt.Errorf("could not determine git repository root: %w", err)
	}

	data.FileStatus, err = GetGitDiffNameStatus()
	if err != nil {
		data.FileStatus = make(map[string]string)
		return data, fmt.Errorf("could not get git diff name status: %w", err)
	}

	if data.Root != "" {
		populateAffectedDirs(&data)
	}

	return data, nil
}

// GetCommitGitStatusData builds a StatusData from the files changed in a
// specific commit, mirroring what GetAllGitStatusData does for staged
// changes. Used by the reword flow so the scope picker shows the files of
// the commit being reworded instead of the live workspace.
func GetCommitGitStatusData(hash string) (StatusData, error) {
	var data StatusData
	data.FileStatus = make(map[string]string)
	data.AffectedDirectories = make(map[string]bool)

	gitRootBytes, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return data, fmt.Errorf("could not determine git root: %w", err)
	}
	data.Root = strings.TrimSpace(string(gitRootBytes))

	output, err := exec.Command("git", "diff-tree", "--no-commit-id", "--name-status", "-r", hash).
		Output()
	if err != nil {
		return data, fmt.Errorf("could not get commit file statuses: %w", err)
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			data.FileStatus[parts[1]] = parts[0]
		}
	}

	populateAffectedDirs(&data)
	return data, nil
}

func populateAffectedDirs(data *StatusData) {
	for filePath := range data.FileStatus {
		dir := filepath.Dir(filePath)
		currentDir := dir
		for currentDir != "" && currentDir != "." && currentDir != "/" {
			data.AffectedDirectories[currentDir] = true
			currentDir = filepath.Dir(currentDir)
		}
		if dir == "." {
			data.AffectedDirectories["."] = true
		}
	}
}
