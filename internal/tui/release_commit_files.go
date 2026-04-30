package tui

import (
	"strings"

	"charm.land/bubbles/v2/list"
)

// splitCommitDiffByFile parses the unified-diff output of a single commit
// (the `git log -p` body the release picker captures into
// WorkspaceCommitItem.Diff) and returns:
//   - an ordered slice of DiffFileItem (one per file touched), with a
//     status code derived from the diff headers (A/D/R/M);
//   - a map keyed by file path holding the per-file diff text so the
//     diff viewport can swap content without re-running git.
//
// The parser is intentionally tolerant: any block that doesn't start with
// "diff --git " is dropped on the floor. We don't need a perfect
// implementation here because the source string already comes from git
// itself; the only edge cases worth handling are renames and binary
// patches.
func splitCommitDiffByFile(diff string) ([]DiffFileItem, map[string]string) {
	if strings.TrimSpace(diff) == "" {
		return nil, map[string]string{}
	}

	const marker = "diff --git "
	items := []DiffFileItem{}
	bodies := map[string]string{}

	rest := diff
	idx := strings.Index(rest, marker)
	if idx == -1 {
		return nil, bodies
	}
	rest = rest[idx:]

	for {
		next := strings.Index(rest[len(marker):], marker)
		var block string
		if next == -1 {
			block = rest
			rest = ""
		} else {
			end := len(marker) + next
			block = rest[:end]
			rest = rest[end:]
		}

		path, status := parseDiffHeader(block)
		if path != "" {
			items = append(items, DiffFileItem{FilePath: path, Status: status})
			bodies[path] = block
		}
		if rest == "" {
			break
		}
	}
	return items, bodies
}

// parseDiffHeader pulls the file path (using the post-image side, b/...)
// and a single-letter status from a single-file diff block.
func parseDiffHeader(block string) (path, status string) {
	lines := strings.SplitN(block, "\n", 12)
	status = "M"
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			rest := strings.TrimPrefix(line, "diff --git ")
			if i := strings.Index(rest, " b/"); i != -1 {
				path = strings.TrimPrefix(rest[i+1:], "b/")
			}
		case strings.HasPrefix(line, "new file mode"):
			status = "A"
		case strings.HasPrefix(line, "deleted file mode"):
			status = "D"
		case strings.HasPrefix(line, "rename from "), strings.HasPrefix(line, "rename to "):
			status = "R"
		case strings.HasPrefix(line, "copy from "), strings.HasPrefix(line, "copy to "):
			status = "C"
		}
	}
	return path, status
}

// loadReleaseCommitFiles refreshes the per-file list + diff cache for the
// commit currently selected in releaseCommitList. Idempotent: if the
// active commit hasn't changed since the last call, it returns early.
// Selecting the first file (when present) drives the initial diff vp
// content.
func (model *Model) loadReleaseCommitFiles() {
	item, ok := model.releaseCommitList.SelectedItem().(WorkspaceCommitItem)
	if !ok {
		model.releaseSelectedCommitHash = ""
		model.releaseFilesList.SetItems([]list.Item{})
		model.releaseDiffViewport.SetContent("")
		return
	}
	if item.Hash == model.releaseSelectedCommitHash {
		return
	}
	model.releaseSelectedCommitHash = item.Hash

	files, bodies := splitCommitDiffByFile(item.Diff)
	if model.releaseFileDiffsByCommit == nil {
		model.releaseFileDiffsByCommit = map[string]map[string]string{}
	}
	model.releaseFileDiffsByCommit[item.Hash] = bodies

	listItems := make([]list.Item, len(files))
	for i, f := range files {
		listItems[i] = f
	}
	model.releaseFilesList.SetItems(listItems)
	if len(files) > 0 {
		model.releaseFilesList.Select(0)
		model.releaseDiffViewport.SetContent(bodies[files[0].FilePath])
	} else {
		model.releaseDiffViewport.SetContent("(no file diffs in this commit)")
	}
	model.releaseDiffViewport.GotoTop()
}

// refreshReleaseDiffForSelectedFile rebinds the diff viewport content to
// the file currently focused in releaseFilesList. Called after every
// change of the file-list selection.
func (model *Model) refreshReleaseDiffForSelectedFile() {
	bodies, ok := model.releaseFileDiffsByCommit[model.releaseSelectedCommitHash]
	if !ok {
		return
	}
	item, ok := model.releaseFilesList.SelectedItem().(DiffFileItem)
	if !ok {
		return
	}
	if body, found := bodies[item.FilePath]; found {
		model.releaseDiffViewport.SetContent(body)
		model.releaseDiffViewport.GotoTop()
	}
}
