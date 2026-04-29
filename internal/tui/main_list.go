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
	selected := index == m.Index()

	idStr := fmt.Sprintf("#%-4d", commit.ID)
	typeTag := strings.ToUpper(commit.Type)
	// Standardize the chip's content width so every row's message column
	// starts at the same column. Tags longer than maxTypeTagLen are
	// hard-truncated (REFACTOR → REFACT) so the chip itself never grows.
	const maxTypeTagLen = 6
	if len(typeTag) > maxTypeTagLen {
		typeTag = typeTag[:maxTypeTagLen]
	}

	// Type chip uses the strong (block) palette when selected and the
	// dim (msg) palette otherwise. The chip is always shown so the
	// row's type identity is legible, but it intensifies under the
	// cursor along with the scope pill and title.
	var typeChipStyle lipgloss.Style
	if selected {
		typeChipStyle = styles.CommitTypeBlockStyle(d.Theme, commit.Type).Bold(true)
	} else {
		typeChipStyle = styles.CommitTypeMsgStyle(d.Theme, commit.Type)
	}
	typeBlock := typeChipStyle.
		Width(maxTypeTagLen).
		Padding(0, 1).
		Render(typeTag)

	// ID and date mirror the title's selection treatment: msg-palette
	// colors + Bold under the cursor, plain Muted text otherwise.
	var idStyle lipgloss.Style
	if selected {
		idStyle = styles.CommitTypeMsgStyle(d.Theme, commit.Type).Bold(true)
	} else {
		idStyle = lipgloss.NewStyle().Foreground(d.Theme.Muted)
	}
	idRendered := idStyle.Render(idStr)

	// Scope and title only "light up" with palette colors when the row
	// is selected. Unselected rows render the original (pre-palette)
	// look — plain Secondary scope with a `": "` separator and a Muted
	// title — so the History list reads as a clean stream and only the
	// cursor row pops with the four-color identity.
	var scopePillStyle, titleStyle lipgloss.Style
	colonStyle := lipgloss.NewStyle().Foreground(d.Theme.Muted)
	if selected {
		scopePillStyle = styles.CommitTypeMsgStyle(d.Theme, commit.Type).Padding(0, 1)
		titleStyle = styles.CommitTypeMsgStyle(d.Theme, commit.Type).Bold(true)
	} else {
		scopePillStyle = lipgloss.NewStyle().Foreground(d.Theme.Secondary)
		titleStyle = lipgloss.NewStyle().Foreground(d.Theme.Muted)
	}

	dateStr := commit.CreatedAt.Format("01-02 15:04")
	var dateStyle lipgloss.Style
	if selected {
		dateStyle = styles.CommitTypeMsgStyle(d.Theme, commit.Type).Bold(true)
	} else {
		dateStyle = lipgloss.NewStyle().Foreground(d.Theme.Muted)
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
		// Truncate the scope text so a long scope doesn't crowd the
		// title column.
		const maxScopeLen = 12
		shownScope := scopePart
		if len(shownScope) > maxScopeLen {
			shownScope = shownScope[:maxScopeLen]
		}
		var scopeRendered, separator string
		if selected {
			// Pill with padding(0,1) — the pill chrome itself acts as
			// the separator, followed by a single space.
			scopeRendered = scopePillStyle.Render(shownScope)
			separator = " "
		} else {
			// Original look: plain Secondary text + ": " separator.
			scopeRendered = scopePillStyle.Render(shownScope)
			separator = colonStyle.Render(": ")
		}
		consumed := lipgloss.Width(scopeRendered) + lipgloss.Width(separator)
		titleAvail := available - consumed
		if titleAvail < 1 {
			titleAvail = 1
		}
		msgBlock = scopeRendered + separator + titleStyle.Render(
			TruncateString(titlePart, titleAvail),
		)
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
