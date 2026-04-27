package tui

import "charm.land/bubbles/v2/key"

// pipelineKeys is the keymap active while the user is on the Pipeline tab.
//
//   - `r`         — full pipeline retry
//   - `1`/`2`/`3` — per-stage retry (cascades downstream where the runner
//     supports it: 1 → all, 2 → 2+3, 3 → 3 only)
//   - `tab`       — cycle the focused stage (s1 → s2 → s3 → s1)
//   - `pgup/pgdn` — scroll the focused stage's viewport
//   - `↑`/`↓`     — scroll the diff sub-block (always, regardless of focus)
//   - `j`/`k`     — move the changed-files cursor (loads its diff)
//   - `enter`     — accept the assembled commit (only when allDone)
//   - `esc`       — cancel a running run
func pipelineKeys() KeyMap {
	return KeyMap{
		Up:   key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "diff up")),
		Down: key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "diff down")),
		PgUp: key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "stage scroll up")),
		PgDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "stage scroll down"),
		),
		FileUp:    key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "prev file")),
		FileDown:  key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "next file")),
		NextField: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "cycle stage focus")),
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
		RerunStage4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "retry changelog refiner"),
		),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
	}
}
