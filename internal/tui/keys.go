package tui

import "charm.land/bubbles/v2/key"

// KeyMap defines a set of keybindings.
// It implements the help.KeyMap interface.
type KeyMap struct {
	// Navigation
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	PgUp       key.Binding
	PgDown     key.Binding
	NextField  key.Binding
	PrevField  key.Binding
	SwitchMode key.Binding

	// Release Workflow
	NextViewPort key.Binding

	// General
	Enter          key.Binding
	Delete         key.Binding
	Quit           key.Binding
	GlobalQuit     key.Binding
	Toggle         key.Binding
	Help           key.Binding
	Esc            key.Binding
	Filter         key.Binding
	Logs           key.Binding
	AddCommit      key.Binding
	AddCommitKey   key.Binding
	CreateIaCommit key.Binding
	SaveDraft      key.Binding
	Edit           key.Binding
	EditIaCommit   key.Binding
	ReleaseCommit  key.Binding
	ToggleDrafts   key.Binding
	SwapMode       key.Binding
	CycleNext      key.Binding
	CyclePrev      key.Binding

	// TextArea
	insertLine      key.Binding
	delteLine       key.Binding
	deleteForward   key.Binding
	deleteBackwards key.Binding

	// Templates
	CreateLocalTomlConfig key.Binding

	// Pipeline tab
	SwitchTab   key.Binding
	RerunStage1 key.Binding
	RerunStage2 key.Binding
	RerunStage3 key.Binding
	RerunStage4 key.Binding
	FileUp      key.Binding
	FileDown    key.Binding
}

func writingMessageKeys() KeyMap {
	return KeyMap{
		SaveDraft: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "Save draft"),
		),
		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "Switch Focus →"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "Switch Focus ←"),
		),
		CreateIaCommit: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "Create the commit using AI"),
		),
		Edit: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "Edit Ia Respone"),
		),
		AddCommitKey: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "Add key point"),
		),

		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "Accept AI suggestion"),
		),
		insertLine: key.NewBinding(
			key.WithKeys("alt+tab", "insert"),
			key.WithHelp("alt+tab or insert", "add new line"),
		),
		Esc:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		PgUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "Scroll keypoints up"),
		),
		PgDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdown", "Scroll keypoints down"),
		),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
		SwitchTab: key.NewBinding(
			key.WithKeys("ctrl+t"),
			key.WithHelp("ctrl+t", "Switch tab"),
		),
		RerunStage1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "Re-run Stage 1+"),
			key.WithDisabled(),
		),
		RerunStage2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "Re-run Stage 2+"),
			key.WithDisabled(),
		),
		RerunStage3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "Re-run Stage 3"),
			key.WithDisabled(),
		),
	}
}

func fileListKeys() KeyMap {
	return KeyMap{
		Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left: key.NewBinding(
			key.WithKeys("left", "shift+tab"),
			key.WithHelp("←/sft+tab", "Parent dir"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "tab"),
			key.WithHelp("→/tab", "Enter to dir"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "Toggle show only Modified files"),
		),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func mainListKeys() KeyMap {
	return KeyMap{
		Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Delete:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "Delete")),
		Quit:         key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit:   key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
		EditIaCommit: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "Edit commit")),
		// Logs:       key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "show logs")),
		ReleaseCommit: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "Create a release")),
		Filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		// Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		CreateLocalTomlConfig: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "Create local config template file"),
		),
		AddCommit: key.NewBinding(
			key.WithKeys("n", "tab"),
			key.WithHelp("Tab/n", "Create a new commit"),
		),
		ToggleDrafts: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "Toggle drafts view"),
		),
		SwitchMode: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "Switch Mode"),
		),
		SwapMode: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "Swap inspect mode"),
		),
		CycleNext: key.NewBinding(
			key.WithKeys("ctrl+]"),
			key.WithHelp("ctrl+]", "Next stage"),
		),
		CyclePrev: key.NewBinding(
			key.WithKeys("ctrl+["),
			key.WithHelp("ctrl+[", "Prev stage"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "More keys"),
		),
	}
}

func releaseMainListKeys() KeyMap {
	return KeyMap{
		ReleaseCommit: key.NewBinding(
			key.WithKeys("r", "tab"),
			key.WithHelp("r/tab", "Create a release"),
		),
		Up:           key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:         key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Delete:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "Delete")),
		Quit:         key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit:   key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
		Help:         key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		EditIaCommit: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "Edit commit")),
		Filter:       key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		SwitchMode: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "Switch Mode"),
		),
	}
}

func releaseKeys() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
		Down:       key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		AddCommit: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("Ctrl+a", "Add the commit"),
		),

		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "Switch Focus →"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "Switch Focus ←"),
		),
		NextViewPort: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("ctrl+q", "Togle ia response / preview"),
			key.WithDisabled(),
		),
	}
}

func viewPortKeys() KeyMap {
	return KeyMap{
		PgUp:   key.NewBinding(key.WithKeys("pgup"), key.WithHelp("pgup", "page up")),
		PgDown: key.NewBinding(key.WithKeys("pgdown"), key.WithHelp("pgdown", "page down")),
		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "Switch Focus →"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "Switch Focus ←"),
		),
		Quit: key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		NextViewPort: key.NewBinding(
			key.WithKeys("ctrl+q"),
			key.WithHelp("ctrl+q", "Togle ia response / preview"),
			key.WithDisabled(),
		),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
	}
}

