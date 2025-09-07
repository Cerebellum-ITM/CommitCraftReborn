package statusbar

import (
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type StatusBar struct {
	Content             string
	Style               lipgloss.Style
	Level               LogLevel
	defaultContentStyle *lipgloss.Style
	spinner             spinner.Model
	showSpinner         bool
}

func New(content string, level LogLevel) StatusBar {
	defaultStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightYellow)
	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.BrightMagenta)
	return StatusBar{
		Content:             content,
		Level:               level,
		Style:               lipgloss.NewStyle().Foreground(lipgloss.BrightYellow),
		defaultContentStyle: &defaultStyle,
		spinner:             s,
		showSpinner:         false,
	}
}

func (sb *StatusBar) StartSpinner() tea.Cmd {
	sb.showSpinner = true
	return sb.spinner.Tick
}

func (sb *StatusBar) StopSpinner() tea.Cmd {
	sb.showSpinner = false
	return nil
}

func (sb *StatusBar) SetDefaultContentStyle(style lipgloss.Style) {
	sb.defaultContentStyle = &style
}

func (sb *StatusBar) ResetContentStyle() {
	if sb.defaultContentStyle != nil {
		sb.Style = *sb.defaultContentStyle
	}
}

func (sb *StatusBar) Update(msg tea.Msg) (StatusBar, tea.Cmd) {
	var cmd tea.Cmd
	if sb.showSpinner {
		sb.spinner, cmd = sb.spinner.Update(msg)
	}
	return *sb, cmd
}

func (sb StatusBar) Render() string {
	var prefixText string
	var spinnerView string
	if sb.showSpinner {
		spinnerView = sb.spinner.View() + " "
	}
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
	return spinnerView + renderedPrefix + renderedContent + spinnerView
}
