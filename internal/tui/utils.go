package tui

import (
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

type FilterState int

const (
	Unfiltered    FilterState = iota // no filter set
	Filtering                        // user is actively setting a filter
	FilterApplied                    // a filter is applied and user is not editing filter
)

type GitStatusData struct {
	Root                string
	FileStatus          map[string]string // Key: full path relative to Git root, Value: "A", "M", "D", etc.
	AffectedDirectories map[string]bool   // Key: full path relative to Git root, Value: true if directory
}

type UpdateFileListFunc func(pwd string, l *list.Model, gitData GitStatusData) error

// ---------------------------------------------------------
// HELPERS
// ---------------------------------------------------------

func ResetAndActiveFilterOnList(l *list.Model) {
	if l != nil {
		l.ResetFilter()
		l.SetFilterText("")
		l.SetFilterState(list.FilterState(Filtering))
	}
}

func GetAllGitStatusData() (GitStatusData, error) {
	var data GitStatusData
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

	return data, nil
}

// GetStagedDiffSummary generates a string containing the diffs of all staged files.
func GetStagedDiffSummary(maxDiffChars int) (string, error) {
	// 1. Get the list of staged file names.
	cmdFiles := exec.Command("git", "diff", "--cached", "--name-only")
	stagedFilesBytes, err := cmdFiles.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to get staged files: %w", err)
	}

	if len(stagedFilesBytes) == 0 {
		return "", nil
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(stagedFilesBytes)), "\n")

	var resultBuilder strings.Builder
	currentChars := 0

	// 2. Loop through each file and get its specific diff, with truncation logic.
	for _, file := range stagedFiles {
		if file == "" {
			continue
		}

		cmdDiff := exec.Command("git", "diff", "--cached", "--unified=0", "--", file)
		diffBytes, err := cmdDiff.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get diff for file %s: %w", file, err)
		}

		block := fmt.Sprintf("=== %s ===\n%s\n", file, string(diffBytes))
		blockChars := len(
			block,
		)

		if currentChars+blockChars > maxDiffChars {
			break
		}

		resultBuilder.WriteString(block)
		currentChars += blockChars
	}

	return resultBuilder.String(), nil
}

func GetGitDiffNameStatus() (map[string]string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--name-status")
	outputBytes, err := cmd.Output()
	if err != nil {
		// If there are no staged changes, git diff --name-status returns an empty string
		// and a non-zero exit code. We should treat this as no changes, not an error.
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to get git diff --name-status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(outputBytes)), "\n")
	statusMap := make(map[string]string)

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			status := parts[0]
			filePath := parts[1]
			statusMap[filePath] = status
		}
	}

	return statusMap, nil
}

func calculatePopupPosition(modelWidth, modelHeight int, popupView string) (startX, startY int) {
	popupWidth := lipgloss.Width(popupView)
	popupHeight := lipgloss.Height(popupView)
	startX = (modelWidth - popupWidth) / 2
	startY = (modelHeight - popupHeight) / 2
	return startX, startY
}

func UpdateCommitList(pwd string, db *storage.DB, log *logger.Logger, l *list.Model) error {
	workspaceCommits, err := db.GetCommits(pwd)
	if err != nil {
		log.Error("Error al recargar commits", "error", err)
		return err
	}

	items := make([]list.Item, len(workspaceCommits))
	for i, c := range workspaceCommits {
		items[i] = HistoryCommitItem{commit: c}
	}
	l.SetItems(items)

	return nil
}

func CreateFileItemsList(pwd string, gitData GitStatusData) ([]list.Item, error) {
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

func UpdateFileListWithFilterItems(pwd string, l *list.Model, gitData GitStatusData) error {
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

func UpdateFileList(pwd string, l *list.Model, gitData GitStatusData) error {
	items, err := CreateFileItemsList(pwd, gitData)
	if err != nil {
		return fmt.Errorf("Error changing list items %s", err)
	}

	l.SetItems(items)
	return nil
}

func ChooseUpdateFileListFunction(showOnlyModified bool) UpdateFileListFunc {
	if showOnlyModified {
		return UpdateFileListWithFilterItems
	}
	return UpdateFileList
}

func TruncatePath(path string, levels int) string {
	if levels <= 0 || path == "" {
		return ""
	}

	parts := strings.Split(path, string(os.PathSeparator))
	filteredParts := []string{}
	for _, part := range parts {
		if part != "" {
			filteredParts = append(filteredParts, part)
		}
	}

	if len(filteredParts) <= levels {
		return path
	}

	startIndex := len(filteredParts) - levels
	truncatedParts := filteredParts[startIndex:]

	prefix := ""
	if startIndex > 0 {
		prefix = "..." + string(os.PathSeparator)
	}

	return prefix + strings.Join(truncatedParts, string(os.PathSeparator))
}

func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	firstLine := s
	if newlineIndex := strings.Index(s, "\n"); newlineIndex != -1 {
		firstLine = s[:newlineIndex]
	}

	runes := []rune(firstLine)
	if len(runes) <= maxLen {
		return firstLine
	}

	if maxLen < 3 {
		return string(runes[:maxLen])
	}

	return string(runes[:maxLen-3]) + "..."
}

func TruncateMessageLines(message string, width int) string {
	lines := strings.Split(message, "\n")
	var formattedLines []string

	for _, line := range lines {
		if line == "" {
			formattedLines = append(formattedLines, "")
			continue
		}
		formattedLines = append(formattedLines, TruncateString(line, width))
	}

	return strings.Join(formattedLines, "\n")
}

func GetNerdFontIcon(filename string, isDir bool) string {
	if isDir {
		return ""
	}

	extension := strings.ToLower(filepath.Ext(filename))
	name := strings.ToLower(filename)

	switch extension {
	// Programming languages
	case ".go":
		return "󰟓"
	case ".py":
		return ""
	case ".js":
		return "󰌞"
	case ".ts":
		return "󰛦"
	case ".java":
		return ""
	case ".cs":
		return "󰌛"
	case ".rs":
		return ""
	case ".c":
		return "󰙱"
	case ".cpp", ".h":
		return "󰙲"

		// Configuration and data files
	case ".json":
		return "󰘦"
	case ".yml", ".yaml":
		return ""
	case ".xml":
		return "󰗀"
	case ".toml":
		return ""
	case ".env":
		return ""

	// Documentation
	case ".md", ".mdx":
		return ""

	// Git
	case ".git":
		return " Git"

	// Media
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return ""

		// Compressed files
	case ".zip", ".tar", ".gz", ".rar":
		return "󰿺"

	default:
		switch name {
		// Special cases by full file name
		case "docker-compose.yml", "dockerfile":
			return "󰡨"
		case "makefile":
			return ""
		case "readme.md":
			return "󰂺"
		case "license":
			return "󰿃"
		case ".gitignore":
			return ""
		}

		// Default icon for any other file
		return ""
	}
}
