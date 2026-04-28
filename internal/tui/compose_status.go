package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// composeMaxChars is the soft cap shown in the status bar's "X / Y chars"
// counter. We don't actually clamp typing at this length; it's a UX hint
// nudging the user to keep summaries terse.
const composeMaxChars = 500

// renderComposeBottomBar draws the static info bar that lives below the
// compose panels:
//
//	[compose]  78 / 500 chars · 3 key points         ▓▓▓░░░ 16%
//
// Char count is the textarea content plus joined keypoints. The progress
// bar is currentChars / composeMaxChars.
func (model *Model) renderComposeBottomBar(width int) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	pill := base.
		Foreground(theme.BG).
		Background(theme.Primary).
		Bold(true).
		Padding(0, 1).
		Render("compose")

	chars := composeCharCount(model)
	pct := chars * 100 / composeMaxChars
	if pct > 100 {
		pct = 100
	}

	info := base.Foreground(theme.Muted).Render(
		fmt.Sprintf("%d / %d chars · %d key points",
			chars, composeMaxChars, len(model.keyPoints),
		),
	)

	left := lipgloss.JoinHorizontal(lipgloss.Top, pill, "  ", info)
	leftW := lipgloss.Width(left)

	// Reserve at least 12 cols for a bar + percentage on the right.
	const minRightW = 12
	rightAvailable := max(minRightW, width-leftW-2)
	barW := max(4, rightAvailable-6) // 6 chars reserved for " 100%"
	filled := barW * pct / 100
	bar := base.Foreground(theme.Primary).Render(strings.Repeat("█", filled)) +
		base.Foreground(theme.Subtle).Render(strings.Repeat("░", barW-filled))
	pctText := base.Foreground(theme.Muted).Render(fmt.Sprintf("%3d%%", pct))
	right := bar + " " + pctText

	rightW := lipgloss.Width(right)
	pad := max(1, width-leftW-rightW)
	spacer := strings.Repeat(" ", pad)

	return left + spacer + right
}

// composeCharCount sums the textarea content with each key point so the
// status bar reflects the total prompt material the user has prepared.
func composeCharCount(model *Model) int {
	chars := len(model.commitsKeysInput.Value())
	for _, kp := range model.keyPoints {
		chars += len(kp)
	}
	return chars
}

// helpEntry is a single key+description pair rendered in the help row.
type helpEntry struct{ key, desc string }

// helpEntriesForFocus returns the keys most relevant to the section the
// user is currently in, plus the always-on global ones (Tab/Esc/help).
// Order goes from most local (the section's own actions) to most global.
func helpEntriesForFocus(f focusableElement) []helpEntry {
	switch f {
	case focusComposeType:
		return []helpEntry{
			{"← →", "cycle type"},
			{"^T", "open list"},
			{"tab", "next section"},
			{"esc", "back"},
			{"?", "help"},
		}
	case focusComposeScope:
		return []helpEntry{
			{"e/↵", "edit scope"},
			{"x", "clear"},
			{"^P", "file picker"},
			{"tab", "next section"},
			{"esc", "back"},
			{"?", "help"},
		}
	case focusComposeSummary, focusMsgInput:
		return []helpEntry{
			{"^W", "generate"},
			{"^A", "add key point"},
			{"@", "mention file"},
			{"tab", "next section"},
			{"esc", "back"},
			{"?", "help"},
		}
	case focusComposeKeypoints:
		return []helpEntry{
			{"↑↓", "navigate"},
			{"x", "remove"},
			{"^A", "add from input"},
			{"tab", "next section"},
			{"esc", "back"},
			{"?", "help"},
		}
	case focusComposePipelineModels:
		return []helpEntry{
			{"↑↓", "pick stage"},
			{"↵", "change model"},
			{"tab", "next section"},
			{"esc", "back"},
			{"?", "help"},
		}
	case focusComposeAISuggestion, focusAIResponse:
		return []helpEntry{
			{"↑↓", "scroll"},
			{"^E", "edit reply"},
			{"↵", "accept"},
			{"tab", "next section"},
			{"esc", "back"},
			{"?", "help"},
		}
	}
	// Fallback: same as summary so the user always has Ctrl+W within
	// reach when focus is in some unexpected state.
	return []helpEntry{
		{"^W", "generate"},
		{"^A", "add key point"},
		{"tab", "next section"},
		{"esc", "back"},
		{"?", "help"},
	}
}

