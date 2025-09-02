package tui

import (
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

// ---------------------------------------------------------
// HELPERS
// ---------------------------------------------------------
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

func UpdateFileList(pwd string, l *list.Model) error {
	dirEntries, err := os.ReadDir(pwd)
	if err != nil {
		return err
	}
	items := make([]list.Item, len(dirEntries))
	for i, entry := range dirEntries {
		items[i] = FileItem{Entry: entry}
	}
	l.SetItems(items)
	l.Title = fmt.Sprintf("Select a file or directory in %s", TruncatePath(pwd, 2))
	return nil
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
		return "  yaml"
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
