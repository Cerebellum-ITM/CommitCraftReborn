package styles

import "charm.land/lipgloss/v2"

// NewGruvboxDarkTheme is the classic gruvbox-dark-medium palette: warm
// brown/cream surfaces with the signature orange/yellow brand colors.
func NewGruvboxDarkTheme(useNerdFont bool) *Theme {
	t := &Theme{
		Name:   "gruvbox-dark",
		IsDark: true,

		BG:      lipgloss.Color("#282828"),
		Surface: lipgloss.Color("#3c3836"),
		FG:      lipgloss.Color("#ebdbb2"),
		Muted:   lipgloss.Color("#928374"),
		Subtle:  lipgloss.Color("#504945"),

		Primary:   lipgloss.Color("#fe8019"),
		Secondary: lipgloss.Color("#83a598"),
		Success:   lipgloss.Color("#b8bb26"),
		Warning:   lipgloss.Color("#fabd2f"),
		Error:     lipgloss.Color("#fb4934"),

		Add:   lipgloss.Color("#b8bb26"),
		Del:   lipgloss.Color("#fb4934"),
		Mod:   lipgloss.Color("#fabd2f"),
		Scope: lipgloss.Color("#8ec07c"),
	}
	t.fillLegacy()
	t.applySymbols(useNerdFont)
	return t
}
