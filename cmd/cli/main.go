package main

import (
	"commit_craft_reborn/internal/tui"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	initialModel, err := tui.NewModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing model: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Oh no! There was an error running the program: %v\n", err)
		os.Exit(1)
	}
}
