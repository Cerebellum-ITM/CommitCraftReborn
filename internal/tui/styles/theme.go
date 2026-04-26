package styles

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Theme is the canonical color palette used across the TUI. The schema is
// kept small on purpose: surfaces, brand colors, semantic colors, and a
// dedicated diff palette. Per-component styles are exposed as methods so
// every theme automatically gets the same vocabulary (pills, panels…).
//
// The fields under "Legacy" exist only so the older TUI panels keep
// compiling while the new compose layout is rolled out. They are derived
// in fillLegacy() and should not be set directly by new themes.
type Theme struct {
	Name   string
	IsDark bool
	Logo   color.Color

	// Surfaces
	BG      color.Color
	Surface color.Color
	FG      color.Color
	Muted   color.Color
	Subtle  color.Color

	// Brand & semantic
	Primary   color.Color
	Secondary color.Color
	Success   color.Color
	Warning   color.Color
	Error     color.Color

	// Pipeline accents: AI highlights running stages and AI-driven UI
	// elements; SuccessDim is the muted green used for the post-completion
	// row flash; AcceptDim is the mid-tone used as the second frame of the
	// final-commit fade-in (Muted → AcceptDim → Success).
	AI         color.Color
	SuccessDim color.Color
	AcceptDim  color.Color

	// Diff
	Add   color.Color
	Del   color.Color
	Mod   color.Color
	Scope color.Color

	// === Legacy (derived; do not set in theme constructors) ===
	FgBase, FgMuted, FgHalfMuted, FgSubtle                  color.Color
	BorderFocus, FillTextLine, FocusableElement, Indicators color.Color
	BgOverlay, Input, Output                                color.Color
	Tertiary, Accent, Blur                                  color.Color
	Info, Fatal                                             color.Color
	Yellow, Purple, White, Red, Green, Black                color.Color

	styles  *Styles
	symbols *Symbols
}

type Symbols struct {
	Commit           string
	Rewrite          string
	NewAndRewrite    string
	Console          string
	GhEnable         string
	GhMissing        string
	CommitCraft      string
	ClipboardEnable  string
	ClipboardMissing string
	KeyPoint         string
}

type Styles struct {
	Base           lipgloss.Style
	IndicatorStyle lipgloss.Style
	Help           help.Styles
	TextArea       textarea.Styles

	KeyPointsInput struct {
		PromptFocused lipgloss.Style
		PromptBlurred lipgloss.Style
		DotsFocused   lipgloss.Style
		DotsBlurred   lipgloss.Style
	}
}

func (t *Theme) AppSymbols() *Symbols {
	if t.symbols == nil {
		t.symbols = DefaultSymbols()
	}
	return t.symbols
}

func (t *Theme) AppStyles() *Styles {
	if t.styles == nil {
		t.styles = t.buildStyles()
	}
	return t.styles
}

// Pill renders a small inline badge with the brand primary as background.
// Use it for the commit-type / scope chips in the compose panel.
func (t *Theme) Pill() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.Primary).
		Foreground(t.BG).
		Padding(0, 1).
		Bold(true)
}

// TypePill is the commit-type chip. It accepts a per-type color sourced from
// the user's config so every commit type keeps its identity, while the
// foreground stays readable against the theme background.
func (t *Theme) TypePill(bg color.Color) lipgloss.Style {
	if bg == nil {
		bg = t.Primary
	}
	return lipgloss.NewStyle().
		Background(bg).
		Foreground(t.BG).
		Padding(0, 1).
		Bold(true)
}

// Panel is the default panel chrome (rounded border + subtle border color).
func (t *Theme) Panel() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Subtle).
		Padding(0, 1)
}

// PanelFocus mirrors Panel with the brand primary on the border so the
// active panel pops in the layout.
func (t *Theme) PanelFocus() lipgloss.Style {
	return t.Panel().BorderForeground(t.Primary)
}

