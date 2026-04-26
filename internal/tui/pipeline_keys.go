package tui

import "charm.land/bubbles/v2/key"

// pipelineKeys is the keymap active while the user is on the Pipeline tab.
// `r` retries the full pipeline; `1`/`2`/`3` retry a single stage and
// cascade re-runs to downstream stages where the existing AI runner does
// so already (callIaSummaryCmd → all 3, callIaCommitBuilderStage2Cmd →
// stages 2 + 3, callIaOutputFormatCmd → stage 3 only).
func pipelineKeys() KeyMap {
	return KeyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		NextField: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch panel")),
		Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "accept commit")),
		Esc:       key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel / back")),
		Toggle:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry pipeline")),
		RerunStage1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "retry stage 1 (cascades)"),
		),
		RerunStage2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "retry stage 2 (cascades)"),
		),
		RerunStage3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "retry stage 3"),
		),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
	}
}
