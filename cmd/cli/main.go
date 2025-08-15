package main

import (
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
)

func main() {
	log := logger.New()
	log.Info("Starting Commit Crafter application...")

	globalCfg, localCfg, err := config.LoadConfigs()
	if err != nil {
		log.Fatal("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	finalCommitTypes := config.ResolveCommitTypes(globalCfg, localCfg)

	db, err := storage.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database", "error", err)
	}
	defer db.Close()
	log.Debug("Database initialized successfully.")

	initialModel, err := tui.NewModel(log, db, finalCommitTypes)
	if err != nil {
		log.Fatal("Error creating the TUI model", "error", err)
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal("Oh no! There was an error", "error", err)
	}

	if m, ok := finalModel.(*tui.Model); ok && m.FinalMessage != "" {
		fmt.Print(m.FinalMessage)
	}
}
