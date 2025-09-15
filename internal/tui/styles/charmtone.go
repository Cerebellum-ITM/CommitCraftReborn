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
		FgMuted:     charmtone.Squid,
		FgHalfMuted: charmtone.Smoke,
		FgSubtle:    charmtone.Oyster,

		BorderFocus:      charmtone.Damson,
		FillTextLine:     charmtone.Sardine,
		FocusableElement: charmtone.Mustard,

		BgOverlay: charmtone.Iron,
		Input:     charmtone.Sardine,
		Output:    charmtone.Guppy,

		Primary:   charmtone.Oceania,
		Secondary: charmtone.Dolly,
		Tertiary:  charmtone.Zest,
		Accent:    charmtone.Anchovy,
		Blur:      charmtone.Charcoal,

		// Status
		Success: charmtone.Bok,
		Error:   charmtone.Sriracha,
		Warning: charmtone.Zest,
		Info:    charmtone.Ox,
		Fatal:   charmtone.Orchid,

		// Colors
		Yellow: charmtone.Mustard,
		Purple: charmtone.Grape,
		White:  charmtone.Butter,
		Red:    charmtone.Coral,
		Green:  charmtone.Guac,
		Black:  charmtone.Pepper,
	}

	return t
}