// renderComposeHelpLine renders the bottom-most help row tailored to the
// currently focused compose section. The format matches the design:
// keys in the brand primary, descriptions in muted, separated by "·".
func (model *Model) renderComposeHelpLine() string {
	return model.renderHelpEntries(helpEntriesForFocus(model.focusedElement))
}

// helpEntriesForState returns the keys most relevant to the given app
// state. Used by the global help line so every screen — not just the
// compose view — gets a state-aware hint bar at the bottom.
func helpEntriesForState(s appState, mode appMode) []helpEntry {
	switch s {
	case stateChoosingCommit:
		return []helpEntry{
			{"↑↓", "navigate"},
			{"↵", "open"},
			{"n", "new commit"},
			{"r", "release"},
			{"d", "drafts"},
			{"x", "delete"},
			{"/", "filter"},
			{"^x", "quit"},
		}
	case stateReleaseMainMenu:
		return []helpEntry{
			{"↑↓", "navigate"},
			{"↵", "open"},
			{"n", "new release"},
			{"x", "delete"},
			{"/", "filter"},
			{"esc", "back"},
			{"^x", "quit"},
		}
	case stateChoosingType:
		return []helpEntry{
			{"↑↓", "navigate"},
			{"↵", "select"},
			{"/", "filter"},
			{"esc", "back"},
		}
	case stateChoosingScope:
		return []helpEntry{
			{"↑↓", "navigate"},
			{"→", "enter dir"},
			{"←", "parent"},
			{"↵", "select"},
			{"/", "filter"},
			{"esc", "back"},
		}
	case stateReleaseChoosingCommits, stateReleaseBuildingText:
		return []helpEntry{
			{"↑↓", "navigate"},
			{"↵", "select"},
			{"tab", "switch panel"},
			{"esc", "back"},
		}
	case stateRewordSelectCommit:
		return []helpEntry{
			{"↑↓", "navigate"},
			{"↵", "select"},
			{"esc", "cancel"},
		}
	case stateSettingAPIKey:
		return []helpEntry{
			{"↵", "save"},
			{"esc", "cancel"},
		}
	case statePipeline:
		return []helpEntry{
			{"r", "retry all"},
			{"1/2/3", "retry stage"},
			{"tab", "focus stage"},
			{"pgup/pgdn", "scroll stage"},
			{"↑↓", "scroll diff"},
			{"j/k", "select file"},
			{"↵", "accept"},
			{"esc", "cancel"},
			{"^1/^2/^3", "switch tab"},
			{"^x", "quit"},
		}
	}
	return []helpEntry{
		{"^x", "quit"},
		{"?", "help"},
	}
}

// renderStateHelpLine renders the bottom hint bar for a non-compose state.
func (model *Model) renderStateHelpLine() string {
	return model.renderHelpEntries(helpEntriesForState(model.state, model.AppMode))
}

// renderHelpEntries is the shared formatter: keys in primary, descriptions
// in muted, dot separators between entries.
func (model *Model) renderHelpEntries(entries []helpEntry) string {
	theme := model.Theme
	base := theme.AppStyles().Base
	keyStyle := base.Foreground(theme.Primary)
	descStyle := base.Foreground(theme.Muted)
	sepStyle := descStyle

	parts := make([]string, 0, len(entries)*3)
	for i, e := range entries {
		parts = append(parts,
			keyStyle.Render(e.key),
			" ",
			descStyle.Render(e.desc),
		)
		if i < len(entries)-1 {
			parts = append(parts, "  ", sepStyle.Render("·"), "  ")
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
