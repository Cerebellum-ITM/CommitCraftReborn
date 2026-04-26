package styles

import "charm.land/lipgloss/v2"

// NewHarmonizedTheme is the cool-indigo / sage / amber theme: dark surfaces
// with a violet primary that reads well against the BG without clashing
// with the warmer semantic colors.
func NewHarmonizedTheme(useNerdFont bool) *Theme {
	t := &Theme{
		Name:   "harmonized",
		IsDark: true,

		BG:      lipgloss.Color("#0e1016"),
		Surface: lipgloss.Color("#161922"),
		FG:      lipgloss.Color("#cfd3d8"),
		Muted:   lipgloss.Color("#6f7480"),
		Subtle:  lipgloss.Color("#3a3e48"),

		Primary:   lipgloss.Color("#b79cf4"),
		Secondary: lipgloss.Color("#7ea2d8"),
		Success:   lipgloss.Color("#86c3a7"),
		Warning:   lipgloss.Color("#c9a265"),
		Error:     lipgloss.Color("#d07070"),

		Add:   lipgloss.Color("#86c3a7"),
		Del:   lipgloss.Color("#d07070"),
		Mod:   lipgloss.Color("#c9a265"),
		Scope: lipgloss.Color("#7ea2d8"),

		AI:         lipgloss.Color("#c5a3ff"),
		SuccessDim: lipgloss.Color("#689683"),
		AcceptDim:  lipgloss.Color("#9fa3ac"),
	}
	t.fillLegacy()
	t.applySymbols(useNerdFont)
	return t
}
