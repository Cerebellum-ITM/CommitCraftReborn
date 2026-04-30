// Package git wraps the small set of git porcelain commands the TUI needs.
// It is intentionally low-level: every function shells out to the git binary
// and returns plain Go types so that the rest of the codebase doesn't grow a
// dependency on a particular git library.
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetCurrentGitBranch returns the current branch's short name (the value of
// `git rev-parse --abbrev-ref HEAD`).
func GetCurrentGitBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w\nStderr: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("Error executing git command: %v", err)
	}

	branch := strings.TrimSpace(stdout.String())
	return branch, nil
}

// GetGitBranches returns every local branch name. The current-branch marker
// "* " emitted by `git branch --list` is stripped.
func GetGitBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "--list")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w\nStderr: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("Error executing git command: %v", err)
	}

	outputLines := strings.Split(stdout.String(), "\n")

	var branches []string
	for _, line := range outputLines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		trimmedLine = strings.TrimPrefix(trimmedLine, "* ")
		branches = append(branches, trimmedLine)
	}

	return branches, nil
}

// GetGitDiffStat returns the staged diff --stat output (files changed with
// insertion/deletion counts).
func GetGitDiffStat() (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--stat")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to get git diff --stat: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetStagedDiffSummary builds a single string with the staged diff of every
// changed file, capped at maxDiffChars total characters. Used as input for
// the AI change analyzer. Operates on the current working directory.
func GetStagedDiffSummary(maxDiffChars int) (string, error) {
	return GetStagedDiffSummaryAt("", maxDiffChars)
}

// GetStagedDiffSummaryAt is the path-aware variant: when workspace is
// non-empty, the underlying git invocations are routed through `-C`
// so the diff is read from that repo rather than the caller's cwd.
// An empty workspace falls back to cwd, matching GetStagedDiffSummary.
// Used by `ai regenerate --refresh-diff` so refreshing the snapshot
// works from any directory while the commit's stored Workspace is
// the source of truth for which repo to inspect.
func GetStagedDiffSummaryAt(workspace string, maxDiffChars int) (string, error) {
	gitArgs := func(rest ...string) []string {
		if workspace == "" {
			return rest
		}
		return append([]string{"-C", workspace}, rest...)
	}

	cmdFiles := exec.Command("git", gitArgs("diff", "--cached", "--name-only")...)
	stagedFilesBytes, err := cmdFiles.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git command failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to get staged files: %w", err)
	}

	if len(stagedFilesBytes) == 0 {
		return "", nil
	}

	stagedFiles := strings.Split(strings.TrimSpace(string(stagedFilesBytes)), "\n")

	var resultBuilder strings.Builder
	currentChars := 0

	for _, file := range stagedFiles {
		if file == "" {
			continue
		}

		cmdDiff := exec.Command("git", gitArgs("diff", "--cached", "--unified=0", "--", file)...)
		diffBytes, err := cmdDiff.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get diff for file %s: %w", file, err)
		}

		block := fmt.Sprintf("=== %s ===\n%s\n", file, string(diffBytes))
		blockChars := len(block)

		if currentChars+blockChars > maxDiffChars {
			break
		}

		resultBuilder.WriteString(block)
		currentChars += blockChars
	}

	return resultBuilder.String(), nil
}

// GetGitDiffNameStatus returns the staged file → status code map (A, M, D,
// R…). An empty repo or no staged changes returns an empty map (not an
// error).
func GetGitDiffNameStatus() (map[string]string, error) {
	cmd := exec.Command("git", "diff", "--staged", "--name-status")
	outputBytes, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to get git diff --name-status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(outputBytes)), "\n")
	statusMap := make(map[string]string)

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			status := parts[0]
			filePath := parts[1]
			statusMap[filePath] = status
		}
	}

	return statusMap, nil
}

// FileNumstat captures the per-file `+N -M` counts produced by
// `git diff --staged --numstat`. Binary files report Adds = Dels = -1.
type FileNumstat struct {
	Adds int
	Dels int
}

