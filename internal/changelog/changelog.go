// Package changelog detects an existing CHANGELOG.md format, suggests the
// next semantic version, and prepends a freshly produced entry preserving the
// surrounding markdown.
package changelog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Info captures the bits about the existing CHANGELOG that the refiner prompt
// needs to imitate the project's style.
type Info struct {
	// Path is the absolute or repo-relative path actually used to read the
	// file. Reused by the writer so detection and write touch the same file.
	Path string
	// FormatSample contains the first heading + the first existing entry,
	// trimmed to a reasonable size, so the model can mimic the layout.
	FormatSample string
	// LatestVersion is the most recent vX.Y.Z found at the start of any
	// heading. Empty when the changelog uses a non-semver style.
	LatestVersion string
}

var (
	// versionHeadingRe captures the version part of a heading line such as
	// "## v1.2.3 — 2026-04-26" or "## [1.2.3] - 2026-04-26". The optional
	// leading "v" is part of the captured group so the suggested next
	// version preserves the project's prefix style.
	versionHeadingRe = regexp.MustCompile(`(?m)^#{1,6}\s+\[?(v?\d+\.\d+\.\d+)\]?`)
	// semverRe is used by SuggestNextVersion to bump a version string.
	semverRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)
)

// Detect locates the changelog at path (resolved relative to repoRoot when
// not absolute), reads it, and returns sampling info usable by the AI prompt.
// Returns os.ErrNotExist when the file is missing — callers should treat that
// as "no changelog, skip" rather than as a hard failure.
func Detect(repoRoot, path string) (*Info, error) {
	if path == "" {
		path = "CHANGELOG.md"
	}
	full := path
	if !filepath.IsAbs(full) {
		full = filepath.Join(repoRoot, path)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	text := string(data)

	info := &Info{Path: full}
	if m := versionHeadingRe.FindStringSubmatch(text); len(m) > 1 {
		info.LatestVersion = m[1]
	}
	info.FormatSample = sampleFormat(text)
	return info, nil
}

// sampleFormat returns the file's title (if any) plus the first entry block,
// truncated to keep prompts compact.
func sampleFormat(text string) string {
	lines := strings.Split(text, "\n")
	var out []string
	headingsSeen := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isHeading := strings.HasPrefix(trimmed, "#")
		if isHeading {
			// Stop right before the second non-H1 heading — that's the
			// boundary between the latest entry and the previous one.
			if headingsSeen >= 2 {
				break
			}
			headingsSeen++
		}
		out = append(out, line)
		if len(out) > 40 {
			break
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// SuggestNextVersion bumps latest by strategy. Empty latest yields "v0.1.0".
// Unknown strategy falls back to "patch".
func SuggestNextVersion(latest, strategy string) string {
	if latest == "" {
		return "v0.1.0"
	}
	m := semverRe.FindStringSubmatch(latest)
	if len(m) != 4 {
		return latest
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])

	switch strings.ToLower(strategy) {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	default:
		patch++
	}
	prefix := ""
	if strings.HasPrefix(latest, "v") {
		prefix = "v"
	}
	return fmt.Sprintf("%s%d.%d.%d", prefix, major, minor, patch)
}

// Prepend inserts entry into the changelog right after the file's H1 title (or
// at the very top if the file has no H1). It preserves trailing newlines and
// guarantees a blank line between entry and the next existing block.
func Prepend(path, entry string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	original := string(data)

	entry = strings.TrimRight(entry, "\n") + "\n\n"

	insertAt := findInsertOffset(original)
	merged := original[:insertAt] + entry + original[insertAt:]

	return os.WriteFile(path, []byte(merged), 0o644)
}

// findInsertOffset returns the byte offset where a new entry should be
// inserted: just after the H1 title block (and any intro paragraph immediately
// below it), or 0 when no H1 is present.
func findInsertOffset(text string) int {
	lines := strings.SplitAfter(text, "\n")
	offset := 0
	titleSeen := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		isHeading := strings.HasPrefix(trimmed, "#")
		if isHeading {
			if !titleSeen && strings.HasPrefix(trimmed, "# ") {
				titleSeen = true
				offset += len(line)
				continue
			}
			// Stop at the first sub-heading after the title — that's the
			// previous latest entry, we insert before it.
			if titleSeen {
				return offset
			}
			// No H1 at all — insert at top.
			return 0
		}
		if titleSeen {
			offset += len(line)
			continue
		}
		offset += len(line)
	}
	if !titleSeen {
		return 0
	}
	return offset
}
