package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	charmlog "charm.land/log/v2"

	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui"
	"commit_craft_reborn/internal/tui/styles"
)

var version = "v0.34.2"

func main() {
	log := logger.New()
	log.Info("Starting Commit Crafter application...")
	startInReleaseMode := flag.Bool(
		"r",
		false,
		"Start the application in release choosing release mode",
	)
	directOutput := flag.Bool(
		"o",
		false,
		"Output the commit message directly to stdout without showing options popup",
	)
	rewordHash := flag.String(
		"w",
		"",
		"Start the application directly in reword mode for the given commit hash",
	)
	flag.Parse()

	globalCfg, localCfg, err := config.LoadConfigs()
	if err != nil {
		log.Fatal("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	finalCommitTypes := config.ResolveCommitTypes(globalCfg, localCfg)
	config.PopulateCommitTypePalettes(&globalCfg, finalCommitTypes)
	registerCommitTypePalettes(globalCfg.CommitFormat.CommitTypePalettes)
	config.ResolveReleaseConfig(&globalCfg, localCfg)
	config.ResolveTUIConfig(&globalCfg, localCfg)

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

	// Hydrate the rate-limit cache from disk so the per-model bars in the
	// compose tab and picker footer don't read "no data yet" right after
	// startup. Failures are non-fatal — bars simply stay empty until the
	// next live API call refreshes them.
	if persisted, err := db.LoadAllModelRateLimits(); err != nil {
		log.Warn("Failed to load persisted rate-limits", "error", err)
	} else {
		for _, p := range persisted {
			api.RecordRateLimits(p.ModelID, api.RateLimits{
				LimitRequests:     p.LimitRequests,
				RemainingRequests: p.RemainingRequests,
				ResetRequests:     time.Duration(p.ResetRequestsMs) * time.Millisecond,
				LimitTokens:       p.LimitTokens,
				RemainingTokens:   p.RemainingTokens,
				ResetTokens:       time.Duration(p.ResetTokensMs) * time.Millisecond,
				CapturedAt:        p.CapturedAt,
				RequestsParsed:    p.RequestsParsed,
				TokensParsed:      p.TokensParsed,
				RequestsToday:     p.RequestsToday,
				RequestsDay:       p.RequestsDay,
			})
		}
	}

	appMode := tui.CommitMode
	if *startInReleaseMode {
		appMode = tui.ReleaseMode
	}

	initialModel, err := tui.NewModel(
		log,
		db,
		globalCfg,
		finalCommitTypes,
		pwd,
		appMode,
		version,
		*directOutput,
		*rewordHash,
	)
	if err != nil {
		log.Fatal("Error creating the TUI model", "error", err)
	}

	p := tea.NewProgram(initialModel, tea.WithOutput(os.Stderr))

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal("Oh no! There was an error", "error", err)
	}

	if m, ok := finalModel.(*tui.Model); ok {
		if m.AutodraftedID != 0 {
			printAutodraftNotice(m.AutodraftedID, m.AutodraftedTab, m.Theme)
		}
		if m.RewordHash != "" {
			if err := git.RewordCommit(m.RewordHash, m.FinalMessage); err != nil {
				fmt.Fprintf(os.Stderr, "Error rewording commit: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Commit %s reworded successfully.\n", m.RewordHash[:7])
		} else if m.FinalMessage != "" {
			fmt.Print(m.FinalMessage)
		}
	}
}

// printAutodraftNotice writes a single charm-log INFO line to stderr after
// the TUI has torn down, telling the user that a draft was auto-saved on
// the way out. The git glyph comes from the theme's symbol table so it
// follows the same nerd-font/no-nerd-font branch as the rest of the UI.
// The brand red stays hardcoded because it is a Git-logo color, not a
// theme concern.
func printAutodraftNotice(draftID int, tabLabel string, theme *styles.Theme) {
	styledIcon := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F05032")). // official Git red
		Render(theme.AppSymbols().Git)
	l := charmlog.NewWithOptions(os.Stderr, charmlog.Options{
		ReportTimestamp: true,
		TimeFormat:      time.Kitchen,
	})
	l.Info(
		fmt.Sprintf("%s  Exit in %s — draft saved", styledIcon, tabLabel),
		"draft_id", draftID,
	)
}

// registerCommitTypePalettes adapts the config-side palette mirror (raw
// hex strings) into the anonymous-struct shape `styles` expects, keeping
// the styles package free of any config-package import.
func registerCommitTypePalettes(palettes map[string]config.CommitTypePalette) {
	if len(palettes) == 0 {
		styles.RegisterCustomCommitTypePalettes(nil)
		return
	}
	out := make(map[string]struct {
		BgBlock, FgBlock, BgMsg, FgMsg string
	}, len(palettes))
	for tag, p := range palettes {
		out[tag] = struct {
			BgBlock, FgBlock, BgMsg, FgMsg string
		}{p.BgBlock, p.FgBlock, p.BgMsg, p.FgMsg}
	}
	styles.RegisterCustomCommitTypePalettes(out)
}
