package styles

import (
	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
)

type Theme struct {
	Name   string
	IsDark bool
	Logo   color.Color

	FgBase      color.Color
	BorderFocus color.Color

	Primary   color.Color
	Secondary color.Color
	Tertiary  color.Color
	Accent    color.Color
	Blur      color.Color

	Success color.Color
	Error   color.Color
	Warning color.Color
	Info    color.Color

	styles *Styles
}

type Styles struct {
	Base lipgloss.Style
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
	}
}