// fillLegacy derives the old field names from the new schema so the
// existing TUI code keeps building unchanged. Remove field-by-field as
// each panel is rewritten against the new schema.
func (t *Theme) fillLegacy() {
	t.FgBase = t.FG
	t.FgMuted = t.Muted
	t.FgHalfMuted = t.Muted
	t.FgSubtle = t.Subtle
	t.BorderFocus = t.Primary
	t.FillTextLine = t.Subtle
	t.FocusableElement = t.Subtle
	t.Indicators = t.Primary
	t.BgOverlay = t.Surface
	t.Input = t.FG
	t.Output = t.FG
	t.Tertiary = t.Secondary
	t.Accent = t.Primary
	t.Blur = t.Muted
	t.Info = t.Secondary
	t.Fatal = t.Error
	t.Yellow = t.Warning
	t.Purple = t.Primary
	t.White = t.FG
	t.Red = t.Error
	t.Green = t.Success
	t.Black = t.BG
	if t.Logo == nil {
		t.Logo = t.Primary
	}
	if t.AI == nil {
		t.AI = t.Secondary
	}
	if t.SuccessDim == nil {
		t.SuccessDim = t.Success
	}
	if t.AcceptDim == nil {
		t.AcceptDim = t.Muted
	}
}

func (t *Theme) buildStyles() *Styles {
	base := lipgloss.NewStyle().
		Foreground(t.FgBase)
	indicator := base.Foreground(t.Indicators)
	helpStyles := help.DefaultStyles(t.IsDark)
	helpStyles.ShortKey = base.Foreground(t.Accent)
	helpStyles.ShortDesc = base.Foreground(t.FgMuted)
	helpStyles.ShortSeparator = base.Foreground(t.White)
	helpStyles.FullKey = base.Foreground(t.Accent)
	helpStyles.FullDesc = base.Foreground(t.FgMuted)
	helpStyles.FullSeparator = base.Foreground(t.White)
	helpStyles.Ellipsis = base.Foreground(t.FgSubtle)

	s := &Styles{
		Base:           base,
		IndicatorStyle: indicator,
		Help:           helpStyles,
		TextArea: textarea.Styles{
			Focused: textarea.StyleState{
				Base:             base.Foreground(t.BorderFocus),
				Text:             base,
				LineNumber:       base.Foreground(t.White),
				CursorLine:       base.Background(t.BgOverlay),
				CursorLineNumber: base.Foreground(t.White),
				Placeholder:      base.Foreground(t.White),
				Prompt:           base.Foreground(t.FillTextLine),
			},
			Blurred: textarea.StyleState{
				Base:             base.Foreground(t.FocusableElement),
				Text:             base.Foreground(t.FgMuted),
				LineNumber:       base.Foreground(t.Blur),
				CursorLine:       base,
				CursorLineNumber: base.Foreground(t.Blur),
				Placeholder:      base.Foreground(t.Blur),
				Prompt:           base.Foreground(t.FocusableElement),
			},
			Cursor: textarea.CursorStyle{
				Color: t.Secondary,
				Shape: tea.CursorBar,
				Blink: true,
			},
		},
	}

	s.KeyPointsInput.PromptFocused = base.Foreground(t.Green).SetString("  > ")
	s.KeyPointsInput.PromptBlurred = base.Foreground(t.Blur).SetString("  > ")
	s.KeyPointsInput.DotsFocused = base.Foreground(t.Green).SetString("::: ")
	s.KeyPointsInput.DotsBlurred = base.Foreground(t.Blur).SetString("::: ")

	return s
}

// applySymbols selects the symbol set based on the user's nerd-font flag.
// Called by every theme constructor to keep them DRY.
func (t *Theme) applySymbols(useNerdFont bool) {
	if useNerdFont {
		t.symbols = NerdFontSymbols()
	} else {
		t.symbols = DefaultSymbols()
	}
}
