package styles

import (
	"image/color"

	"github.com/charmbracelet/bubbles/v2/help"
	"github.com/charmbracelet/bubbles/v2/textarea"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
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
	Console          string
	GhEnable         string
	GhMissing        string
	ClipboardEnable  string
	ClipboardMissing string
}

type Styles struct {
	Base           lipgloss.Style
	IndicatorStyle lipgloss.Style
	Help           help.Styles
	TextArea       textarea.Styles
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
	return &Styles{
		Base:           base,
		IndicatorStyle: indicator,
		Help: help.Styles{
			ShortKey:       base.Foreground(t.Accent),
			ShortDesc:      base.Foreground(t.FgMuted),
			ShortSeparator: base.Foreground(t.White),
			FullKey:        base.Foreground(t.Accent),
			FullDesc:       base.Foreground(t.FgMuted),
			FullSeparator:  base.Foreground(t.White),
			Ellipsis:       base.Foreground(t.FgSubtle),
		},
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
}
