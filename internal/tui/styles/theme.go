package styles

import (
	"image/color"

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

	BorderFocus color.Color

	BgOverlay color.Color

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

	styles *Styles
}

type Styles struct {
	Base     lipgloss.Style
	TextArea textarea.Styles
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
	return &Styles{
		Base: base,
		TextArea: textarea.Styles{
			Focused: textarea.StyleState{
				Base:             base,
				Text:             base,
				LineNumber:       base.Foreground(t.White),
				CursorLine:       base.Background(t.BgOverlay),
				CursorLineNumber: base.Foreground(t.White),
				Placeholder:      base.Foreground(t.White),
				Prompt:           base.Foreground(t.BorderFocus),
			},
			Blurred: textarea.StyleState{
				Base:             base,
				Text:             base.Foreground(t.FgMuted),
				LineNumber:       base.Foreground(t.Blur),
				CursorLine:       base,
				CursorLineNumber: base.Foreground(t.Blur),
				Placeholder:      base.Foreground(t.Blur),
				Prompt:           base.Foreground(t.Blur),
			},
			Cursor: textarea.CursorStyle{
				Color: t.Secondary,
				Shape: tea.CursorBar,
				Blink: true,
			},
		},
	}
}
