package statusbar

import (
	"strings"
	"time"

	"commit_craft_reborn/internal/tui/styles"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type clearStatusMsg struct{}

// StatusBar holds the state of the top-of-screen status surface. Its
// rendered output follows the "TYPE - message" two-pill scheme: a coloured
// label pill on the left, the message body in a darker shade of the same
// hue family next to it, and the version + ⌘ logo right-aligned.
type StatusBar struct {
	AppWith        int
	showSpinner    bool
	Content        string
	Version        string
	TmpStringChest string
	Level          LogLevel
	theme          *styles.Theme
	spinner        spinner.Model
}

func New(content string, level LogLevel, with int, theme *styles.Theme, version string) StatusBar {
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
		Version:     version,
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

func (sb *StatusBar) ShowMessageForDuration(
	message string,
	level LogLevel,
	duration time.Duration,
) tea.Cmd {
	sb.Level = level
	sb.TmpStringChest = sb.Content
	sb.Content = message
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func (sb *StatusBar) Update(msg tea.Msg) (StatusBar, tea.Cmd) {
	var cmd tea.Cmd

	if _, ok := msg.(clearStatusMsg); ok {
		sb.Content = sb.TmpStringChest
		sb.Level = LevelInfo
		return *sb, nil
	}

	if sb.showSpinner {
		sb.spinner, cmd = sb.spinner.Update(msg)
	}

	return *sb, cmd
}

// Render produces the full-width status row: TYPE pill + message body on
// the left, spinner (when active) + version pill + logo pill right-aligned.
// The right-side pills are theme-derived so they keep the brand mark
// consistent with the rest of the UI; only the left "TYPE - message"
// block uses the fixed dark palette per level.
func (sb StatusBar) Render() string {
	left := RenderStatus(sb.Level, sb.Content)

	logo := sb.theme.AppStyles().Base.
		Background(sb.theme.Primary).
		Foreground(sb.theme.FG).
		Bold(true).
		Padding(0, 1).SetString("⌘ CommitCraft")

	version := sb.theme.AppStyles().Base.
		Background(sb.theme.Black).
		Foreground(sb.theme.White).
		Padding(0, 1).SetString(sb.Version)

	rightParts := []string{}
	if sb.showSpinner {
		sb.spinner.Style = sb.spinner.Style.Foreground(sb.theme.Primary)
		rightParts = append(rightParts, sb.spinner.View(), "  ")
	}
	rightParts = append(rightParts, version.String(), logo.String())
	right := lipgloss.JoinHorizontal(lipgloss.Top, rightParts...)

	gap := sb.AppWith - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// --- pill / body styles -------------------------------------------------

var pillStyle = lipgloss.NewStyle().
	Padding(0, 1).
	Bold(true)

var msgStyle = lipgloss.NewStyle().
	Padding(0, 1)

var (
	pillInfo = pillStyle.Background(lipgloss.Color("#2c4360")).Foreground(lipgloss.Color("#d6e4f4"))
	msgInfo  = msgStyle.Background(lipgloss.Color("#182230")).Foreground(lipgloss.Color("#b8c5d4"))

	pillOK = pillStyle.Background(lipgloss.Color("#2b3f34")).Foreground(lipgloss.Color("#d1ead9"))
	msgOK  = msgStyle.Background(lipgloss.Color("#182219")).Foreground(lipgloss.Color("#b9d2bf"))

	pillWarn = pillStyle.Background(lipgloss.Color("#4a3a25")).Foreground(lipgloss.Color("#ecd9b5"))
	msgWarn  = msgStyle.Background(lipgloss.Color("#2a2014")).Foreground(lipgloss.Color("#d4bf95"))

	pillErr = pillStyle.Background(lipgloss.Color("#4a2729")).Foreground(lipgloss.Color("#f4cdcf"))
	msgErr  = msgStyle.Background(lipgloss.Color("#2a1416")).Foreground(lipgloss.Color("#d4a8aa"))

	pillAI = pillStyle.Background(lipgloss.Color("#3e3268")).Foreground(lipgloss.Color("#e9e0ff"))
	msgAI  = msgStyle.Background(lipgloss.Color("#1d1830")).Foreground(lipgloss.Color("#c8bce0"))

	pillRun = pillStyle.Background(lipgloss.Color("#2c4f5e")).Foreground(lipgloss.Color("#c4e0ec"))
	msgRun  = msgStyle.Background(lipgloss.Color("#16252e")).Foreground(lipgloss.Color("#a4c4d2"))

	pillDebug = pillStyle.Background(lipgloss.Color("#2a2d36")).
			Foreground(lipgloss.Color("#b8bcc4"))
	msgDebug = msgStyle.Background(lipgloss.Color("#14161c")).Foreground(lipgloss.Color("#8a8e98"))

	ctxStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6f7480"))
)

// stylesFor maps a LogLevel to its (pill, body, label) triple.
func stylesFor(level LogLevel) (lipgloss.Style, lipgloss.Style, string) {
	switch level {
	case LevelInfo:
		return pillInfo, msgInfo, "INFO"
	case LevelSuccess:
		return pillOK, msgOK, "OK"
	case LevelWarning:
		return pillWarn, msgWarn, "WARN"
	case LevelError, LevelFatal:
		return pillErr, msgErr, "ERROR"
	case LevelAI:
		return pillAI, msgAI, "AI"
	case LevelRun:
		return pillRun, msgRun, "RUN"
	case LevelDebug:
		return pillDebug, msgDebug, "DEBUG"
	}
	return pillInfo, msgInfo, "INFO"
}

// RenderStatus draws "TYPE - message" as two adjacent flat-rectangle pills.
func RenderStatus(level LogLevel, msg string) string {
	pill, body, label := stylesFor(level)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		pill.Render(label),
		body.Render(msg),
	)
}

// RenderStatusFull stretches the bar to width, with a right-aligned ctx
// (typically version, request stats, or other metadata).
func RenderStatusFull(level LogLevel, msg, ctx string, width int) string {
	left := RenderStatus(level, msg)
	right := ctxStyle.Render(ctx)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
