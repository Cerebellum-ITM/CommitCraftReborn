package tui

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

type WorkspaceCommitItem struct {
	Selected bool
	Hash     string
	Date     string
	Subject  string
	Body     string
	Preview  string
	Diff     string
	// Tags holds the git refs pointing at this commit that are release tags.
	// Populated from `git log --pretty=...%D...` so we can surface the last
	// release boundary in the commit picker without extra git calls.
	Tags []string
}

// extractTagsFromRefs pulls release tags out of git's %D decoration string.
// The format looks like: "HEAD -> main, tag: v0.7.0, tag: v0.6.1, origin/main".
func extractTagsFromRefs(refs string) []string {
	if refs == "" {
		return nil
	}
	var tags []string
	for _, ref := range strings.Split(refs, ",") {
		ref = strings.TrimSpace(ref)
		if strings.HasPrefix(ref, "tag: ") {
			tags = append(tags, strings.TrimPrefix(ref, "tag: "))
		}
	}
	return tags
}

func (wsi WorkspaceCommitItem) FilterValue() string {
	return wsi.Hash + " " + wsi.Subject
}

type ReleaseListDelegate struct {
	list.DefaultDelegate
	Theme          *styles.Theme
	hashStyle      lipgloss.Style
	indicatorStyle lipgloss.Style
	textStyle      lipgloss.Style
	dateStyle      lipgloss.Style
	selectedStyle  lipgloss.Style
	tagStyle       lipgloss.Style
}

func NewReleaseListDelegate(theme *styles.Theme) list.ItemDelegate {
	base := theme.AppStyles().Base
	baseFg := theme.FgMuted
	baseState := base.Foreground(baseFg)

	return ReleaseListDelegate{
		Theme:          theme,
		hashStyle:      baseState,
		textStyle:      baseState,
		dateStyle:      baseState,
		selectedStyle:  base.Foreground(theme.FgBase),
		indicatorStyle: theme.AppStyles().IndicatorStyle,
		tagStyle: base.
			Foreground(theme.FgBase).
			Background(theme.Secondary).
			Padding(0, 1).
			Bold(true),
	}
}

func (d ReleaseListDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(WorkspaceCommitItem)
	if !ok {
		return
	}

	var indicator string
	var selected string
	var hashStyle lipgloss.Style
	var textStyle lipgloss.Style
	var dateStyle lipgloss.Style
	// var selectedStyle lipgloss.Style

	if item.Selected {
		selected = d.Theme.AppSymbols().Commit
	} else {
		selected = " "
	}

	if index == m.Index() {
		indicator = d.indicatorStyle.Render("❯")
		// selectedStyle = d.selectedStyle.Foreground(d.Theme.Yellow)
		hashStyle = d.hashStyle.Foreground(d.Theme.Accent)
		textStyle = d.textStyle.Foreground(d.Theme.FgBase)
		dateStyle = d.dateStyle.Foreground(d.Theme.Yellow)

	} else {
		indicator = " "
		hashStyle = d.hashStyle
		textStyle = d.textStyle
		dateStyle = d.dateStyle
		// selectedStyle = d.selectedStyle
	}

	hashLength := 11

	var tagString string
	if len(item.Tags) > 0 {
		tagString = d.tagStyle.Render(item.Tags[0]) + " "
	}

	maxWith := max(
		0,
		m.Width()-hashLength-lipgloss.Width(
			item.Date,
		)-lipgloss.Width(
			selected,
		)-lipgloss.Width(
			tagString,
		),
	)
	hashString := hashStyle.Render(TruncateString(item.Hash, hashLength))
	subjetString := textStyle.Render(TruncateString(item.Subject, maxWith))
	dateString := dateStyle.Render(item.Date)
	selectedString := d.selectedStyle.Render(selected)
	line := fmt.Sprintf(
		"%s %s %s %s %s%s",
		selectedString,
		indicator,
		hashString,
		dateString,
		tagString,
		subjetString,
	)
	fmt.Fprint(w, line)
}

func NewReleaseCommitList(pwd string, theme *styles.Theme) list.Model {
	var stderr bytes.Buffer
	var preview strings.Builder
	var workspaceItems []WorkspaceCommitItem

	commitHistoryCmd := exec.Command(
		"git",
		"log",
		"-p",
		"--date=format:%y-%m-%d %H:%M",
		"--pretty=format:%x00COMMIT_ITEM_START%x00%H%x00%s%x00%b%x00%ad%x00%D%x00COMMIT_METADATA_END%x00",
	)
	commitHistoryCmd.Stderr = &stderr

	rawOutput, _ := commitHistoryCmd.Output()
	rawOutputStr := string(rawOutput)
	commitBlocks := strings.Split(rawOutputStr, "\x00COMMIT_ITEM_START\x00")
	for _, block := range commitBlocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		parts := strings.SplitN(block, "\x00COMMIT_METADATA_END\x00", 2)
		if len(parts) < 1 {
			continue
		}

		metadataStr := parts[0]
		diffStr := ""
		if len(parts) > 1 {
			diffStr = strings.TrimPrefix(strings.TrimSpace(parts[1]), "\n")
		}

		metaFields := strings.Split(metadataStr, "\x00")
		commit := WorkspaceCommitItem{}
		if len(metaFields) > 0 {
			commit.Hash = metaFields[0]
		}
		if len(metaFields) > 1 {
			commit.Subject = metaFields[1]
			preview.WriteString(fmt.Sprintf("## %s\n", commit.Subject))
		}
		if len(metaFields) > 2 {
			commit.Body = metaFields[2]
			preview.WriteString(fmt.Sprintf("%s\n", commit.Body))
		}
		if len(metaFields) > 3 {
			commit.Date = metaFields[3]
		}
		if len(metaFields) > 4 {
			commit.Tags = extractTagsFromRefs(metaFields[4])
		}
		commit.Diff = diffStr
		preview.WriteString(fmt.Sprintf("```\n%s```", commit.Diff))
		commit.Preview = preview.String()
		preview.Reset()

		workspaceItems = append(workspaceItems, commit)
	}

	items := make([]list.Item, len(workspaceItems))
	for i, wsi := range workspaceItems {
		items[i] = wsi
	}
	releaseList := list.New(items, NewReleaseListDelegate(theme), 0, 0)
	releaseList.SetShowHelp(false)
	releaseList.SetShowTitle(false)
	releaseList.SetStatusBarItemName("commit", "commits")
	return releaseList
}
