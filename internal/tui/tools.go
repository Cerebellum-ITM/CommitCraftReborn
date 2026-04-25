package tui

import (
	"image/color"
	"os/exec"
	"runtime"

	"commit_craft_reborn/internal/tui/styles"
)

// ToolInfo describes one external CLI dependency (xclip, gh, …) and how it
// should render in the TUI when present or missing.
type ToolInfo struct {
	name      string
	available bool
	textColor color.Color
	icon      string
}

// Tools groups the external CLI deps the TUI cares about.
type Tools struct {
	xclip ToolInfo
	gh    ToolInfo
}

// CheckTools probes the system for the external CLIs we depend on
// (clipboard tool and the GitHub CLI), returning per-tool availability
// info plus theme-aware icon and color choices.
func CheckTools(theme styles.Theme) Tools {
	tools := Tools{}
	var icon string
	var textColor color.Color

	osType := runtime.GOOS
	clipCommand := "xclip"
	if osType == "darwin" {
		clipCommand = "pbcopy"
	}
	_, err := exec.LookPath(clipCommand)
	toolAvailable := err == nil

	if toolAvailable {
		icon = theme.AppSymbols().ClipboardEnable
		textColor = theme.Accent
	} else {
		icon = theme.AppSymbols().ClipboardMissing
		textColor = theme.Purple
	}

	tools.xclip = ToolInfo{
		name:      clipCommand,
		available: toolAvailable,
		icon:      icon,
		textColor: textColor,
	}

	_, err = exec.LookPath("gh")
	toolAvailable = err == nil

	if toolAvailable {
		icon = theme.AppSymbols().GhEnable
		textColor = theme.Success
	} else {
		icon = theme.AppSymbols().GhMissing
		textColor = theme.Purple
	}

	tools.gh = ToolInfo{
		name:      clipCommand,
		available: toolAvailable,
		icon:      icon,
		textColor: textColor,
	}

	return tools
}
