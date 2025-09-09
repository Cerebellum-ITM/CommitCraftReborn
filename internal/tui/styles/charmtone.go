package styles

import (
	// "github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

func NewCharmtoneTheme() *Theme {
	t := &Theme{
		Name:   "charmtone",
		IsDark: true,
		Logo:   charmtone.Oceania,

		FgBase:      charmtone.Ash,
		BorderFocus: charmtone.Damson,

		BgOverlay: charmtone.Iron,

		Primary:   charmtone.Sapphire,
		Secondary: charmtone.Dolly,
		Tertiary:  charmtone.Zest,
		Accent:    charmtone.Plum,
		Blur:      charmtone.Pepper,

		// Status
		Success: charmtone.Guac,
		Error:   charmtone.Sriracha,
		Warning: charmtone.Zest,
		Info:    charmtone.Ox,
		Fatal:   charmtone.Orchid,

		// Colors
		Yellow: charmtone.Mustard,
		Purple: charmtone.Grape,
		White:  charmtone.Butter,
	}

	return t
}
