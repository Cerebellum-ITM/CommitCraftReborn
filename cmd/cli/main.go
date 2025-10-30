package main

import (
	"flag"
	"fmt"
	"os"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui"

	tea "github.com/charmbracelet/bubbletea/v2"
)

func main() {
	log := logger.New()
	log.Info("Starting Commit Crafter application...")
	startInReleaseMode := flag.Bool(
		"r",
		false,
		"Start the application in release choosing release mode",
	)
	flag.Parse()

	globalCfg, localCfg, err := config.LoadConfigs()
	if err != nil {
		log.Fatal("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	finalCommitTypes := config.ResolveCommitTypes(globalCfg, localCfg)
	config.PopulateCommitTypeColors(&globalCfg, finalCommitTypes)
	config.ResolveReleaseConfig(&globalCfg, localCfg)

	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal("Error trying to get the current directory", "error", err)
		os.Exit(1)
	}

	db, err := storage.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database", "error", err)
	}
	defer db.Close()
	log.Debug("Database initialized successfully.")

	appMode := tui.CommitMode
	if *startInReleaseMode {
		appMode = tui.ReleaseMode
	}

	initialModel, err := tui.NewModel(log, db, globalCfg, finalCommitTypes, pwd, appMode)
	if err != nil {
		log.Fatal("Error creating the TUI model", "error", err)
	}

	p := tea.NewProgram(initialModel, tea.WithOutput(os.Stderr))

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal("Oh no! There was an error", "error", err)
	}

	if m, ok := finalModel.(*tui.Model); ok && m.FinalMessage != "" {
		fmt.Print(m.FinalMessage)
	}
}
