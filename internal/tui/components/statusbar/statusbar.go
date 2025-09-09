package statusbar

import (
	"commit_craft_reborn/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type StatusBar struct {
	Content             string
	theme               *styles.Theme
	Style               lipgloss.Style
	Level               LogLevel
	spinner             spinner.Model
	showSpinner         bool
	defaultContentStyle *lipgloss.Style
	AppWith             int
}

func New(content string, level LogLevel, with int, theme *styles.Theme) StatusBar {
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
		theme:               theme,
		AppWith:             with,
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

	logo := sb.theme.AppStyles().Base.
		Background(sb.theme.Logo).
		Padding(0, 1).SetString("CommitCraft")

	messageSeparator := sb.theme.AppStyles().
		Base.Background(sb.theme.Blur).
		SetString("  »").
		String()

	prefixStyle := sb.theme.AppStyles().Base
	fillContent := sb.theme.AppStyles().Base.Background(lipgloss.Black)
	contentStyle := sb.theme.AppStyles().Base.Background(sb.theme.Blur)

	switch sb.Level {
	case LevelInfo:
		prefixText = prefixStyle.Background(sb.theme.Info).SetString("  INFO  ").String()
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

	renderedContent := contentStyle.Render(" " + sb.Content + "  ")
	finalContent := prefixText + messageSeparator + renderedContent + spinnerView
	remainingSpace := sb.AppWith - lipgloss.Width(logo.String()) - lipgloss.Width(finalContent) - 10
	leftDashes := fillContent.SetString(strings.Repeat("─", remainingSpace/2)).String()
	rightDashes := fillContent.SetString(strings.Repeat("─", remainingSpace-remainingSpace/2)).
		String()

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		logo.String(),
		"     ",
		leftDashes,
		finalContent,
		rightDashes,
		"      ",
	)
}
