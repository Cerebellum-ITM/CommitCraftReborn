package styles

import "github.com/charmbracelet/x/exp/charmtone"

func NerdFontSymbols() *Symbols {
	return &Symbols{
		Commit:           "󰜘",
		Console:          "󰆍",
		GhEnable:         "",
		GhMissing:        "",
		CommitCraft:      "",
		ClipboardEnable:  "󱄗",
		ClipboardMissing: "󱘛",
	}
}

func DefaultSymbols() *Symbols {
	return &Symbols{
		Commit:           "X",
		Console:          "🖊",
		ClipboardEnable:  "📋",
		ClipboardMissing: "X",
		GhEnable:         "💻",
		GhMissing:        "X",
		CommitCraft:      "📇",
	}
}

func NewCharmtoneTheme(useNerdFont bool) *Theme {
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
		Indicators:       charmtone.Bok,

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
	if useNerdFont {
		t.symbols = NerdFontSymbols()
	} else {
		t.symbols = DefaultSymbols()
	}
	return t
}
