package styles

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// CommitTypeColors holds the four-color palette assigned to a commit type:
// the type-block background/foreground (the chip on the left of a row) and
// the inline-message background/foreground (the [TYPE] tag rendered inside
// the message preview). Contrast for every entry is ≥ 4.5:1 (WCAG AA).
type CommitTypeColors struct {
	BgBlock color.Color
	FgBlock color.Color
	BgMsg   color.Color
	FgMsg   color.Color
}

var commitTypePalette = map[string]CommitTypeColors{
	"ADD": {
		lipgloss.Color("#2b3f34"),
		lipgloss.Color("#d1ead9"),
		lipgloss.Color("#182219"),
		lipgloss.Color("#b9d2bf"),
	},
	"FIX": {
		lipgloss.Color("#4a2729"),
		lipgloss.Color("#f4cdcf"),
		lipgloss.Color("#2a1416"),
		lipgloss.Color("#d4a8aa"),
	},
	"DOC": {
		lipgloss.Color("#2c4360"),
		lipgloss.Color("#d6e4f4"),
		lipgloss.Color("#182230"),
		lipgloss.Color("#b8c5d4"),
	},
	"WIP": {
		lipgloss.Color("#4a3a25"),
		lipgloss.Color("#ecd9b5"),
		lipgloss.Color("#2a2014"),
		lipgloss.Color("#d4bf95"),
	},
	"STYLE": {
		lipgloss.Color("#3e3268"),
		lipgloss.Color("#e9e0ff"),
		lipgloss.Color("#1d1830"),
		lipgloss.Color("#c8bce0"),
	},
	"REFACTOR": {
		lipgloss.Color("#2c4f5e"),
		lipgloss.Color("#c4e0ec"),
		lipgloss.Color("#16252e"),
		lipgloss.Color("#a4c4d2"),
	},
	"TEST": {
		lipgloss.Color("#34401e"),
		lipgloss.Color("#d4e3a8"),
		lipgloss.Color("#1b2010"),
		lipgloss.Color("#b1c189"),
	},
	"PERF": {
		lipgloss.Color("#4a2949"),
		lipgloss.Color("#f1c5ee"),
		lipgloss.Color("#271425"),
		lipgloss.Color("#cfa1cb"),
	},
	"CHORE": {
		lipgloss.Color("#2a2d36"),
		lipgloss.Color("#b8bcc4"),
		lipgloss.Color("#14161c"),
		lipgloss.Color("#8a8e98"),
	},
	"DEL": {
		lipgloss.Color("#3e1c1c"),
		lipgloss.Color("#e8b6b6"),
		lipgloss.Color("#1f0c0c"),
		lipgloss.Color("#c89494"),
	},
	"BUILD": {
		lipgloss.Color("#4a3422"),
		lipgloss.Color("#ecc7a3"),
		lipgloss.Color("#2a1c10"),
		lipgloss.Color("#d3aa84"),
	},
	"CI": {
		lipgloss.Color("#313a55"),
		lipgloss.Color("#c8d2ea"),
		lipgloss.Color("#161a26"),
		lipgloss.Color("#a3acc4"),
	},
	"REVERT": {
		lipgloss.Color("#4a2f1c"),
		lipgloss.Color("#ecbe9a"),
		lipgloss.Color("#2a190d"),
		lipgloss.Color("#cda07b"),
	},
	"SEC": {
		lipgloss.Color("#4a232b"),
		lipgloss.Color("#f0bdc4"),
		lipgloss.Color("#29101a"),
		lipgloss.Color("#cf99a3"),
	},
}

// CommitTypePalette returns the four-color set associated with a commit type
// tag. Lookups are case-insensitive. Tags outside the spec (e.g. legacy
// IMP/REM/REF/MOV/REL) fall back to a neutral palette derived from the theme
// so the row still renders without a hard-coded color.
func CommitTypePalette(theme *Theme, tag string) CommitTypeColors {
	if p, ok := commitTypePalette[strings.ToUpper(tag)]; ok {
		return p
	}
	return CommitTypeColors{
		BgBlock: theme.Surface,
		FgBlock: theme.FgMuted,
		BgMsg:   theme.BG,
		FgMsg:   theme.Blur,
	}
}
