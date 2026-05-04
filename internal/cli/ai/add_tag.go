package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
)

// addTagResult is the JSON wire format for `ai add-tag`. Added/Skipped
// echo the tags as supplied so the caller can correlate against the
// CLI flags it passed in.
type addTagResult struct {
	Added      []string `json:"added"`
	Skipped    []string `json:"skipped"`
	ConfigPath string   `json:"config_path"`
}

// runAddTag appends one or more builtin commit-type tags (those listed
// by `commit.GetAddableCommitTypes`) to the workspace's
// `.commitcraft.toml`, creating the file from the default template if
// it doesn't exist yet. Tags already present are reported under
// `skipped`. Validation rejects unknown tags so an agent can't smuggle
// in a custom tag without a palette — the TUI's tag picker enforces
// the same rule.
func runAddTag(args []string) int {
	fs := flagSet("ai add-tag")
	var tags stringSlice
	fs.Var(&tags, "tag", "Builtin tag to add to the local config (repeatable)")
	fs.Var(&tags, "t", "Shorthand for --tag")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}
	if len(tags) == 0 {
		printErrorJSON("invalid_input", "at least one --tag is required")
		return 2
	}

	addable := commit.GetAddableCommitTypes()
	byTag := make(map[string]commit.CommitType, len(addable))
	for _, a := range addable {
		byTag[strings.ToUpper(a.Tag)] = a
	}

	resolved := make([]commit.CommitType, 0, len(tags))
	supplied := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, raw := range tags {
		key := strings.ToUpper(strings.TrimSpace(raw))
		if key == "" {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		t, ok := byTag[key]
		if !ok {
			printErrorJSON(
				"invalid_input",
				fmt.Sprintf(
					"unknown builtin tag %q — run `commitcraft ai list-tags` to see addable tags",
					raw,
				),
			)
			return 2
		}
		resolved = append(resolved, t)
		supplied = append(supplied, t.Tag)
	}

	// Ensure the local config file exists first so the snapshot reflects
	// any template-seeded entries (the default `.commitcraft.toml`
	// template ships with the addable tags as examples). Otherwise a
	// fresh-bootstrap run would mis-classify those template entries as
	// `added` when they actually came from the scaffold.
	if err := config.CreateLocalConfigTomlTmpl(); err != nil {
		printErrorJSON("config_write_error", err.Error())
		return 1
	}
	preExisting := loadLocalTagSet()

	if _, err := config.AppendCommitTypesToLocalConfig(resolved); err != nil {
		printErrorJSON("config_write_error", err.Error())
		return 1
	}

	added := make([]string, 0, len(supplied))
	skipped := make([]string, 0)
	for _, tag := range supplied {
		if _, dup := preExisting[strings.ToUpper(tag)]; dup {
			skipped = append(skipped, tag)
			continue
		}
		added = append(added, tag)
	}

	path, err := config.LocalConfigPath()
	if err != nil {
		printErrorJSON("config_write_error", err.Error())
		return 1
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(addTagResult{
		Added:      added,
		Skipped:    skipped,
		ConfigPath: path,
	})
	return 0
}

// loadLocalTagSet reads the workspace `.commitcraft.toml` (if present)
// and returns its commit-type tags as an upper-cased set. Errors and
// missing files yield an empty set — the caller treats that as "nothing
// pre-existed", which is correct for both cases.
func loadLocalTagSet() map[string]struct{} {
	out := map[string]struct{}{}
	_, localCfg, err := config.LoadConfigs()
	if err != nil {
		return out
	}
	for _, t := range localCfg.CommitTypes.Types {
		out[strings.ToUpper(t.Tag)] = struct{}{}
	}
	return out
}
