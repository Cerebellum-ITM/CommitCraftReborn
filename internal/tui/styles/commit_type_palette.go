package styles

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// CommitTypeChipInnerWidth is the canonical content width of a commit-type
// chip across every surface (History MasterList, type popup, compose
// pills). Combined with Padding(0, 1) and Align(Center), it produces a
// chip that is always CommitTypeChipInnerWidth+2 cells wide with the tag
// visually centered. The cap fits every default tag fully (longest is
// `REFACTOR` at 8 chars). Tags longer than this width are hard-truncated.
const CommitTypeChipInnerWidth = 8

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

// commitTypeAliases routes tags that don't have their own palette entry
// to the closest semantic match in the spec. Legacy CommitCraft tags
// (IMP/REM/REF/MOV/REL) and common project-specific tags (UI) get
// colored this way until the user defines their own palette entries.
var commitTypeAliases = map[string]string{
	"IMP": "REFACTOR", // improvements ≈ refactor
	"REM": "DEL",      // removal
	"REF": "REFACTOR", // refactor
	"MOV": "CHORE",    // file moves = housekeeping
	"REL": "BUILD",    // release involves build/packaging
	"UI":  "STYLE",    // ui changes are visual style
}

// CommitTypePalette returns the four-color set associated with a commit type
// tag. Lookups are case-insensitive. Tags are first resolved through
// commitTypeAliases so legacy/custom tags get colored too; anything
// still unmatched falls back to a neutral palette derived from the theme.
func CommitTypePalette(theme *Theme, tag string) CommitTypeColors {
	upper := strings.ToUpper(tag)
	if alias, ok := commitTypeAliases[upper]; ok {
		upper = alias
	}
	if p, ok := commitTypePalette[upper]; ok {
		return p
	}
	return CommitTypeColors{
		BgBlock: theme.Surface,
		FgBlock: theme.FgMuted,
		BgMsg:   theme.BG,
		FgMsg:   theme.Blur,
	}
}

// CommitTypeBlockStyle returns a Style configured with the "block" colors
// of the commit-type palette (BgBlock + FgBlock). Use it for high-emphasis
// chips/pills like the type chip on the left of a row or the scope pill
// inside the message column. The style intentionally has no padding or
// width set — the caller decides those depending on the chip layout.
func CommitTypeBlockStyle(theme *Theme, tag string) lipgloss.Style {
	p := CommitTypePalette(theme, tag)
	return lipgloss.NewStyle().Background(p.BgBlock).Foreground(p.FgBlock)
}

// CommitTypeMsgStyle returns a Style configured with the "message" colors
// of the commit-type palette (BgMsg + FgMsg). Use it for the dimmer
// surface of a row — typically the commit title.
func CommitTypeMsgStyle(theme *Theme, tag string) lipgloss.Style {
	p := CommitTypePalette(theme, tag)
	return lipgloss.NewStyle().Background(p.BgMsg).Foreground(p.FgMsg)
}

// commitTypeNerdIcons maps each canonical commit-type tag to a Nerd Font
// glyph that telegraphs the tag's semantics at a glance. Lookups go
// through the same alias table (`commitTypeAliases`) the colour palette
// uses, so legacy/custom tags inherit their alias's icon.
//
// Codepoints are written as explicit `\uXXXX` escapes (instead of the
// pasted glyph character) so the source bytes match the rune the font
// actually has. Pasting a raw nerd-font glyph through an editor that
// normalises private-use codepoints can silently rewrite it; escapes
// are immune.
var commitTypeNerdIcons = map[string]string{
	"ADD":      "", // nf-oct-diff_added
	"DEL":      "", // nf-fa-delete_left
	"FIX":      "", // nf-fa-bandage
	"DOC":      "", // nf-fa-book_journal_whills
	"WIP":      "", // nf-fa-hammer
	"STYLE":    "", // nf-seti-stylelint
	"UI":       "", // nf-fa-window_restore
	"REFACTOR": "", // nf-fa-recycle
	"TEST":     "", // nf-fa-flask
	"PERF":     "", // nf-fa-tachometer
	"CHORE":    "", // nf-fa-broom
	"BUILD":    "", // nf-fa-cogs
	"CI":       "", // nf-fa-server
	"REVERT":   "", // nf-fa-undo
	"SEC":      "", // nf-fa-shield
}

// commitTypeAsciiIcons is the no-nerd-fonts fallback. ASCII bullets give
// each tag a distinct silhouette without depending on a glyph patched
// font.
var commitTypeAsciiIcons = map[string]string{
	"ADD":      "+",
	"DEL":      "-",
	"FIX":      "*",
	"DOC":      "@",
	"WIP":      "~",
	"STYLE":    "/",
	"REFACTOR": "&",
	"TEST":     "!",
	"PERF":     "^",
	"CHORE":    ".",
	"BUILD":    "#",
	"CI":       "$",
	"REVERT":   "<",
	"SEC":      "|",
}

// IconForCommitTag returns the per-tag glyph rendered next to the
// type chip in the Release inspect list (and anywhere else that wants
// a tag-aware icon). Falls back to ASCII when nerd fonts are off and
// to the bandage glyph for tags that don't have a dedicated entry yet,
// so the row never collapses to no icon at all.
func IconForCommitTag(tag string, useNerdFonts bool) string {
	upper := strings.ToUpper(tag)
	if useNerdFonts {
		if icon, ok := commitTypeNerdIcons[upper]; ok {
			return icon
		}
		if alias, ok := commitTypeAliases[upper]; ok {
			if icon, ok := commitTypeNerdIcons[alias]; ok {
				return icon
			}
		}
		return "" // nf-fa-bandage as the generic catch-all
	}
	if icon, ok := commitTypeAsciiIcons[upper]; ok {
		return icon
	}
	if alias, ok := commitTypeAliases[upper]; ok {
		if icon, ok := commitTypeAsciiIcons[alias]; ok {
			return icon
		}
	}
	return "#"
}
