package tui

import (
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

func calculatePopupPosition(modelWidth, modelHeight int, popupView string) (startX, startY int) {
	popupWidth := lipgloss.Width(popupView)
	popupHeight := lipgloss.Height(popupView)
	startX = (modelWidth - popupWidth) / 2
	startY = (modelHeight - popupHeight) / 2
	return startX, startY
}

func UpdateCommitList(db *storage.DB, log *logger.Logger, l *list.Model) error {
	workspaceCommits, err := db.GetCommits()
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
