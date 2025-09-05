package tui

import "github.com/charmbracelet/bubbles/v2/key"

// KeyMap defines a set of keybindings.
// It implements the help.KeyMap interface.
type KeyMap struct {
	Up             key.Binding
	Down           key.Binding
	Left           key.Binding
	Right          key.Binding
	Enter          key.Binding
	Delete         key.Binding
	Quit           key.Binding
	GlobalQuit     key.Binding
	Help           key.Binding
	Esc            key.Binding
	Filter         key.Binding
	Logs           key.Binding
	AddCommit      key.Binding
	NextField      key.Binding
	PrevField      key.Binding
	CreateIaCommit key.Binding
}

func writingMessageKeys() KeyMap {
	return KeyMap{
		NextField: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next field"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "previous field"),
		),
		CreateIaCommit: key.NewBinding(
			key.WithKeys("ctrl+w"),
			key.WithHelp("ctrl+w", "Create the commit using AI"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "Accept AI suggestion"),
		),
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
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
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func mainListKeys() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Delete:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "Delete")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		// Logs:       key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "show logs")),
		Filter: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		// Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		AddCommit: key.NewBinding(
			key.WithKeys("n", "tab"),
			key.WithHelp("Tab/n", "Create a new commit"),
		),
	}
}

func listKeys() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func textInputKeys() KeyMap {
	return KeyMap{
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
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
	if k.CreateIaCommit.Enabled() {
		b = append(b, k.CreateIaCommit)
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
	if k.Esc.Enabled() {
		b = append(b, k.Esc)
	}
	if k.Filter.Enabled() {
		b = append(b, k.Filter)
	}
	if k.Quit.Enabled() {
		b = append(b, k.Quit)
	}
	if k.Delete.Enabled() {
		b = append(b, k.Delete)
	}
	if k.GlobalQuit.Enabled() {
		b = append(b, k.GlobalQuit)
	}
	if k.Logs.Enabled() {
		b = append(b, k.Logs)
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
	if k.Help.Enabled() {
		b = append(b, k.Help)
	}
	return b
}

func (k KeyMap) FullHelp() [][]key.Binding {
	b := [][]key.Binding{{}}
	return b
}
