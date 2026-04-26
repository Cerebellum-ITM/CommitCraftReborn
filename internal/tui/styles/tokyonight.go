package styles

import "charm.land/lipgloss/v2"

// NewTokyoNightTheme is a Tokyo Night Storm-inspired palette: deep blue
// backgrounds, soft cyan primary, magenta accents.
func NewTokyoNightTheme(useNerdFont bool) *Theme {
	t := &Theme{
		Name:   "tokyonight",
		IsDark: true,

		BG:      lipgloss.Color("#1a1b26"),
		Surface: lipgloss.Color("#24283b"),
		FG:      lipgloss.Color("#c0caf5"),
		Muted:   lipgloss.Color("#565f89"),
		Subtle:  lipgloss.Color("#3b4261"),

		Primary:   lipgloss.Color("#7aa2f7"),
		Secondary: lipgloss.Color("#bb9af7"),
		Success:   lipgloss.Color("#9ece6a"),
		Warning:   lipgloss.Color("#e0af68"),
		Error:     lipgloss.Color("#f7768e"),

		Add:   lipgloss.Color("#9ece6a"),
		Del:   lipgloss.Color("#f7768e"),
		Mod:   lipgloss.Color("#e0af68"),
		Scope: lipgloss.Color("#7dcfff"),

		AI:         lipgloss.Color("#bb9af7"),
		SuccessDim: lipgloss.Color("#7da159"),
		AcceptDim:  lipgloss.Color("#8b94bf"),
	}
	t.fillLegacy()
	t.applySymbols(useNerdFont)
	return t
}