// GetStagedNumstat parses `git diff --staged --numstat` into a map keyed
// by file path. Used by the Pipeline tab to render `+N -M` next to each
// changed file and aggregate totals at the bottom.
func GetStagedNumstat() (map[string]FileNumstat, error) {
	cmd := exec.Command("git", "diff", "--staged", "--numstat")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get numstat: %w", err)
	}
	result := map[string]FileNumstat{}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		var ns FileNumstat
		if parts[0] == "-" {
			ns.Adds = -1
		} else {
			fmt.Sscanf(parts[0], "%d", &ns.Adds)
		}
		if parts[1] == "-" {
			ns.Dels = -1
		} else {
			fmt.Sscanf(parts[1], "%d", &ns.Dels)
		}
		result[parts[2]] = ns
	}
	return result, nil
}

// GetStagedFileDiff returns the staged diff for a single file with --unified=4
// context. Used by the diff popup.
func GetStagedFileDiff(filePath string) (string, error) {
	cmd := exec.Command("git", "diff", "--cached", "--unified=4", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff for %s: %w", filePath, err)
	}
	return string(out), nil
}

// GetCommitDiffSummary returns the diff of a specific commit, structured per
// file, with the same format as GetStagedDiffSummary so it can be fed to the
// same AI prompt. Falls back to diff-tree for the initial commit.
func GetCommitDiffSummary(hash string, maxDiffChars int) (string, error) {
	cmdFiles := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", hash)
	filesOutput, err := cmdFiles.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit file list: %w", err)
	}

	if len(filesOutput) == 0 {
		return "", nil
	}

	files := strings.Split(strings.TrimSpace(string(filesOutput)), "\n")
	var resultBuilder strings.Builder
	currentChars := 0

	for _, file := range files {
		if file == "" {
			continue
		}

		cmdDiff := exec.Command("git", "diff", "--unified=0", hash+"^", hash, "--", file)
		diffBytes, err := cmdDiff.Output()
		if err != nil {
			cmdDiff = exec.Command(
				"git", "diff-tree",
				"-p", "--unified=0", "--no-commit-id", "-r", hash,
				"--", file,
			)
			diffBytes, err = cmdDiff.Output()
			if err != nil {
				continue
			}
		}

		block := fmt.Sprintf("=== %s ===\n%s\n", file, string(diffBytes))
		blockLen := len(block)

		if maxDiffChars > 0 && currentChars+blockLen > maxDiffChars {
			break
		}

		resultBuilder.WriteString(block)
		currentChars += blockLen
	}

	return resultBuilder.String(), nil
}

// ResolveCommitHash expands a partial hash (or any rev-spec git accepts) to a
// full SHA1. Returns an error if the revision does not exist.
func ResolveCommitHash(rev string) (string, error) {
	out, err := exec.Command("git", "rev-parse", "--verify", rev+"^{commit}").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CommitMessage is the resolved subject + body for a single commit hash.
// Either field can be empty (subject usually isn't, but body very often
// is for one-line commits).
type CommitMessage struct {
	Subject string
	Body    string
}

// LookupCommitMessages resolves the subject and body for each hash via
// `git show -s --format=%s%x00%b <hash>` per hash. The returned map is
// keyed by the same strings the caller passed in (full or abbreviated).
// Hashes that don't resolve (rebased away, shallow clones, typos) get a
// zero-value CommitMessage in the map so the UI can distinguish "not
// found" from "no lookup attempted".
//
// Per-hash calls (instead of a single bulk `git log --no-walk <hashes…>`)
// are deliberate: bulk fails entirely the moment one revision is invalid,
// which silently wipes every subject for the whole release. Releases
// usually carry under 50 commits, and `git show -s` is cheap, so the
// per-hash cost is well below user-perceptible.
func LookupCommitMessages(hashes []string) (map[string]CommitMessage, error) {
	out := make(map[string]CommitMessage, len(hashes))
	for _, h := range hashes {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, ok := out[h]; ok {
			continue
		}
		// %s = subject (first line), %x00 = NUL separator, %b = body.
		// NUL is safe — commit messages are 8-bit-clean text and git
		// guarantees they don't contain NUL bytes.
		cmd := exec.Command("git", "show", "-s", "--format=%s%x00%b", h)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			out[h] = CommitMessage{}
			continue
		}
		raw := strings.TrimRight(stdout.String(), "\n")
		parts := strings.SplitN(raw, "\x00", 2)
		msg := CommitMessage{Subject: parts[0]}
		if len(parts) == 2 {
			msg.Body = strings.TrimSpace(parts[1])
		}
		out[h] = msg
	}
	return out, nil
}

