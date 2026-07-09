package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// initRepo creates a throwaway git repo in a temp dir and returns its path.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func gitAdd(t *testing.T, dir, file string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "add", "--", file)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add %s: %v\n%s", file, err, out)
	}
}

// TestStagedDiffSummary_LargeDiffIsTruncatedNotDropped reproduces the
// no_staged_diff bug: a single staged file whose diff exceeds the cap used to
// be dropped wholesale, yielding an empty summary that the CLI misreported as
// "no staged changes". The fix truncates the block instead, so the summary is
// non-empty and the truncated flag is set.
func TestStagedDiffSummary_LargeDiffIsTruncatedNotDropped(t *testing.T) {
	dir := initRepo(t)

	// ~1 MB of staged additions, well over the 80 KB cap and the historical
	// 64 KB scanner limit named in the bug report.
	var b strings.Builder
	line := "msgid \"the quick brown fox jumps over the lazy dog\"\n"
	for b.Len() < 1_000_000 {
		b.WriteString(line)
	}
	big := filepath.Join(dir, "large.po")
	if err := os.WriteFile(big, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAdd(t, dir, "large.po")

	const maxBytes = 80_000
	diff, truncated, err := stagedDiffSummary(dir, maxBytes)
	if err != nil {
		t.Fatalf("stagedDiffSummary: %v", err)
	}

	if strings.TrimSpace(diff) == "" {
		t.Fatal("summary is empty for a large staged diff — would be misreported as no_staged_diff")
	}
	if !truncated {
		t.Error("expected truncated=true for a diff larger than the cap")
	}
	if len(diff) > maxBytes {
		t.Errorf("summary is %d bytes, exceeds cap of %d", len(diff), maxBytes)
	}
	if !utf8.ValidString(diff) {
		t.Error("truncated summary is not valid UTF-8")
	}
	if !strings.HasPrefix(diff, "=== large.po ===") {
		t.Errorf("summary missing file header, got prefix %q", diff[:min(40, len(diff))])
	}
}

// TestStagedDiffSummary_NothingStaged keeps the genuine no_staged_diff path:
// an empty stage returns an empty summary and truncated=false.
func TestStagedDiffSummary_NothingStaged(t *testing.T) {
	dir := initRepo(t)
	diff, truncated, err := stagedDiffSummary(dir, 80_000)
	if err != nil {
		t.Fatalf("stagedDiffSummary: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty summary, got %q", diff)
	}
	if truncated {
		t.Error("expected truncated=false when nothing is staged")
	}
}

// TestStagedDiffSummary_SmallDiffNotTruncated confirms the common case is
// untouched: a small staged diff is returned whole with truncated=false.
func TestStagedDiffSummary_SmallDiffNotTruncated(t *testing.T) {
	dir := initRepo(t)
	f := filepath.Join(dir, "small.txt")
	if err := os.WriteFile(f, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitAdd(t, dir, "small.txt")

	diff, truncated, err := stagedDiffSummary(dir, 80_000)
	if err != nil {
		t.Fatalf("stagedDiffSummary: %v", err)
	}
	if truncated {
		t.Error("small diff should not be truncated")
	}
	if !strings.Contains(diff, "=== small.txt ===") {
		t.Errorf("summary missing file block: %q", diff)
	}
}
