package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"os"

	"commit_craft_reborn/internal/config"
)

// agentFlags holds the three delegate-mode flags shared by generate,
// regenerate, merge, and release. Register with registerAgentFlags so every
// subcommand exposes the same surface.
type agentFlags struct {
	agent    *bool
	noAgent  *bool
	strategy *string
}

// registerAgentFlags wires --agent / --no-agent / --agent-strategy onto a
// flag set, so every delegate-capable subcommand exposes the same surface.
func registerAgentFlags(fs *flag.FlagSet) agentFlags {
	return agentFlags{
		agent: fs.Bool("agent", false,
			"Delegate generation to the calling AI agent (no Groq call). Emits a "+
				"prompt bundle to fulfill, then submit via `ai submit`."),
		noAgent: fs.Bool("no-agent", false,
			"Force the normal Groq pipeline even when agent mode is enabled in config."),
		strategy: fs.String("agent-strategy", "",
			"Delegate prompting strategy: single | staged. Default from config (single)."),
	}
}

// resolveAgentMode decides whether this call runs in delegate mode. Explicit
// flags win over config; --agent and --no-agent together is a usage error.
func resolveAgentMode(cfg config.Config, f agentFlags) (bool, error) {
	if *f.agent && *f.noAgent {
		return false, errors.New("--agent and --no-agent are mutually exclusive")
	}
	if *f.agent {
		return true, nil
	}
	if *f.noAgent {
		return false, nil
	}
	return config.NormalizeAgentMode(cfg.Agent.Mode) == config.AgentModeDelegate, nil
}

// resolveStrategy picks the delegate prompting strategy: the --agent-strategy
// flag if set, else the config default, normalized to single|staged.
func resolveStrategy(cfg config.Config, f agentFlags) string {
	if *f.strategy != "" {
		return config.NormalizeAgentStrategy(*f.strategy)
	}
	return config.NormalizeAgentStrategy(cfg.Agent.Strategy)
}

// printJSON writes any value as indented JSON to stdout — used for delegate
// bundles, which don't fit the commitJSON shape.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