// GetLastGitTag returns the most recent tag using natural-version sort order.
// Empty string + nil error means the repo has no tags yet.
func GetLastGitTag() (string, error) {
	out, err := exec.Command(
		"git", "tag",
		"--sort=-v:refname",
	).Output()
	if err != nil {
		return "", err
	}
	tags := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, t := range tags {
		if t = strings.TrimSpace(t); t != "" {
			return t, nil
		}
	}
	return "", nil
}

// HasFileChanges reports whether path has any uncommitted state — staged,
// unstaged, or untracked. Uses `git status --porcelain -- <path>` so a single
// call covers every "the file is dirty" case. Empty output (and no error)
// means the working tree matches HEAD for that path.
func HasFileChanges(path string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--", path)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return false, fmt.Errorf("git status: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// StageFile runs `git add <path>` so the file is included in the next commit.
// Used to bundle CHANGELOG.md updates produced by the AI pipeline together
// with the user's already-staged changes.
func StageFile(path string) error {
	cmd := exec.Command("git", "add", "--", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("git add %s: %w (%s)", path, err, strings.TrimSpace(stderr.String()))
		}
		return fmt.Errorf("git add %s: %w", path, err)
	}
	return nil
}

// RewordCommit changes the commit message of the given hash to newMessage.
// For HEAD it uses git commit --amend; for other commits it uses a
// non-interactive rebase driven by temp scripts that replace the pick line
// and the commit message editor.
func RewordCommit(hash, newMessage string) error {
	headOut, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	headHash := strings.TrimSpace(string(headOut))

	if strings.HasPrefix(headHash, hash) || strings.HasPrefix(hash, headHash) {
		return exec.Command("git", "commit", "--amend", "-m", newMessage).Run()
	}

	shortHash := hash
	if len(hash) > 7 {
		shortHash = hash[:7]
	}

	seqEditor, err := os.CreateTemp("", "cc_seq_*.sh")
	if err != nil {
		return fmt.Errorf("creating temp sequence editor: %w", err)
	}
	fmt.Fprintf(
		seqEditor,
		"#!/bin/sh\nsed -i.bak 's/^pick %s/reword %s/' \"$1\"\n",
		shortHash,
		shortHash,
	)
	seqEditor.Close()
	os.Chmod(seqEditor.Name(), 0o755)
	defer os.Remove(seqEditor.Name())
	defer os.Remove(seqEditor.Name() + ".bak")

	msgFile, err := os.CreateTemp("", "cc_msg_*.txt")
	if err != nil {
		return fmt.Errorf("creating temp message file: %w", err)
	}
	fmt.Fprint(msgFile, newMessage)
	msgFile.Close()
	defer os.Remove(msgFile.Name())

	msgEditor, err := os.CreateTemp("", "cc_editmsg_*.sh")
	if err != nil {
		return fmt.Errorf("creating temp message editor: %w", err)
	}
	fmt.Fprintf(msgEditor, "#!/bin/sh\ncp %s \"$1\"\n", msgFile.Name())
	msgEditor.Close()
	os.Chmod(msgEditor.Name(), 0o755)
	defer os.Remove(msgEditor.Name())

	cmd := exec.Command("git", "rebase", "-i", hash+"^")
	cmd.Env = append(os.Environ(),
		"GIT_SEQUENCE_EDITOR="+seqEditor.Name(),
		"GIT_EDITOR="+msgEditor.Name(),
	)
	return cmd.Run()
}