func popupKeys() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Accept")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func listKeys() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func rewordSelectKeys() KeyMap {
	return KeyMap{
		Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "Reword selected commit"),
		),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
	}
}

func textInputKeys() KeyMap {
	return KeyMap{
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+x"), key.WithHelp("ctrl+x", "quit")),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	b := []key.Binding{}
	if k.AddCommit.Enabled() {
		b = append(b, k.AddCommit)
	}
	if k.AddCommitKey.Enabled() {
		b = append(b, k.AddCommitKey)
	}
	if k.CreateIaCommit.Enabled() {
		b = append(b, k.CreateIaCommit)
	}
	if k.Edit.Enabled() {
		b = append(b, k.Edit)
	}
	if k.insertLine.Enabled() {
		b = append(b, k.insertLine)
	}
	if k.Enter.Enabled() {
		b = append(b, k.Enter)
	}
	if k.Left.Enabled() {
		b = append(b, k.Left)
	}
	if k.Right.Enabled() {
		b = append(b, k.Right)
	}
	if k.Toggle.Enabled() {
		b = append(b, k.Toggle)
	}
	if k.Esc.Enabled() {
		b = append(b, k.Esc)
	}
	if k.PgUp.Enabled() {
		b = append(b, k.PgUp)
	}
	if k.PgDown.Enabled() {
		b = append(b, k.PgDown)
	}
	if k.Filter.Enabled() {
		b = append(b, k.Filter)
	}
	if k.Delete.Enabled() {
		b = append(b, k.Delete)
	}
	if k.Logs.Enabled() {
		b = append(b, k.Logs)
	}
	if k.NextField.Enabled() {
		b = append(b, k.NextField)
	}
	if k.Help.Enabled() {
		b = append(b, k.Help)
	}
	if k.GlobalQuit.Enabled() {
		b = append(b, k.GlobalQuit)
	}
	if k.SwitchTab.Enabled() {
		b = append(b, k.SwitchTab)
	}
	if k.RerunStage1.Enabled() {
		b = append(b, k.RerunStage1)
	}
	if k.RerunStage2.Enabled() {
		b = append(b, k.RerunStage2)
	}
	if k.RerunStage3.Enabled() {
		b = append(b, k.RerunStage3)
	}
	return b
}

func (k KeyMap) FullHelp() [][]key.Binding {
	b := []key.Binding{}
	if k.Up.Enabled() {
		b = append(b, k.Up)
	}
	if k.Down.Enabled() {
		b = append(b, k.Down)
	}
	if k.Left.Enabled() {
		b = append(b, k.Left)
	}
	if k.Right.Enabled() {
		b = append(b, k.Right)
	}
	if k.AddCommitKey.Enabled() {
		b = append(b, k.AddCommitKey)
	}
	if k.EditIaCommit.Enabled() {
		b = append(b, k.EditIaCommit)
	}
	if k.ReleaseCommit.Enabled() {
		b = append(b, k.ReleaseCommit)
	}
	if k.Enter.Enabled() {
		b = append(b, k.Enter)
	}
	if k.Delete.Enabled() {
		b = append(b, k.Delete)
	}
	if k.Quit.Enabled() {
		b = append(b, k.Quit)
	}
	if k.GlobalQuit.Enabled() {
		b = append(b, k.GlobalQuit)
	}
	if k.Toggle.Enabled() {
		b = append(b, k.Toggle)
	}
	if k.NextViewPort.Enabled() {
		b = append(b, k.NextViewPort)
	}
	if k.Help.Enabled() {
		b = append(b, k.Help)
	}
	if k.Esc.Enabled() {
		b = append(b, k.Esc)
	}
	if k.Filter.Enabled() {
		b = append(b, k.Filter)
	}
	if k.deleteBackwards.Enabled() {
		b = append(b, k.deleteBackwards)
	}
	if k.deleteForward.Enabled() {
		b = append(b, k.deleteForward)
	}
	if k.Logs.Enabled() {
		b = append(b, k.Logs)
	}
	if k.AddCommit.Enabled() {
		b = append(b, k.AddCommit)
	}
	if k.NextField.Enabled() {
		b = append(b, k.NextField)
	}
	if k.PrevField.Enabled() {
		b = append(b, k.PrevField)
	}
	if k.CreateIaCommit.Enabled() {
		b = append(b, k.CreateIaCommit)
	}
	if k.Edit.Enabled() {
		b = append(b, k.Edit)
	}
	if k.SwitchMode.Enabled() {
		b = append(b, k.SwitchMode)
	}
	if k.CreateLocalTomlConfig.Enabled() {
		b = append(b, k.CreateLocalTomlConfig)
	}
	if k.SwitchTab.Enabled() {
		b = append(b, k.SwitchTab)
	}
	if k.RerunStage1.Enabled() {
		b = append(b, k.RerunStage1)
	}
	if k.RerunStage2.Enabled() {
		b = append(b, k.RerunStage2)
	}
	if k.RerunStage3.Enabled() {
		b = append(b, k.RerunStage3)
	}
	return [][]key.Binding{b}
}
