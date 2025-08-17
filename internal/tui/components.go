package tui

import (
	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/storage"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/v2/list"
	"github.com/charmbracelet/lipgloss/v2"
)

type CommitTypeDelegate struct {
	list.DefaultDelegate
	TypeFormat string
	Color      string
}
type CommitTypeItem struct {
	commit.CommitType
}

func (cti CommitTypeItem) Title() string { return cti.CommitType.Tag }
func (cti CommitTypeItem) Color() string { return cti.CommitType.Color }

func (cti CommitTypeItem) Description() string { return cti.CommitType.Description }

func (cti CommitTypeItem) FilterValue() string {
	return cti.CommitType.Tag + " " + cti.CommitType.Description
}

func (d CommitTypeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(CommitTypeItem)
	if !ok {
		return
	}

	commitType := it.Title()
	commitDesc := it.Description()
	commitColor := it.Color()
	formattedCommitType := fmt.Sprintf(d.TypeFormat, commitType)

	var renderedType, renderedDesc string

	if index == m.Index() {
		styleType := lipgloss.NewStyle().
			Foreground(lipgloss.Color(commitColor)).
			Bold(true)

		styleDesc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")) // Amarillo claro

		renderedType = styleType.Render(formattedCommitType)
		renderedDesc = styleDesc.Render(commitDesc)

		fmt.Fprintf(w, "❯ %s - %s", renderedType, renderedDesc)
	} else {
		styleType := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")) // Gris

		styleDesc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")) // Gris más oscuro

		renderedType = styleType.Render(formattedCommitType)
		renderedDesc = styleDesc.Render(commitDesc)

		fmt.Fprintf(w, "  %s - %s", renderedType, renderedDesc)
	}
}

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
		hci.commit.CreatedAt.Format("2006-01-02") + " " +
		fmt.Sprintf("%d", hci.commit.ID) + " " +
		hci.commit.Type + " " + hci.commit.Scope
}

type HistoryCommitDelegate struct {
	list.DefaultDelegate

	commitTypeStyle          lipgloss.Style
	selectedContainerStyle   lipgloss.Style
	unselectedContainerStyle lipgloss.Style

	dateStyle          lipgloss.Style
	idStyle            lipgloss.Style
	msgOriginalStyle   lipgloss.Style
	msgTranslatedStyle lipgloss.Style
}

func NewHistoryCommitDelegate() list.ItemDelegate {
	return HistoryCommitDelegate{
		selectedContainerStyle: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("220")).
			PaddingLeft(1),

		unselectedContainerStyle: lipgloss.NewStyle().
			PaddingLeft(2),

		commitTypeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		dateStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		idStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		msgOriginalStyle: lipgloss.NewStyle(),

		msgTranslatedStyle: lipgloss.NewStyle(),
	}
}

func (d HistoryCommitDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(HistoryCommitItem)
	if !ok {
		return
	}

	commit := it.commit
	tagStr := fmt.Sprintf("(Type: %s)", commit.Type)

	dateStr := commit.CreatedAt.Format("2006-01-02 15:04")
	idStr := fmt.Sprintf("(ID: %d)", commit.ID)
	originalMsg := commit.MessageEN
	translatedMsg := commit.MessageES

	var (
		indicator                 = " "
		indicatorStyle            lipgloss.Style
		currentCommitTypeStyle    lipgloss.Style
		itemDisplayStyle          lipgloss.Style
		currentDateStyle          lipgloss.Style
		currentIDStyle            lipgloss.Style
		currentMsgOriginalStyle   lipgloss.Style
		currentMsgTranslatedStyle lipgloss.Style
	)

	indicatorStyle = lipgloss.NewStyle().Foreground(lipgloss.BrightYellow)

	if index == m.Index() {
		indicator = "❯"
		currentCommitTypeStyle = d.commitTypeStyle.Foreground(lipgloss.BrightMagenta)
		itemDisplayStyle = d.selectedContainerStyle
		currentDateStyle = d.dateStyle.Bold(true).
			Foreground(lipgloss.Color("10"))
		currentIDStyle = d.idStyle.Bold(true).
			Foreground(lipgloss.Color("45"))
		currentMsgOriginalStyle = d.msgOriginalStyle.Foreground(
			lipgloss.Color("255"),
		)
		currentMsgTranslatedStyle = d.msgTranslatedStyle.Italic(true).
			Foreground(lipgloss.Color("205"))
	} else {
		itemDisplayStyle = d.unselectedContainerStyle
		currentDateStyle = d.dateStyle
		currentIDStyle = d.idStyle
		currentMsgOriginalStyle = d.msgOriginalStyle
		currentMsgTranslatedStyle = d.msgTranslatedStyle
		currentCommitTypeStyle = d.commitTypeStyle
	}
	contentAvailableWidth := m.Width() - itemDisplayStyle.GetHorizontalPadding() - itemDisplayStyle.
		GetHorizontalBorderSize() - lipgloss.Width(indicator+" ")

	line1Content := lipgloss.JoinHorizontal(lipgloss.Top,
		currentDateStyle.Render(dateStr),
		" ",
		currentIDStyle.Render(idStr),
		" ",
		currentCommitTypeStyle.Render(tagStr),
	)

	msgPrefixWidth := 7
	maxMsgLength := contentAvailableWidth - msgPrefixWidth
	maxMsgLength = max(maxMsgLength, 10)

	truncatedOriginalMsg := truncateString(originalMsg, maxMsgLength)
	truncatedTranslatedMsg := truncateString(translatedMsg, maxMsgLength)

	line1 := fmt.Sprintf("%s %s", indicatorStyle.Render(indicator), line1Content)
	line2 := fmt.Sprintf("  %s %s", "Msg:", currentMsgOriginalStyle.Render(truncatedOriginalMsg))
	line3 := fmt.Sprintf(
		"  %s %s",
		"Trn:",
		currentMsgTranslatedStyle.Render(truncatedTranslatedMsg),
	)

	finalRender := lipgloss.JoinVertical(lipgloss.Left,
		line1,
		line2,
		line3,
	)

	fmt.Fprint(w, itemDisplayStyle.Width(m.Width()).Render(finalRender))
}

func (d HistoryCommitDelegate) Height() int  { return 3 }
func (d HistoryCommitDelegate) Spacing() int { return 1 }

func NewHistoryCommitList(workspaceCommits []storage.Commit, pwd string) list.Model {
	items := make([]list.Item, len(workspaceCommits))
	for i, c := range workspaceCommits {
		items[i] = HistoryCommitItem{commit: c}
	}

	historyList := list.New(items, NewHistoryCommitDelegate(), 0, 0)
	historyList.Title = fmt.Sprintf("%s - %s", "Commit History", TruncatePath(pwd, 2))
	historyList.SetShowHelp(false)
	historyList.SetStatusBarItemName("commit", "commits")
	historyList.SetFilteringEnabled(true)
	return historyList
}

func truncateString(s string, maxLen int) string {
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

func NewCommitTypeList(commitTypes []commit.CommitType, commitFormat string) list.Model {
	items := make([]list.Item, len(commitTypes))
	for i, ct := range commitTypes {
		items[i] = CommitTypeItem{CommitType: ct}
	}

	delegate := CommitTypeDelegate{
		TypeFormat: commitFormat,
	}
	typeList := list.New(items, delegate, 0, 0)
	typeList.Title = "Choose Commit Type"
	typeList.SetFilteringEnabled(true)
	typeList.SetShowHelp(false)

	return typeList
}
