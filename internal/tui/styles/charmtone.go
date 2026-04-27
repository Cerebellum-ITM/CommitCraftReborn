package styles

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

func NerdFontSymbols() *Symbols {
	return &Symbols{
		Commit:           "󰜘",
		Console:          "󰆍",
		Rewrite:          "",
		NewAndRewrite:    "󰼍",
		GhEnable:         "",
		GhMissing:        "",
		CommitCraft:      "",
		ClipboardEnable:  "󱄗",
		ClipboardMissing: "󱘛",
		KeyPoint:         ">",
	}
}

func DefaultSymbols() *Symbols {
	return &Symbols{
		Commit:           "X",
		Console:          "🖊",
		Rewrite:          "",
		NewAndRewrite:    "",
		ClipboardEnable:  "📋",
		ClipboardMissing: "X",
		GhEnable:         "💻",
		GhMissing:        "X",
		CommitCraft:      "📇",
		KeyPoint:         ">",
	}
}

// NewCharmtoneTheme is the default theme. Maps Charm's "charmtone" palette
// to the harmonized schema. Kept as the bootstrap default so callers that
// haven't migrated to the registry keep working unchanged.
func NewCharmtoneTheme(useNerdFont bool) *Theme {
	t := &Theme{
		Name:   "charmtone",
		IsDark: true,
		Logo:   charmtone.Oceania,

		BG:      charmtone.Pepper,
		Surface: charmtone.Iron,
		FG:      charmtone.Ash,
		Muted:   charmtone.Squid,
		Subtle:  charmtone.Charcoal,

		Primary:   charmtone.Oceania,
		Secondary: charmtone.Dolly,
		Success:   charmtone.Bok,
		Warning:   charmtone.Zest,
		Error:     charmtone.Sriracha,

		Add:   charmtone.Guac,
		Del:   charmtone.Coral,
		Mod:   charmtone.Mustard,
		Scope: charmtone.Dolly,

		AI:         lipgloss.Color("#b08cff"),
		SuccessDim: lipgloss.Color("#5d7a3a"),
		AcceptDim:  lipgloss.Color("#888a96"),
	}
	t.fillLegacy()
	t.applySymbols(useNerdFont)
	return t
}
