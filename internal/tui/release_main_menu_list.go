package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

// HistoryReleaseItem
type HistoryReleaseItem struct {
	release storage.Release
}

func (hci HistoryReleaseItem) Title() string {
	return ""
}

func (hci HistoryReleaseItem) Description() string {
	return ""
}

func (hci HistoryReleaseItem) FilterValue() string {
	return hci.release.Title + " " + hci.release.Body + " " + hci.release.Branch
}

type HistoryReleaseDelegate struct {
	list.DefaultDelegate

	releaseFormat string

	Theme        *styles.Theme
	globalConfig config.Config

	idStyle                  lipgloss.Style
	infoText                 lipgloss.Style
	dateStyle                lipgloss.Style
	hashStyle                lipgloss.Style
	indicatorStyle           lipgloss.Style
	adicionalUiText          lipgloss.Style
	releaseTypeStyle         lipgloss.Style
	selectedContainerStyle   lipgloss.Style
	unselectedContainerStyle lipgloss.Style
}

func NewHistoryReleaseDelegate(globalConfig config.Config, theme *styles.Theme) list.ItemDelegate {
	base := theme.AppStyles().Base
	baseFg := theme.FgMuted
	baseState := base.Foreground(baseFg)
	return HistoryReleaseDelegate{
		globalConfig:  globalConfig,
		releaseFormat: globalConfig.CommitFormat.TypeFormat,
		Theme:         theme,
		selectedContainerStyle: baseState.
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(theme.Accent).
			PaddingLeft(1),
		unselectedContainerStyle: baseState.
			PaddingLeft(2),
		releaseTypeStyle: baseState,
		dateStyle:        baseState,
		idStyle:          baseState,
		indicatorStyle:   theme.AppStyles().IndicatorStyle,
		infoText:         baseState,
		adicionalUiText:  baseState,
		hashStyle:        baseState,
	}
}

func (d HistoryReleaseDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(HistoryReleaseItem)
	if !ok {
		return
	}

	var (
		indicatorStr            string
		hashLitsStr             []string
		currentInfoText         lipgloss.Style
		indicatorStyle          lipgloss.Style
		adicionalUiText         lipgloss.Style
		currentDateStyle        lipgloss.Style
		currentIDStyle          lipgloss.Style
		currentReleaseTypeStyle lipgloss.Style
		currentHashStyle        lipgloss.Style
		itemDisplayStyle        lipgloss.Style
	)

	hashLength := 11
	release := it.release
	indicatorStyle = d.indicatorStyle
	dateStr := release.CreatedAt.Format("2006-01-02 15:04")
	idStr := fmt.Sprintf("ID: %d", release.ID)
	hashList := strings.Split(release.CommitList, ",")
	formattedReleaseType := fmt.Sprintf(d.releaseFormat, release.Type)
	finalStr := fmt.Sprintf("%s %s: %s", formattedReleaseType, release.Branch, release.Title)

	for _, hash := range hashList {
		hashLitsStr = append(hashLitsStr, TruncateString(hash, hashLength))
	}

	if index == m.Index() {
		indicatorStr = "❯"
		adicionalUiText = d.adicionalUiText.Foreground(d.Theme.White)
		currentReleaseTypeStyle = d.releaseTypeStyle.Foreground(d.Theme.Success)
		currentInfoText = d.infoText.Foreground(d.Theme.FgBase)
		currentHashStyle = d.hashStyle.Foreground(d.Theme.Yellow)
		itemDisplayStyle = d.selectedContainerStyle
		currentIDStyle = d.idStyle.Foreground(d.Theme.FillTextLine)
		currentDateStyle = d.dateStyle.Foreground(d.Theme.Accent)

	} else {
		indicatorStr = " "
		adicionalUiText = d.adicionalUiText
		currentDateStyle = d.dateStyle
		currentIDStyle = d.idStyle
		currentInfoText = d.infoText
		currentHashStyle = d.hashStyle
		itemDisplayStyle = d.unselectedContainerStyle
		currentReleaseTypeStyle = d.releaseTypeStyle
	}

	openBracket := adicionalUiText.SetString("(").String()
	closeBracket := adicionalUiText.SetString(")").String()
	line1Content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		currentDateStyle.Render(dateStr),
		" ",
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			openBracket,
			currentIDStyle.Render(idStr),
			closeBracket,
		),
		" ",
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			openBracket,
			currentReleaseTypeStyle.SetString("Type: ").String(),
			currentReleaseTypeStyle.Render(release.Type),
			closeBracket,
		),
	)

	contentAvailableWidth := m.Width() - itemDisplayStyle.GetHorizontalPadding() - itemDisplayStyle.
		GetHorizontalBorderSize() - lipgloss.Width(
		indicatorStr+" ",
	)

	msgPrefixWidth := 7
	maxMsgLength := contentAvailableWidth - msgPrefixWidth
	maxMsgLength = max(10, maxMsgLength)
	truncatedHashes := TruncateString(strings.Join(hashLitsStr, ","), maxMsgLength)

	line1 := fmt.Sprintf("%s %s", indicatorStyle.Render(indicatorStr), line1Content)
	line2 := fmt.Sprintf(
		"  %s %s",
		currentInfoText.Render("List:"),
		currentHashStyle.Render(fmt.Sprintf("[%s]", truncatedHashes)),
	)
	line3 := fmt.Sprintf(
		"  %s %s",
		currentInfoText.Render("Titl:"),
		indicatorStyle.Render(finalStr),
	)
	// line4 := fmt.Sprintf(
	// "  %s %s",
	// currentInfoText.Render("Body:"),
	// indicatorStyle.Render(release.Body),
	// )

	finalRender := lipgloss.JoinVertical(lipgloss.Left,
		line1,
		line2,
		line3,
		// line4,
	)

	fmt.Fprint(w, itemDisplayStyle.Width(m.Width()).Render(finalRender))
}

func (d HistoryReleaseDelegate) Height() int  { return 3 }
func (d HistoryReleaseDelegate) Spacing() int { return 1 }

func NewHistoryReleaseList(
	workspaceReleases []storage.Release,
	pwd string,
	globalConfig config.Config,
	theme *styles.Theme,
) list.Model {
	items := make([]list.Item, len(workspaceReleases))
	for i, c := range workspaceReleases {
		items[i] = HistoryReleaseItem{release: c}
	}

	historyList := list.New(items, NewHistoryReleaseDelegate(globalConfig, theme), 0, 0)
	historyList.Title = fmt.Sprintf("%s: %s", "Working directory", TruncatePath(pwd, 2))
	historyList.SetShowTitle(false)
	historyList.SetShowFilter(true)
	historyList.SetShowHelp(false)
	historyList.SetStatusBarItemName("release", "releases")
	historyList.Styles.StatusBar = historyList.Styles.StatusBar.Foreground(theme.Accent)
	historyList.SetFilteringEnabled(true)
	historyList.StatusMessageLifetime = 5 * time.Second
	return historyList
}
