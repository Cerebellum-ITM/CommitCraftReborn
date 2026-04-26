package tui

import (
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderHeader draws the top breadcrumb row that sits above the tab bar:
//
//	⌘ commitCraft / compose · ~/Projects/cast        v0.9.0  [commitCraft]
//
// Left half: app icon, breadcrumb with the active tab and the working
// directory (collapsed to ~ when inside the user's home).
// Right half: version string and the green app pill.
func (model *Model) renderHeader(width int) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	muted := base.Foreground(theme.Muted)
	fgBold := base.Foreground(theme.FG).Bold(true)

	icon := muted.Render("⌘")
	appName := muted.Render("commitCraft")
	sep := muted.Render("/")
	tabName := fgBold.Render(strings.ToLower(tabLabel(model.topTab)))
	dotSep := muted.Render("·")
	pwd := muted.Render(condenseHomePath(model.pwd))

	left := lipgloss.JoinHorizontal(lipgloss.Top,
		icon, " ", appName, " ", sep, " ", tabName, " ", dotSep, " ", pwd,
	)

	versionText := muted.Render(model.Version)
	appPill := base.
		Foreground(theme.BG).
		Background(theme.Success).
		Bold(true).
		Padding(0, 1).
		Render("commitCraft")
	right := lipgloss.JoinHorizontal(lipgloss.Top, versionText, " ", appPill)

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	pad := max(1, width-leftW-rightW)
	spacer := strings.Repeat(" ", pad)

	return left + spacer + right
}

// condenseHomePath replaces the user's home prefix with "~" so the header
// breadcrumb stays compact even on long paths.
func condenseHomePath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+string(filepath.Separator)) {
		return "~" + p[len(home):]
	}
	return p
}
