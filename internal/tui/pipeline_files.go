package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/git"
)

// refreshPipelineNumstat reloads the cached numstat map from `git diff
// --staged --numstat`. Cheap (one fork+exec); we call it on tab enter
// and after every pipeline re-run so per-row counts stay accurate.
func refreshPipelineNumstat(model *Model) {
	ns, err := git.GetStagedNumstat()
	if err != nil {
		model.log.Debug("pipeline: failed to read numstat", "err", err)
		model.pipeline.numstat = nil
		return
	}
	model.pipeline.numstat = ns
}

// setDiffFromSelectedFile pushes the staged diff for the currently
// highlighted entry in pipelineDiffList into pipeline.diffViewport,
// pre-coloured per `+`/`-`/`@@` line. Called whenever the cursor moves.
func setDiffFromSelectedFile(model *Model) {
	it, ok := model.pipelineDiffList.SelectedItem().(DiffFileItem)
	if !ok {
		model.pipeline.diffViewport.SetContent("")
		return
	}
	var diff string
	var err error
	if model.usePreloadedDiff {
		diff = model.dbFileDiffs[it.FilePath]
	} else {
		diff, err = git.GetStagedFileDiff(it.FilePath)
	}
	if err != nil || strings.TrimSpace(diff) == "" {
		model.pipeline.diffViewport.SetContent(
			lipgloss.NewStyle().
				Foreground(model.Theme.Muted).
				Render("(no staged diff for this file)"),
		)
		return
	}

	lines := strings.Split(diff, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, colorizeDiffLine(line))
	}
	model.pipeline.diffViewport.SetContent(strings.Join(out, "\n"))
	model.pipeline.diffViewport.GotoTop()
}

// pipelineFilesFooter renders the bottom totals row in the changed-files
// panel: "<n> files +<adds> -<dels>".
func (model *Model) pipelineFilesFooter() string {
	theme := model.Theme
	count := len(model.pipelineDiffList.Items())
	adds, dels := 0, 0
	for _, ns := range model.pipeline.numstat {
		if ns.Adds > 0 {
			adds += ns.Adds
		}
		if ns.Dels > 0 {
			dels += ns.Dels
		}
	}
	muted := lipgloss.NewStyle().Foreground(theme.Muted)
	addStyle := lipgloss.NewStyle().Foreground(theme.Add)
	delStyle := lipgloss.NewStyle().Foreground(theme.Del)

	parts := []string{
		muted.Render(plural(count, "file", "files")),
		addStyle.Render("+" + strconv.Itoa(adds)),
		delStyle.Render("-" + strconv.Itoa(dels)),
	}
	return strings.Join(parts, "  ")
}

// pipelineFilesDelegate is a 2-row list delegate for the Pipeline tab's
// changed-files panel: file path on top row, "+N -M" counters underneath.
// Rebuild + SetDelegate on the list whenever numstat refreshes so each
// row shows up-to-date counts (the delegate captures the map by value).
type pipelineFilesDelegate struct {
	numstat map[string]git.FileNumstat
}

func (d pipelineFilesDelegate) Height() int  { return 2 }
func (d pipelineFilesDelegate) Spacing() int { return 0 }
func (d pipelineFilesDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d pipelineFilesDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(DiffFileItem)
	if !ok {
		return
	}
	statusText := it.Status
	statusStyle := statusStyles[""]
	if s, found := statusStyles[it.Status]; found {
		statusStyle = s
	}
	name := filepath.Base(it.FilePath)
	dir := filepath.Dir(it.FilePath)
	if dir == "." {
		dir = ""
	}
	dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("253"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	prefix := "  "
	if index == m.Index() {
		prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).Render("❯ ")
		nameStyle = nameStyle.Foreground(lipgloss.Color("205")).Bold(true)
	}

	var dirSuffix string
	if dir != "" {
		dirSuffix = dirStyle.Render("  " + dir)
	}

	row1 := fmt.Sprintf("%s%s %s%s",
		prefix,
		statusStyle.Render(statusText),
		nameStyle.Render(name),
		dirSuffix,
	)

	ns := d.numstat[it.FilePath]
	addsText := "+0"
	delsText := "-0"
	if ns.Adds > 0 {
		addsText = "+" + strconv.Itoa(ns.Adds)
	} else if ns.Adds < 0 {
		addsText = "+bin"
	}
	if ns.Dels > 0 {
		delsText = "-" + strconv.Itoa(ns.Dels)
	} else if ns.Dels < 0 {
		delsText = "-bin"
	}
	row2 := "      " + mutedStyle.Render(addStyle.Render(addsText)+" "+delStyle.Render(delsText))

	fmt.Fprintf(w, "%s\n%s", row1, row2)
}

