package tui

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// releaseChooseSelectedOnly mirrors the workspace commit picker's
// segmented "All commits / Selected only" indicator into FilterValue.
// When true, unselected items return an empty FilterValue, which the
// custom list.Filter below treats as "skip" — that's how we hide
// non-selected rows without rebuilding the underlying items slice
// (rebuilding would lose the cursor + scroll position).
var releaseChooseSelectedOnly bool

// releaseChooseSentinel is the magic filter text we feed the list
// bubble when "Selected only" is on but the user hasn't typed any
// query. The list short-circuits filterItems to "show everything"
// whenever FilterInput.Value() is empty — bypassing our custom Filter
// entirely — so we hand it a non-empty sentinel that
// `releaseChooseListFilter` recognises and treats as "no user term".
const releaseChooseSentinel = "\x00release-choose-selected-only\x00"

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
	if releaseChooseSelectedOnly && !wsi.Selected {
		return ""
	}
	// Mirror HistoryCommitItem.FilterValue: switch on the active picker
	// mode so cycling ctrl+f re-evaluates against a different field
	// without rebuilding items.
	switch CurrentReleasePickerFilterMode() {
	case ReleasePickerFilterModeHash:
		return wsi.Hash
	case ReleasePickerFilterModeType:
		return extractCommitTypeTag(wsi.Subject)
	case ReleasePickerFilterModeTag:
		return strings.Join(wsi.Tags, " ")
	default:
		return wsi.Subject
	}
}

// extractCommitTypeTag pulls the leading "[XXX]" tag off a Conventional
// Commit-style subject so the picker's TYPE filter mode can match
// against just the type. Returns "" when the subject doesn't start
// with a bracketed token.
func extractCommitTypeTag(subject string) string {
	s := strings.TrimSpace(subject)
	if !strings.HasPrefix(s, "[") {
		return ""
	}
	end := strings.IndexByte(s, ']')
	if end <= 1 {
		return ""
	}
	return s[1:end]
}

// releaseChooseListFilter is the workspace picker's filter func. It
// honours `releaseChooseSelectedOnly` by dropping items whose
// FilterValue is "" (i.e. unselected ones) and otherwise delegates to
// list.DefaultFilter for the actual fuzzy match. The empty-term path
// returns every kept index in order so the cursor remains usable when
// the user hasn't typed anything yet.
func releaseChooseListFilter(term string, targets []string) []list.Rank {
	if term == releaseChooseSentinel {
		term = ""
	}
	if !releaseChooseSelectedOnly {
		return list.DefaultFilter(term, targets)
	}
	keptIdx := make([]int, 0, len(targets))
	keptStr := make([]string, 0, len(targets))
	for i, t := range targets {
		if t == "" {
			continue
		}
		keptIdx = append(keptIdx, i)
		keptStr = append(keptStr, t)
	}
	if term == "" {
		out := make([]list.Rank, len(keptIdx))
		for i, idx := range keptIdx {
			out[i] = list.Rank{Index: idx}
		}
		return out
	}
	ranks := list.DefaultFilter(term, keptStr)
	for i := range ranks {
		ranks[i].Index = keptIdx[ranks[i].Index]
	}
	return ranks
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
	// The custom ReleaseFilterBar owns the "/" key + visible counter, so
	// drop the built-in status bar and pagination strip (they duplicate
	// the count rendered by the bar) and disable the list's filter
	// keybinding so "/" reaches our handler instead of opening the
	// list's own filter input.
	releaseList.SetShowStatusBar(false)
	releaseList.SetShowPagination(false)
	// SetShowFilter(false) hides the list's built-in "Filter: …" prompt
	// in the title row; the custom ReleaseFilterBar already renders it.
	// Without this, every time we drive the list into FilterApplied
	// state the bubble pops a second filter input on top of ours.
	releaseList.SetShowFilter(false)
	releaseList.KeyMap.Filter = key.NewBinding(key.WithDisabled())
	releaseList.KeyMap.ClearFilter = key.NewBinding(key.WithDisabled())
	releaseList.Filter = releaseChooseListFilter
	return releaseList
}
