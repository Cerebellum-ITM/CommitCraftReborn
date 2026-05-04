package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"strings"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
)

// addableTagEntry is the wire format for `ai list-addable-tags`. Every
// entry here is a builtin tag that the code knows about and that is
// not yet present in the local `.commitcraft.toml`, so the agent can
// register it with `ai add-tag --tag X` before using it in `generate`.
type addableTagEntry struct {
	Tag         string `json:"tag"`
	Description string `json:"description"`
}

// runListAddableTags prints the builtin commit-type tags from
// `commit.GetAddableCommitTypes` minus the ones already in the local
// config. The output is independent of `commit_types.behavior`
// (append/replace) so an agent can always discover what builtins exist
// even when global/local configs are in `replace` mode.
func runListAddableTags(args []string) int {
	fs := flagSet("ai list-addable-tags")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}

	_, localCfg, err := config.LoadConfigs()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}

	taken := make(map[string]struct{}, len(localCfg.CommitTypes.Types))
	for _, t := range localCfg.CommitTypes.Types {
		taken[strings.ToUpper(t.Tag)] = struct{}{}
	}

	addable := commit.GetAddableCommitTypes()
	entries := make([]addableTagEntry, 0, len(addable))
	for _, a := range addable {
		if _, dup := taken[strings.ToUpper(a.Tag)]; dup {
			continue
		}
		entries = append(entries, addableTagEntry{
			Tag:         a.Tag,
			Description: a.Description,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(entries)
	return 0
}
