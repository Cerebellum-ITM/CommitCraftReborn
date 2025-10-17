package tui

import (
	"fmt"
	"image/color"
	"io"
	"time"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

// HistoryCommitItem
type HistoryCommitItem struct {
	commit storage.Commit
}

func (hci HistoryCommitItem) Title() string {
	return ""
}

func (hci HistoryCommitItem) Description() string {
	return ""
}

func (hci HistoryCommitItem) FilterValue() string {
	return hci.commit.MessageEN + " " + hci.commit.MessageES + " " +
		hci.commit.Type + " " + hci.commit.Scope
}

type HistoryCommitDelegate struct {
	list.DefaultDelegate

	commitTypeStyle          lipgloss.Style
	selectedContainerStyle   lipgloss.Style
	unselectedContainerStyle lipgloss.Style
	Theme                    *styles.Theme
	globalConfig             config.Config
	infoText                 lipgloss.Style

	commitFormat       string
	dateStyle          lipgloss.Style
	idStyle            lipgloss.Style
	msgOriginalStyle   lipgloss.Style
	msgTranslatedStyle lipgloss.Style
	finalMsgStyle      lipgloss.Style
}

func NewHistoryCommitDelegate(globalConfig config.Config, theme *styles.Theme) list.ItemDelegate {
	baseStyle := theme.AppStyles().Base
	return HistoryCommitDelegate{
		globalConfig: globalConfig,
		commitFormat: globalConfig.CommitFormat.TypeFormat,
		Theme:        theme,
		selectedContainerStyle: baseStyle.
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(theme.Accent).
			PaddingLeft(1),

		unselectedContainerStyle: baseStyle.
			PaddingLeft(2),

		commitTypeStyle: baseStyle.
			Foreground(theme.Blur),
		dateStyle: baseStyle.
			Foreground(theme.Blur),

		idStyle: baseStyle.
			Foreground(theme.Blur),

		msgOriginalStyle: baseStyle.Foreground(theme.Blur),

		msgTranslatedStyle: baseStyle.Foreground(theme.Blur),
		finalMsgStyle:      baseStyle.Foreground(theme.Blur),
		infoText:           baseStyle.Foreground(theme.Blur),
	}
}

func (d HistoryCommitDelegate) GetCommitTypeColor(commitType string) color.Color {
	colorStr, ok := d.globalConfig.CommitFormat.CommitTypeColors[commitType]
	if !ok || colorStr == "" {
		// Fallback color if not found
		return d.Theme.White
	}
	return lipgloss.Color(colorStr)
}

func (d HistoryCommitDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(HistoryCommitItem)
	if !ok {
		return
	}

	commit := it.commit
	tagStr := fmt.Sprintf("%s", commit.Type)

	dateStr := commit.CreatedAt.Format("2006-01-02 15:04")
	idStr := fmt.Sprintf("ID: %d", commit.ID)
	originalMsg := commit.MessageES
	translatedMsg := commit.MessageEN
	formattedCommitType := fmt.Sprintf(d.commitFormat, commit.Type)
	finalStr := fmt.Sprintf("%s %s: %s", formattedCommitType, commit.Scope, commit.MessageEN)

	var (
		indicator                 = " "
		indicatorStyle            lipgloss.Style
		currentCommitTypeStyle    lipgloss.Style
		itemDisplayStyle          lipgloss.Style
		currentDateStyle          lipgloss.Style
		currentIDStyle            lipgloss.Style
		currentMsgOriginalStyle   lipgloss.Style
		currentMsgTranslatedStyle lipgloss.Style
		currentFinalMsgStyle      lipgloss.Style
		currentInfoText           lipgloss.Style
		adicionalUiText           lipgloss.Style
	)

	indicatorStyle = lipgloss.NewStyle().Foreground(d.Theme.Accent)

	if index == m.Index() {
		indicator = "❯"
		currentCommitTypeStyle = d.commitTypeStyle.Foreground(d.GetCommitTypeColor(commit.Type))
		itemDisplayStyle = d.selectedContainerStyle
		currentDateStyle = d.dateStyle.Bold(true).
			Foreground(d.Theme.Accent)
		currentIDStyle = d.idStyle.Bold(true).
			Foreground(d.Theme.FillTextLine)
		currentMsgOriginalStyle = d.msgOriginalStyle.Foreground(d.Theme.Input)
		currentMsgTranslatedStyle = d.msgTranslatedStyle.Italic(true).
			Foreground(d.Theme.Output)
		currentFinalMsgStyle = d.finalMsgStyle.Foreground(d.GetCommitTypeColor(commit.Type))
		currentInfoText = d.infoText.Foreground(d.Theme.FgSubtle)
		adicionalUiText = d.Theme.AppStyles().Base.Foreground(d.Theme.White)
	} else {
		itemDisplayStyle = d.unselectedContainerStyle
		currentDateStyle = d.dateStyle
		currentIDStyle = d.idStyle
		currentMsgOriginalStyle = d.msgOriginalStyle
		currentMsgTranslatedStyle = d.msgTranslatedStyle
		currentCommitTypeStyle = d.commitTypeStyle
		currentFinalMsgStyle = d.finalMsgStyle
		currentInfoText = d.infoText
		adicionalUiText = d.Theme.AppStyles().Base.Foreground(d.Theme.Blur)
	}
	contentAvailableWidth := m.Width() - itemDisplayStyle.GetHorizontalPadding() - itemDisplayStyle.
		GetHorizontalBorderSize() - lipgloss.Width(indicator+" ")

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
			currentCommitTypeStyle.SetString("Type: ").String(),
			currentCommitTypeStyle.Render(tagStr),
			closeBracket,
		),
	)

	msgPrefixWidth := 7
	maxMsgLength := contentAvailableWidth - msgPrefixWidth
	maxMsgLength = max(maxMsgLength, 10)

	truncatedOriginalMsg := TruncateString(originalMsg, maxMsgLength)
	truncatedTranslatedMsg := TruncateString(translatedMsg, maxMsgLength)
	truncatedFinalStr := TruncateString(finalStr, maxMsgLength)

	line1 := fmt.Sprintf("%s %s", indicatorStyle.Render(indicator), line1Content)
	line2 := fmt.Sprintf(
		"  %s %s",
		currentInfoText.Render("Inp:"),
		currentMsgOriginalStyle.Render(truncatedOriginalMsg),
	)
	line3 := fmt.Sprintf(
		"  %s %s",
		currentInfoText.Render("Out:"),
		currentMsgTranslatedStyle.Render(truncatedTranslatedMsg),
	)
	line4 := fmt.Sprintf(
		"  %s %s",
		currentInfoText.Render("Fnl:"),
		currentFinalMsgStyle.Render(truncatedFinalStr),
	)

	finalRender := lipgloss.JoinVertical(lipgloss.Left,
		line1,
		line2,
		line3,
		line4,
	)

	fmt.Fprint(w, itemDisplayStyle.Width(m.Width()).Render(finalRender))
}

func (d HistoryCommitDelegate) Height() int  { return 3 }
func (d HistoryCommitDelegate) Spacing() int { return 1 }

func NewHistoryCommitList(
	workspaceCommits []storage.Commit,
	pwd string,
	globalConfig config.Config,
	theme *styles.Theme,
) list.Model {
	items := make([]list.Item, len(workspaceCommits))
	for i, c := range workspaceCommits {
		items[i] = HistoryCommitItem{commit: c}
	}

	historyList := list.New(items, NewHistoryCommitDelegate(globalConfig, theme), 0, 0)
	historyList.Title = fmt.Sprintf("%s: %s", "Working directory", TruncatePath(pwd, 2))
	historyList.SetShowTitle(false)
	historyList.SetShowFilter(true)
	historyList.SetShowHelp(false)
	historyList.SetStatusBarItemName("commit", "commits")
	historyList.Styles.StatusBar = historyList.Styles.StatusBar.Foreground(theme.Accent)
	historyList.SetFilteringEnabled(true)
	historyList.StatusMessageLifetime = 5 * time.Second
	return historyList
}
