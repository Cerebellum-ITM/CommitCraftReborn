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
	var (
		prefixText       string
		spinnerView      string
		leftDashes       string
		rightDashes      string
		leftDashesCount  int
		rightDashesCount int
	)

	if sb.showSpinner {
		spinnerView = sb.spinner.View() + " "
	}

	logo := sb.theme.AppStyles().Base.
		Background(sb.theme.Logo).
		Padding(0, 1).SetString("CommitCraft")

	prefixStyle := sb.theme.AppStyles().Base
	fillContent := sb.theme.AppStyles().Base.Background(lipgloss.Black)
	contentStyle := sb.theme.AppStyles().Base.Background(sb.theme.Blur)
	horizontalSpace := sb.theme.AppStyles().
		Base.Background(lipgloss.Black).
		SetString("     ").
		String()

	switch sb.Level {
	case LevelInfo:
		prefixText = prefixStyle.Background(sb.theme.Info).SetString("  INFO  ").String()
	case LevelWarning:
		contentStyle = prefixStyle.Background(sb.theme.Warning).
			Foreground(sb.theme.BgOverlay)

		prefixText = prefixStyle.Background(sb.theme.Yellow).
			Foreground(sb.theme.BgOverlay).
			SetString("  Warning  ").
			String()
	case LevelError:
		prefixText = "[ERROR]: "
		prefixStyle = prefixStyle.Foreground(lipgloss.Color("9"))
	case LevelFatal:
		prefixText = "[FATAL]: "
		prefixStyle = prefixStyle.Foreground(lipgloss.Color("196"))
	default:
		return contentStyle.Render(sb.Content)
	}

	renderedContent := contentStyle.Render(" " + sb.Content + "  " + spinnerView)
	finalContent := prefixText + contentStyle.SetString("  »").String() + renderedContent
	statusBarSpace := lipgloss.Width(
		logo.String(),
	) + lipgloss.Width(
		finalContent,
	) + 2*lipgloss.Width(
		horizontalSpace,
	)

	effectiveWidth := max(0, sb.AppWith)
	if statusBarSpace >= effectiveWidth {
		leftDashes = ""
		rightDashes = ""
		leftDashesCount = 0
		rightDashesCount = 0
	} else {
		remainingSpaceForDashes := effectiveWidth - statusBarSpace
		remainingSpaceForDashes = max(0, remainingSpaceForDashes)

		leftDashesCount = max(0, remainingSpaceForDashes)
		rightDashesCount = max(0, remainingSpaceForDashes-leftDashesCount)

		leftDashes = fillContent.SetString(strings.Repeat("─", leftDashesCount)).String()
		rightDashes = fillContent.SetString(strings.Repeat("─", rightDashesCount)).String()
	}
	_ = rightDashes

	centralBlock := lipgloss.JoinHorizontal(
		lipgloss.Left,
		finalContent,
		horizontalSpace,
		leftDashes,
	)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		centralBlock,
		horizontalSpace,
		logo.String(),
	)
}
