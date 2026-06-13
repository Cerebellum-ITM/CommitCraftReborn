package config

import "strings"

// Agent delegate-mode vocabulary. Mode selects whether the headless `ai`
// subcommands call Groq ("groq", the default) or emit a prompt bundle for the
// calling AI agent to fulfill ("delegate"). Strategy picks the prompting shape
// in delegate mode.
const (
	AgentModeGroq       = "groq"
	AgentModeDelegate   = "delegate"
	AgentStrategySingle = "single"
	AgentStrategyStaged = "staged"
)

// NormalizeAgentMode maps arbitrary input to a valid mode, defaulting to
// "groq" so any unset/unknown value keeps the normal Groq pipeline. Only the
// literal "delegate" (case-insensitive) opts into delegate mode.
func NormalizeAgentMode(mode string) string {
	if strings.EqualFold(strings.TrimSpace(mode), AgentModeDelegate) {
		return AgentModeDelegate
	}
	return AgentModeGroq
}

// NormalizeAgentStrategy maps arbitrary input to a valid strategy, defaulting
// to "single" (one unified prompt, one inference). Only the literal "staged"
// (case-insensitive) selects the per-stage prompt bundle.
func NormalizeAgentStrategy(strategy string) string {
	if strings.EqualFold(strings.TrimSpace(strategy), AgentStrategyStaged) {
		return AgentStrategyStaged
	}
	return AgentStrategySingle
}
