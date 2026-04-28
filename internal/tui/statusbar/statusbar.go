package statusbar

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

type clearStatusMsg struct{}

// ChangelogIndicator drives the persistent CHANGELOG pill drawn at the
// right edge of the status bar (next to the version + logo pills). The
// model refreshes it whenever the changelog state may have changed —
// startup, before each Ctrl+W, and after a commit is created.
type ChangelogIndicator struct {
	// Present is true when CHANGELOG.md (or the configured path) exists in
	// the repo and the feature is enabled in config.
	Present bool
	// WillAutoUpdate is true when the next Ctrl+W will run the refiner
	// (file present AND clean). When false but Present is true, the file
	// is dirty (staged/unstaged) and the refiner will be skipped.
	WillAutoUpdate bool
	// UseNerdFonts toggles between NerdFont icons and an ASCII fallback.
	UseNerdFonts bool
}

// ScopeStaleIndicator drives the persistent warning pill that appears
// when a commit was loaded from the DB without a linked git hash, so
// the scope picker cannot highlight the commit's actual modified files.
type ScopeStaleIndicator struct {
	Stale        bool
	UseNerdFonts bool
}

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
	// Changelog drives the persistent CHANGELOG pill on the right side.
	// Populated by Model.syncChangelogIndicator after every state refresh.
	Changelog ChangelogIndicator
	// ScopeStale drives the persistent "scope data stale" warning pill,
	// shown when a DB-loaded commit has no linked hash and the picker
	// cannot resolve its real modified files.
	ScopeStale ScopeStaleIndicator
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

// SetTheme swaps the active theme so subsequent renders pick up the new
// palette (logo background, spinner color, etc.).
func (sb *StatusBar) SetTheme(t *styles.Theme) {
	sb.theme = t
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
		Background(sb.theme.Logo).
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
	if sb.ScopeStale.Stale {
		rightParts = append(rightParts, renderScopeStalePill(sb.ScopeStale), " ")
	}
	if sb.Changelog.Present {
		rightParts = append(rightParts, renderChangelogPill(sb.Changelog), " ")
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

	pillChangelog = pillStyle.Background(lipgloss.Color("#2f4a3a")).
			Foreground(lipgloss.Color("#cdeadc"))
	msgChangelog = msgStyle.Background(lipgloss.Color("#152821")).
			Foreground(lipgloss.Color("#a9d2bb"))

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
	case LevelChangelog:
		return pillChangelog, msgChangelog, "≡ CHANGELOG"
	}
	return pillInfo, msgInfo, "INFO"
}

// renderChangelogPill draws the persistent CHANGELOG indicator that lives
// on the right edge of the status bar. It uses the same green palette as
// LevelChangelog so it visually groups with the in-flight "≡ CHANGELOG"
// pill the user already knows.
//
// Icons are NerdFont private-use codepoints:
//   - U+F11FC (󱇼) — file present but auto-update will NOT run
//     (feature off, file already dirty, etc.).
//   - U+F1AD3 (󱫓) — auto-update will run on the next Ctrl+W.
//
// When NerdFonts are disabled the indicator falls back to the same triple
// horizontal-line glyph used by the in-flight pill so the bar still hints
// at "changelog" without requiring a special font.
func renderChangelogPill(ind ChangelogIndicator) string {
	icon := "≡"
	if ind.UseNerdFonts {
		icon = "\U000f11fc" // 󱇼 — file present
		if ind.WillAutoUpdate {
			icon = "\U000f1ad3" // 󱫓 — auto-update active
		}
	}
	return pillChangelog.Render(icon)
}

// renderScopeStalePill draws the persistent warning pill shown when the
// loaded commit has no linked git hash, so the scope picker cannot
// resolve its actual modified files. Reuses the warning palette so it
// visually groups with WARN status messages.
//
// Icon U+F13D2 (󱏒) is the NerdFont "alert/missing data" glyph; the
// ASCII fallback is "!" enclosed for legibility.
func renderScopeStalePill(ind ScopeStaleIndicator) string {
	icon := "!"
	if ind.UseNerdFonts {
		icon = "\U000f13d2" // 󱏒
	}
	return pillWarn.Render(icon)
}

// RenderCwdPill draws a persistent "CWD <path>" two-segment pill using the
// debug palette (slate label + near-black body). It is rendered between the
// main panels and the help line so the user always knows which working
// directory CommitCraft is operating on.
//
// maxWidth caps the total pill width; when the path doesn't fit, it is
// truncated from the LEFT with a leading "…" so the trailing path segments
// (the most informative part) stay visible.
func RenderCwdPill(path string, maxWidth int) string {
	pill := pillDebug.Render("CWD")
	pillW := lipgloss.Width(pill)
	// msgDebug has Padding(0, 1) on each side — subtract 2 cells of chrome
	// before deciding how much path text fits.
	bodyAvail := maxWidth - pillW - 2
	if bodyAvail < 3 {
		bodyAvail = 3
	}
	runes := []rune(path)
	if len(runes) > bodyAvail {
		runes = append([]rune{'…'}, runes[len(runes)-bodyAvail+1:]...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		pill,
		msgDebug.Render(string(runes)),
	)
}

// RenderMentionPill draws an inline `@<token>` chip using the success
// palette. Used by callers that render commit input/output themselves
// (key-points list, AI suggestion panel) so file references the user
// typed with `@` stand out as a coloured chip rather than plain prose.
func RenderMentionPill(token string) string {
	return pillOK.Render(token)
}

// RenderStatus draws "TYPE - message" as two adjacent flat-rectangle pills.
func RenderStatus(level LogLevel, msg string) string {
	pill, body, label := stylesFor(level)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		pill.Render(label),
		body.Render(msg),
	)
}

// RenderLabeled mirrors RenderStatus but with a caller-provided label so
// the same dark pill palettes can drive contextual section markers (e.g.
// the compose bottom bar uses one level per focused section, with the
// section name as the pill text).
func RenderLabeled(level LogLevel, label, msg string) string {
	pill, body, _ := stylesFor(level)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		pill.Render(label),
		body.Render(msg),
	)
}

// LabelPillWidth returns the rendered width of just the pill (not the
// body) for a given level + label. Useful when the caller needs to align
// extra content right of the pair.
func LabelPillWidth(level LogLevel, label string) int {
	pill, _, _ := stylesFor(level)
	return lipgloss.Width(pill.Render(label))
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
