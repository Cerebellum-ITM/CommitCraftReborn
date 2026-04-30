package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"os"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
)

// tagEntry is the wire format for `ai list-tags`. Source tracks where
// the tag came from (default / global / local) so an AI agent can
// understand why a tag exists and bias its choice toward the
// project-specific ones.
type tagEntry struct {
	Tag         string `json:"tag"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

// runListTags prints the resolved set of commit-type tags as a JSON
// array. The merge follows the same precedence rules as
// config.ResolveCommitTypes (defaults → global → local), but we
// re-implement it here so we can stamp each entry with its origin.
func runListTags(args []string) int {
	fs := flagSet("ai list-tags")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}

	globalCfg, localCfg, err := config.LoadConfigs()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}

	var entries []tagEntry
	seen := map[string]int{} // tag → index in entries (for replace semantics)

	add := func(t commit.CommitType, source string) {
		if t.Tag == "" {
			return
		}
		if idx, ok := seen[t.Tag]; ok {
			entries[idx] = tagEntry{Tag: t.Tag, Description: t.Description, Source: source}
			return
		}
		seen[t.Tag] = len(entries)
		entries = append(entries, tagEntry{
			Tag:         t.Tag,
			Description: t.Description,
			Source:      source,
		})
	}
	clear := func() {
		entries = nil
		seen = map[string]int{}
	}

	// Defaults always seed the list.
	for _, d := range commit.GetDefaultCommitTypes() {
		add(d, "default")
	}

	// Global config: append unless behavior == "replace".
	if globalCfg.CommitTypes.Behavior == "replace" && len(globalCfg.CommitTypes.Types) > 0 {
		clear()
	}
	for _, t := range globalCfg.CommitTypes.Types {
		add(commit.CommitType{Tag: t.Tag, Description: t.Description}, "global")
	}

	// Local config: same rules as global, applied last.
	if localCfg.CommitTypes.Behavior == "replace" && len(localCfg.CommitTypes.Types) > 0 {
		clear()
	}
	for _, t := range localCfg.CommitTypes.Types {
		add(commit.CommitType{Tag: t.Tag, Description: t.Description}, "local")
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(entries)
	return 0
}
