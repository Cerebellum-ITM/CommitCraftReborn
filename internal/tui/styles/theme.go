package styles

import (
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Theme struct {
	Name   string
	IsDark bool
	Logo   color.Color

	FgBase      color.Color
	FgMuted     color.Color
	FgHalfMuted color.Color
	FgSubtle    color.Color

	BorderFocus      color.Color
	FillTextLine     color.Color
	FocusableElement color.Color
	Indicators       color.Color

	BgOverlay color.Color
	Input     color.Color
	Output    color.Color

	Primary   color.Color
	Secondary color.Color
	Tertiary  color.Color
	Accent    color.Color
	Blur      color.Color

	Success color.Color
	Error   color.Color
	Warning color.Color
	Info    color.Color
	Fatal   color.Color

	Yellow color.Color
	Purple color.Color
	White  color.Color
	Red    color.Color
	Green  color.Color
	Black  color.Color

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
