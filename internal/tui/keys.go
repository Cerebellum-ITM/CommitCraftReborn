package tui

import "github.com/charmbracelet/bubbles/v2/key"

// KeyMap defines a set of keybindings.
// It implements the help.KeyMap interface.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Quit       key.Binding
	GlobalQuit key.Binding
	Help       key.Binding
	Esc        key.Binding
	Filter     key.Binding
	Logs       key.Binding
}

func listKeys() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:       key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Enter:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Quit:       key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		GlobalQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Logs:       key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "show logs")),
		Filter:     key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Esc:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	}
}

func textInputKeys() KeyMap {
	return KeyMap{
		Enter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Esc:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	b := []key.Binding{}
	if k.Help.Enabled() {
		b = append(b, k.Help)
	}
	if k.Filter.Enabled() {
		b = append(b, k.Filter)
	}
	if k.Quit.Enabled() {
		b = append(b, k.Quit)
	}
	if k.GlobalQuit.Enabled() {
		b = append(b, k.GlobalQuit)
	}
	if k.Enter.Enabled() {
		b = append(b, k.Enter)
	}
	if k.Logs.Enabled() {
		b = append(b, k.Logs)
	}
	if k.Esc.Enabled() {
		b = append(b, k.Esc)
	}
	return b
}

func (k KeyMap) FullHelp() [][]key.Binding {
	b := [][]key.Binding{{}}
	return b
}
