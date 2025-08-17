package tui

import (
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"os"
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
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
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
