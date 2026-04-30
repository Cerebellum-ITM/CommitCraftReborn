package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/git"
)

type DiffFileItem struct {
	FilePath string
	Status   string
}

func (d DiffFileItem) Title() string       { return filepath.Base(d.FilePath) }
func (d DiffFileItem) Description() string { return d.FilePath }
func (d DiffFileItem) FilterValue() string { return d.FilePath }

// diffFileShowFullPath toggles the diff-file picker between two
// rendering modes:
//   - false: filename-first (default), with the parent directory
//     dimmed and pushed to the right.
//   - true: full relative path written out as a single string with no
//     directory/file split.
//
// Flipped via ctrl+e while the release commit picker has focus on the
// files list. Package-level so the delegate can read it without
// threading state through every Render call.
var diffFileShowFullPath bool

type diffFileDelegate struct {
	useNerdFonts bool
}

func (d diffFileDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(DiffFileItem)
	if !ok {
		return
	}

	statusText := it.Status
	statusStyle := statusStyles[""]
	if s, found := statusStyles[it.Status]; found {
		statusStyle = s
	}

	var icon string
	if d.useNerdFonts {
		icon = GetNerdFontIcon(it.FilePath, false)
	} else {
		icon = "📄"
	}

	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("253"))

	var name, dirSuffix string
	if diffFileShowFullPath {
		name = it.FilePath
	} else {
		name = filepath.Base(it.FilePath)
	}

	if index == m.Index() {
		selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		fmt.Fprintf(w, "❯ %s %s %s%s",
			statusStyle.Render(statusText),
			selectedStyle.Render(icon),
			selectedStyle.Render(name),
			dirSuffix,
		)
	} else {
		fmt.Fprintf(w, "  %s %s %s%s",
			statusStyle.Render(statusText),
			nameStyle.Render(icon),
			nameStyle.Render(name),
			dirSuffix,
		)
	}
}

func (d diffFileDelegate) Height() int  { return 1 }
func (d diffFileDelegate) Spacing() int { return 0 }
func (d diffFileDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func NewDiffFileList(gitStatus git.StatusData, useNerdFonts bool) list.Model {
	paths := make([]string, 0, len(gitStatus.FileStatus))
	for p := range gitStatus.FileStatus {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	items := make([]list.Item, len(paths))
	for i, p := range paths {
		items[i] = DiffFileItem{FilePath: p, Status: gitStatus.FileStatus[p]}
	}

	l := list.New(items, diffFileDelegate{useNerdFonts: useNerdFonts}, 0, 0)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	return l
}