// parseDbDiff extracts files, numstats, and per-file diff text from the
// Diff_code blob persisted by GetStagedDiffSummary (format:
// "=== <path> ===\n<diff bytes>\n=== <next path> ===\n..."). Returns
// items sorted alphabetically so the list stays stable across reloads.
func parseDbDiff(diffCode string) ([]list.Item, map[string]git.FileNumstat, map[string]string) {
	if strings.TrimSpace(diffCode) == "" {
		return nil, nil, nil
	}
	type block struct {
		path string
		body strings.Builder
	}
	var blocks []*block
	var cur *block
	for _, line := range strings.Split(diffCode, "\n") {
		if strings.HasPrefix(line, "=== ") && strings.HasSuffix(line, " ===") {
			path := strings.TrimSuffix(strings.TrimPrefix(line, "=== "), " ===")
			cur = &block{path: path}
			blocks = append(blocks, cur)
			continue
		}
		if cur != nil {
			cur.body.WriteString(line)
			cur.body.WriteByte('\n')
		}
	}
	if len(blocks) == 0 {
		return nil, nil, nil
	}
	paths := make([]string, 0, len(blocks))
	bodies := make(map[string]string, len(blocks))
	numstat := make(map[string]git.FileNumstat, len(blocks))
	for _, b := range blocks {
		body := strings.TrimRight(b.body.String(), "\n")
		bodies[b.path] = body
		paths = append(paths, b.path)

		adds, dels := 0, 0
		for _, l := range strings.Split(body, "\n") {
			switch {
			case strings.HasPrefix(l, "+++") || strings.HasPrefix(l, "---"):
				continue
			case strings.HasPrefix(l, "+"):
				adds++
			case strings.HasPrefix(l, "-"):
				dels++
			}
		}
		numstat[b.path] = git.FileNumstat{Adds: adds, Dels: dels}
	}
	sort.Strings(paths)
	items := make([]list.Item, 0, len(paths))
	for _, p := range paths {
		items = append(items, DiffFileItem{FilePath: p, Status: "M"})
	}
	return items, numstat, bodies
}

// loadPipelineFilesFromDb rebuilds the pipeline files list + numstat from
// a DB-loaded commit's Diff_code. Caches the per-file diff text on the
// model so setDiffFromSelectedFile can render it without invoking git.
func loadPipelineFilesFromDb(model *Model, diffCode string) {
	items, numstat, bodies := parseDbDiff(diffCode)
	model.dbFileDiffs = bodies
	if items == nil {
		model.pipeline.numstat = nil
		model.pipelineDiffList.SetItems(nil)
		applyPipelineFilesDelegate(model)
		return
	}
	model.pipeline.numstat = numstat
	model.pipelineDiffList.SetItems(items)
	model.pipelineDiffList.Select(0)
	applyPipelineFilesDelegate(model)
}

// applyPipelineFilesDelegate swaps the list delegate with a fresh
// pipelineFilesDelegate that captures the latest numstat map. Called
// after refreshPipelineNumstat so the per-row counts re-render.
func applyPipelineFilesDelegate(model *Model) {
	model.pipelineDiffList.SetDelegate(pipelineFilesDelegate{
		numstat: model.pipeline.numstat,
	})
}

func plural(n int, sing, plur string) string {
	if n == 1 {
		return "1 " + sing
	}
	return strconv.Itoa(n) + " " + plur
}
