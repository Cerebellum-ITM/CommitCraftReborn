package statusbar

import (
	"commit_craft_reborn/internal/tui/styles"
	"strings"

	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

type StatusBar struct {
	Content     string
	theme       *styles.Theme
	Level       LogLevel
	spinner     spinner.Model
	showSpinner bool
	AppWith     int
}

func New(content string, level LogLevel, with int, theme *styles.Theme) StatusBar {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.BrightMagenta)
	return StatusBar{
		Content:     content,
		Level:       level,
		spinner:     s,
		showSpinner: false,
		theme:       theme,
		AppWith:     with,
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
		sb.spinner.Style = sb.spinner.Style.Foreground(sb.theme.Blur)
		spinnerView = sb.theme.AppStyles().
			Base.Background(sb.theme.FgBase).
			Padding(0, 2).
			SetString(sb.spinner.View()).
			String()
	}

	logo := sb.theme.AppStyles().Base.
		Background(sb.theme.Logo).
		Padding(0, 1).SetString("CommitCraft")

	version := sb.theme.AppStyles().Base.
		Background(sb.theme.Black).
		Foreground(sb.theme.White).
		Padding(0, 1).SetString("v0.2.3")

	prefixStyle := sb.theme.AppStyles().Base.Padding(0, 2)
	fillContent := sb.theme.AppStyles().Base
	contentStyle := sb.theme.AppStyles().Base.Background(sb.theme.Blur)
	horizontalSpace := sb.theme.AppStyles().
		Base.
		SetString("   ").
		String()

	switch sb.Level {
	case LevelInfo:
		prefixText = prefixStyle.Background(sb.theme.Info).SetString(sb.Level.String()).String()
	case LevelSuccess:
		contentStyle = contentStyle.Background(sb.theme.Success).
			Foreground(sb.theme.Black)

		prefixText = prefixStyle.Background(sb.theme.Green).
			Foreground(sb.theme.White).
			SetString(sb.Level.String()).
			String()

	case LevelWarning:
		contentStyle = contentStyle.Background(sb.theme.Warning).
			Foreground(sb.theme.BgOverlay)

		prefixText = prefixStyle.Background(sb.theme.Yellow).
			Foreground(sb.theme.BgOverlay).
			SetString(sb.Level.String()).
			String()
	case LevelError:
		contentStyle = contentStyle.Background(sb.theme.Red).
			Foreground(sb.theme.White)

		prefixText = prefixStyle.Background(sb.theme.Error).
			Foreground(sb.theme.White).
			SetString(sb.Level.String()).
			String()
	case LevelFatal:
		contentStyle = contentStyle.Background(sb.theme.Fatal).
			Foreground(sb.theme.White)

		prefixText = prefixStyle.Background(sb.theme.Purple).
			Foreground(sb.theme.White).
			SetString(sb.Level.String()).
			String()

	default:
		return contentStyle.Render(sb.Content)
	}

	renderedContent := contentStyle.Render(" " + sb.Content + " ")
	finalContent := prefixText + contentStyle.SetString("  »").
		String() +
		renderedContent + spinnerView
	statusBarSpace := lipgloss.Width(logo.String()) +
		lipgloss.Width(finalContent) +
		2*lipgloss.Width(horizontalSpace) +
		lipgloss.Width(version.String())

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
		version.String(),
		logo.String(),
	)
}
