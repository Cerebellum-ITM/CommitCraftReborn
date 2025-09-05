package statusbar

import (
	"github.com/charmbracelet/lipgloss/v2"
)

type StatusBar struct {
	Content             string
	Style               lipgloss.Style
	Level               LogLevel
	defaultContentStyle *lipgloss.Style
}

func New(content string, level LogLevel) StatusBar {
	defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightYellow)
	return StatusBar{
		Content:             content,
		Level:               level,
		Style:               lipgloss.NewStyle().Foreground(lipgloss.BrightYellow),
		defaultContentStyle: &defaultStyle,
	}
}

func (sb *StatusBar) SetDefaultContentStyle(style lipgloss.Style) {
	sb.defaultContentStyle = &style
}

func (sb *StatusBar) ResetContentStyle() {
	if sb.defaultContentStyle != nil {
		sb.Style = *sb.defaultContentStyle
	}
}

func (sb StatusBar) Render() string {
	var prefixText string
	contentStyle := sb.Style
	prefixStyle := lipgloss.NewStyle().PaddingRight(1).Foreground(lipgloss.White)

	switch sb.Level {
	case LevelInfo:
		prefixText = "[INFO]: "
		prefixStyle = prefixStyle.Foreground(lipgloss.Color("12"))
	case LevelWarning:
		prefixText = "[WARN]: "
		prefixStyle = prefixStyle.Foreground(lipgloss.Color("220"))
	case LevelError:
		prefixText = "[ERROR]: "
		prefixStyle = prefixStyle.Foreground(lipgloss.Color("9"))
	case LevelFatal:
		prefixText = "[FATAL]: "
		prefixStyle = prefixStyle.Foreground(lipgloss.Color("196"))
	default:
		return contentStyle.Render(sb.Content)
	}

	renderedPrefix := prefixStyle.Render(prefixText)
	renderedContent := contentStyle.Render(sb.Content)
	return renderedPrefix + renderedContent
}
