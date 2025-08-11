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
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Oh no! There was an error: %v\n", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(*tui.Model); ok && m.FinalMessage != "" {
		fmt.Println(m.FinalMessage)
	}
}
