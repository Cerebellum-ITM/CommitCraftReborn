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

// HistoryReleaseItem wraps a release row so the bubbles list can render it
// through HistoryReleaseDelegate. Implements list.Item — the actual filter
// value comes from the package-level `currentReleaseFilterMode` so cycling
// the filter pill applies live without rebuilding items.
type HistoryReleaseItem struct {
	release storage.Release
}

func (hri HistoryReleaseItem) Title() string       { return "" }
func (hri HistoryReleaseItem) Description() string { return "" }

// FilterValue returns the field that matches the active release filter
// mode, mirroring the workspace history's mode-aware filter dispatch.
func (hri HistoryReleaseItem) FilterValue() string {
	switch CurrentReleaseFilterMode() {
	case ReleaseFilterModeType:
		return hri.release.Type
	case ReleaseFilterModeVersion:
		return hri.release.Version
	case ReleaseFilterModeBranch:
		return hri.release.Branch
	default:
		return hri.release.Title
	}
}

// HistoryReleaseDelegate renders a single dense row per release matching
// the History (commits) layout: id+type-block | title (flex) | when.
//
//	#5 REL    [v0.21.2] main: bug fixes        04-29 11:20
type HistoryReleaseDelegate struct {
	list.DefaultDelegate
	Theme        *styles.Theme
	globalConfig config.Config
}

func NewHistoryReleaseDelegate(globalConfig config.Config, theme *styles.Theme) list.ItemDelegate {
	return HistoryReleaseDelegate{globalConfig: globalConfig, Theme: theme}
}

// releaseTypePaletteTag picks the commit-type palette tag whose colors
// paint the row's TYPE chip / scope pill. REL gets the green ADD palette,
// MERGE rides STYLE (purple). Anything else falls back to STYLE so an
// unknown type still renders something visible.
func releaseTypePaletteTag(t string) string {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "REL":
		return "ADD"
	case "MERGE":
		return "STYLE"
	default:
		return "STYLE"
	}
}

func (d HistoryReleaseDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(HistoryReleaseItem)
	if !ok {
		return
	}
	release := it.release
	selected := index == m.Index()
	tag := releaseTypePaletteTag(release.Type)

	idStr := fmt.Sprintf("#%-4d", release.ID)
	typeTag := strings.ToUpper(release.Type)
	if len(typeTag) > styles.CommitTypeChipInnerWidth {
		typeTag = typeTag[:styles.CommitTypeChipInnerWidth]
	}

	var typeChipStyle lipgloss.Style
	if selected {
		typeChipStyle = styles.CommitTypeBlockStyle(d.Theme, tag).Bold(true)
	} else {
		typeChipStyle = styles.CommitTypeMsgStyle(d.Theme, tag)
	}
	typeBlock := typeChipStyle.
		Width(styles.CommitTypeChipInnerWidth).
		Padding(0, 1).
		Align(lipgloss.Center).
		Render(typeTag)

	var idStyle lipgloss.Style
	if selected {
		idStyle = styles.CommitTypeMsgStyle(d.Theme, tag).Bold(true)
	} else {
		idStyle = lipgloss.NewStyle().Foreground(d.Theme.Muted)
	}
	idRendered := idStyle.Render(idStr)

	var versionPillStyle, titleStyle lipgloss.Style
	colonStyle := lipgloss.NewStyle().Foreground(d.Theme.Muted)
	if selected {
		versionPillStyle = styles.CommitTypeMsgStyle(d.Theme, tag).Padding(0, 1)
		titleStyle = styles.CommitTypeMsgStyle(d.Theme, tag).Bold(true)
	} else {
		versionPillStyle = lipgloss.NewStyle().Foreground(d.Theme.Secondary)
		titleStyle = lipgloss.NewStyle().Foreground(d.Theme.Muted)
	}

	dateStr := release.CreatedAt.Format("01-02 15:04")
	var dateStyle lipgloss.Style
	if selected {
		dateStyle = styles.CommitTypeMsgStyle(d.Theme, tag).Bold(true)
	} else {
		dateStyle = lipgloss.NewStyle().Foreground(d.Theme.Muted)
	}
	dateRendered := dateStyle.Render(dateStr)

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

	versionPart := strings.TrimSpace(release.Version)
	titlePart := release.Title
	if titlePart == "" {
		titlePart = release.Branch
	}
	available := msgWidth
	if available < 4 {
		available = 4
	}

	var msgBlock string
	if versionPart != "" {
		const maxVersionLen = 12
		shownVersion := versionPart
		if len(shownVersion) > maxVersionLen {
			shownVersion = shownVersion[:maxVersionLen]
		}
		var versionRendered, separator string
		if selected {
			versionRendered = versionPillStyle.Render(shownVersion)
			separator = " "
		} else {
			versionRendered = versionPillStyle.Render(shownVersion)
			separator = colonStyle.Render(": ")
		}
		consumed := lipgloss.Width(versionRendered) + lipgloss.Width(separator)
		titleAvail := available - consumed
		if titleAvail < 1 {
			titleAvail = 1
		}
		msgBlock = versionRendered + separator + titleStyle.Render(
			TruncateString(titlePart, titleAvail),
		)
	} else {
		msgBlock = titleStyle.Render(TruncateString(titlePart, available))
	}

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

func (d HistoryReleaseDelegate) Height() int  { return 1 }
func (d HistoryReleaseDelegate) Spacing() int { return 0 }

func NewHistoryReleaseList(
	workspaceReleases []storage.Release,
	pwd string,
	globalConfig config.Config,
	theme *styles.Theme,
) list.Model {
	items := make([]list.Item, len(workspaceReleases))
	for i, r := range workspaceReleases {
		items[i] = HistoryReleaseItem{release: r}
	}

	historyList := list.New(items, NewHistoryReleaseDelegate(globalConfig, theme), 0, 0)
	historyList.Title = fmt.Sprintf("%s: %s", "Working directory", TruncatePath(pwd, 2))
	historyList.SetShowTitle(false)
	historyList.SetShowFilter(false)
	historyList.SetShowHelp(false)
	historyList.SetStatusBarItemName("release", "releases")
	historyList.Styles.StatusBar = historyList.Styles.StatusBar.Foreground(theme.Accent)
	historyList.SetFilteringEnabled(true)
	historyList.StatusMessageLifetime = 5 * time.Second
	return historyList
}
