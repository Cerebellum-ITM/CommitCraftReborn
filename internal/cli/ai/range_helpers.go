package ai

import (
	"fmt"
	"os/exec"
	"strings"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/git"
)

// projectToReleaseCommits maps the git.CommitRange shape used by the
// range helpers to the aiengine.ReleaseCommit shape the release
// pipeline consumes. Identical fields; the projection exists so
// callers don't have to import both `git` and `aiengine` just to
// build a slice literal.
func projectToReleaseCommits(in []git.CommitRange) []aiengine.ReleaseCommit {
	out := make([]aiengine.ReleaseCommit, len(in))
	for i, c := range in {
		out[i] = aiengine.ReleaseCommit{
			Hash:    c.Hash,
			Date:    c.Date,
			Subject: c.Subject,
			Body:    c.Body,
		}
	}
	return out
}

// serializeCommitRange stores the input commit list on the draft's
// Diff_code field so it stays inspectable after the fact. Plain-text
// format mirroring `git log --oneline` plus the body — enough for
// traceability without parsing the actual diff.
func serializeCommitRange(commits []git.CommitRange) string {
	var b strings.Builder
	for _, c := range commits {
		fmt.Fprintf(&b, "%s %s %s\n%s\n\n", c.Hash, c.Date, c.Subject, c.Body)
	}
	return strings.TrimRight(b.String(), "\n")
}

// lastTagAt is the workspace-aware sibling of git.GetLastGitTag.
// Returns the most recent tag by natural-version sort, or empty
// string + nil error when the repo has no tags. Shells out to
// `git -C <workspace> tag --sort=-v:refname` and picks the first
// non-empty line.
func lastTagAt(workspace string) (string, error) {
	args := []string{}
	if workspace != "" {
		args = append(args, "-C", workspace)
	}
	args = append(args, "tag", "--sort=-v:refname")
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line, nil
		}
	}
	return "", nil
}
