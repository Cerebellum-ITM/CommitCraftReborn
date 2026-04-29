package tui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"
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
	return hci.commit.MessageEN + " " + strings.Join(hci.commit.KeyPoints, " ") + " " +
		hci.commit.Type + " " + hci.commit.Scope
}

// HistoryCommitDelegate renders a single dense row per commit:
//
//	#371 DOC   [DOC] docs: update docs organization…  04-23 23:18
//
// Columns: id+type-block (≈12 cols) | message (flex) | when (12 cols, right-aligned).
// The selected row gets a left accent border and a Surface background so the
// cursor position stays visible without breaking the dense layout.
type HistoryCommitDelegate struct {
	list.DefaultDelegate

	Theme        *styles.Theme
	globalConfig config.Config
}

func NewHistoryCommitDelegate(globalConfig config.Config, theme *styles.Theme) list.ItemDelegate {
	return HistoryCommitDelegate{
		globalConfig: globalConfig,
		Theme:        theme,
	}
}

func (d HistoryCommitDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(HistoryCommitItem)
	if !ok {
		return
	}

	commit := it.commit
	palette := styles.CommitTypePalette(d.Theme, commit.Type)
	selected := index == m.Index()

	idStr := fmt.Sprintf("#%-4d", commit.ID)
	typeTag := strings.ToUpper(commit.Type)

	// Type block (chip on the left of the row): per-type bg + fg.
	typeBlock := lipgloss.NewStyle().
		Background(palette.BgBlock).
		Foreground(palette.FgBlock).
		Bold(true).
		Padding(0, 1).
		Render(typeTag)

	idStyle := lipgloss.NewStyle().Foreground(d.Theme.Muted)
	if selected {
		idStyle = idStyle.Foreground(d.Theme.Primary).Bold(true)
	}
	idRendered := idStyle.Render(idStr)

	scopeStyle := lipgloss.NewStyle().Foreground(d.Theme.Secondary)
	colonStyle := lipgloss.NewStyle().Foreground(d.Theme.Muted)
	titleStyle := lipgloss.NewStyle().Foreground(d.Theme.FG)
	if selected {
		titleStyle = titleStyle.Bold(true)
	} else {
		titleStyle = titleStyle.Foreground(d.Theme.Muted)
	}

	dateStr := commit.CreatedAt.Format("01-02 15:04")
	dateStyle := lipgloss.NewStyle().Foreground(d.Theme.Subtle)
	if selected {
		dateStyle = dateStyle.Foreground(d.Theme.Primary)
	}
	dateRendered := dateStyle.Render(dateStr)

	// Container styles: left border accent + Surface bg when selected.
	container := lipgloss.NewStyle().PaddingLeft(2)
	if selected {
		container = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(d.Theme.Primary).
			Background(d.Theme.Surface).
			PaddingLeft(1)
	}

	totalWidth := m.Width() - container.GetHorizontalFrameSize() - container.GetHorizontalPadding()
	if totalWidth < 20 {
		totalWidth = 20
	}

	leftBlock := lipgloss.JoinHorizontal(lipgloss.Top, idRendered, " ", typeBlock)
	leftWidth := lipgloss.Width(leftBlock)
	dateWidth := lipgloss.Width(dateRendered)
	gapBeforeDate := 2

	msgWidth := totalWidth - leftWidth - dateWidth - 1 - gapBeforeDate
	if msgWidth < 8 {
		msgWidth = 8
	}

	// Build "scope: title" suffix, truncated to fit. The type chip on the
	// left is the only place the tag is rendered to avoid duplicating the
	// information on every row.
	scopePart := strings.TrimSpace(commit.Scope)
	titlePart := commit.MessageEN
	available := msgWidth
	if available < 4 {
		available = 4
	}

	var msgBlock string
	if scopePart != "" {
		head := scopeStyle.Render(scopePart) + colonStyle.Render(": ")
		headPlain := scopePart + ": "
		titleAvail := available - lipgloss.Width(headPlain)
		if titleAvail < 1 {
			titleAvail = 1
		}
		msgBlock = head + titleStyle.Render(TruncateString(titlePart, titleAvail))
	} else {
		msgBlock = titleStyle.Render(TruncateString(titlePart, available))
	}

	// Pad the middle area so the date is right-aligned.
	currentRowWidth := leftWidth + 1 + lipgloss.Width(msgBlock)
	pad := totalWidth - currentRowWidth - dateWidth
	if pad < 1 {
		pad = 1
	}

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftBlock,
		" ",
		msgBlock,
		strings.Repeat(" ", pad),
		dateRendered,
	)

	fmt.Fprint(w, container.Width(m.Width()).Render(row))
}

func (d HistoryCommitDelegate) Height() int  { return 1 }
func (d HistoryCommitDelegate) Spacing() int { return 0 }

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
	historyList.SetShowFilter(false)
	historyList.SetShowHelp(false)
	historyList.SetStatusBarItemName("commit", "commits")
	historyList.Styles.StatusBar = historyList.Styles.StatusBar.Foreground(theme.Accent)
	historyList.SetFilteringEnabled(true)
	historyList.StatusMessageLifetime = 5 * time.Second
	return historyList
}
