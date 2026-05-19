package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"commit_craft_reborn/internal/git"
)

// ReleaseDetect holds the auto-detected defaults that pre-fill the release
// configuration popup. All fields are advisory: every individual call is
// best-effort and returns an empty value on failure, so the popup keeps
// working on git-less or freshly-cloned trees.
type ReleaseDetect struct {
	Repository       string // owner/repo, parsed from `git remote get-url origin`
	Branch           string // current branch's short name
	LastTag          string // most recent tag, if any
	SuggestedVersion string // BumpVersionPatch(LastTag) or "v0.1.0"
	AssetsPath       string // first of "bin", "build", "dist" that exists, else ""
	GhTokenSet       bool   // does the env already have GH_TOKEN?
	BuildTool        string // "make" when a Makefile is present, else ""
	BuildTarget      string // detected Makefile target (build_release / build / release)
}

// DetectRelease runs the read-only detection probes against `pwd` and
// returns whatever it could find. Never errors — empty strings indicate
// "couldn't detect; ask the user to fill it in".
func DetectRelease(pwd string) ReleaseDetect {
	d := ReleaseDetect{
		Repository: detectGithubRepository(pwd),
		AssetsPath: detectAssetsDir(pwd),
		GhTokenSet: os.Getenv("GH_TOKEN") != "",
	}
	if branch, err := git.GetCurrentGitBranch(); err == nil {
		d.Branch = branch
	}
	if tag, err := git.GetLastGitTag(); err == nil {
		d.LastTag = tag
	}
	d.SuggestedVersion = BumpVersionPatch(d.LastTag)
	if d.SuggestedVersion == "" {
		d.SuggestedVersion = "v0.1.0"
	}
	d.BuildTool, d.BuildTarget = detectMakefileTarget(pwd)
	return d
}

// makefileTargetRegex captures the leading word of any Makefile target
// declaration (`name:` or `name :` at the start of a line). Used by
// detectMakefileTarget to surface candidate build targets for the
// release config popup.
var makefileTargetRegex = regexp.MustCompile(`(?m)^([A-Za-z0-9_.-]+)\s*:`)

// detectMakefileTarget scans a Makefile in `pwd` and returns the build
// tool name plus the most likely build target. Preference order, from
// strongest to weakest match: `build_release`, `release`, `build`. If no
// preferred target is present but the Makefile exists, return tool
// "make" with empty target so the user can fill it in. If no Makefile
// at all, return ("", "").
func detectMakefileTarget(pwd string) (string, string) {
	path := filepath.Join(pwd, "Makefile")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	targets := map[string]bool{}
	for _, m := range makefileTargetRegex.FindAllStringSubmatch(string(data), -1) {
		if len(m) >= 2 {
			targets[strings.ToLower(m[1])] = true
		}
	}
	for _, preferred := range []string{"build_release", "release", "build"} {
		if targets[preferred] {
			return "make", preferred
		}
	}
	return "make", ""
}

// repoRegex captures the trailing `owner/repo` portion of any github URL,
// stripping a leading `git@github.com:` SSH prefix, an `https://github.com/`
// HTTP prefix, and the optional `.git` suffix. Only github.com is matched
// — the upload helper only knows how to talk to gh, so other hosts get
// no auto-detect.
var repoRegex = regexp.MustCompile(`(?i)github\.com[:/]+([^/\s]+/[^/\s]+?)(?:\.git)?$`)

// detectGithubRepository runs `git -C pwd remote get-url origin` and
// parses the URL into `owner/repo`. Returns "" if there is no remote,
// the remote is not on github.com, or the URL format is unrecognised.
func detectGithubRepository(pwd string) string {
	cmd := exec.Command("git", "-C", pwd, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	m := repoRegex.FindStringSubmatch(url)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// detectAssetsDir returns the first directory among ("bin", "build",
// "dist") that exists inside pwd. Empty string if none of them exist —
// the popup will leave the assets path blank and the user can either
// type one in or leave it empty (notes-only release).
func detectAssetsDir(pwd string) string {
	for _, candidate := range []string{"bin", "build", "dist"} {
		p := filepath.Join(pwd, candidate)
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}
