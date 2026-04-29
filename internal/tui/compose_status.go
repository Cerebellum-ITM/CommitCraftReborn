package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/tui/statusbar"
	"commit_craft_reborn/internal/tui/styles"
)

// composeMaxChars is the soft cap shown in the status bar's "X / Y chars"
// counter. We don't actually clamp typing at this length; it's a UX hint
// nudging the user to keep summaries terse.
const composeMaxChars = 500

// renderComposeBottomBar draws a contextual info bar below the compose
// panels. The pill on the left names the focused section (using one of
// the dark message palettes per section so each one is visually distinct)
// and the body content depends on what the section is about — type
// description, scope value, char counter + progress bar, key-point
// total, model context window, etc.
func (model *Model) renderComposeBottomBar(width int) string {
	level, label, body, right := composeBottomBarContent(model, width)
	left := statusbar.RenderLabeled(level, label, body)

	if right == "" {
		// Pad so the bar fills width; keeps the underlying surface clean
		// when the section has no right-aligned extra.
		gap := max(0, width-lipgloss.Width(left))
		return left + strings.Repeat(" ", gap)
	}

	rightW := lipgloss.Width(right)
	leftW := lipgloss.Width(left)
	pad := max(1, width-leftW-rightW)
	return left + strings.Repeat(" ", pad) + right
}

// composeBottomBarContent picks the (palette, label, body, right-extra)
// tuple for the focused compose section. Right-extra is empty for every
// section except `summary`, which keeps the chars/% progress bar.
func composeBottomBarContent(
	model *Model, width int,
) (statusbar.LogLevel, string, string, string) {
	theme := model.Theme
	base := theme.AppStyles().Base

	switch model.focusedElement {
	case focusComposeType:
		desc := commitTypeDescription(model, model.commitType)
		if desc == "" {
			desc = "(no description)"
		}
		return statusbar.LevelInfo, "TYPE", desc, ""

	case focusComposeScope:
		body := composeScopeBody(model)
		return statusbar.LevelChangelog, "SCOPE", body, ""

	case focusComposeSummary, focusMsgInput:
		chars := composeCharCount(model)
		body := fmt.Sprintf("%d / %d chars", chars, composeMaxChars)
		right := composeProgressBar(theme, base, chars, width)
		return statusbar.LevelAI, "SUMMARY", body, right

	case focusComposeKeypoints:
		body := fmt.Sprintf("%d key point", len(model.keyPoints))
		if len(model.keyPoints) != 1 {
			body += "s"
		}
		return statusbar.LevelWarning, "KEYPOINTS", body, ""

	case focusComposePipelineModels:
		body := composePipelineModelBody(model)
		return statusbar.LevelRun, "PIPELINE", body, ""

	case focusComposeAISuggestion, focusAIResponse:
		body := composeAISuggestionBody(model)
		return statusbar.LevelSuccess, "AI", body, ""
	}

	// Fallback: behaves like the summary section so the bar always has
	// useful content even if a new focus enum slips through.
	chars := composeCharCount(model)
	body := fmt.Sprintf("%d / %d chars", chars, composeMaxChars)
	right := composeProgressBar(theme, base, chars, width)
	return statusbar.LevelAI, "SUMMARY", body, right
}

// composeProgressBar renders the chars/% indicator on the right of the
// summary bottom bar. Uses the same Braille-based ramp as the quota
// bars (`renderBrailleRamp`) so every progress visual in the TUI shares
// one fill style.
func composeProgressBar(
	theme *styles.Theme, base lipgloss.Style, chars, width int,
) string {
	pct := chars * 100 / composeMaxChars
	if pct > 100 {
		pct = 100
	}
	const minBarArea = 12
	rightAvailable := max(minBarArea, width/3)
	barW := max(4, rightAvailable-6)
	fillColor := theme.Primary
	switch {
	case pct >= 90:
		fillColor = theme.Error
	case pct >= 70:
		fillColor = theme.Warning
	}
	bar := renderBrailleRamp(chars, composeMaxChars, barW, base, fillColor, theme.Subtle)
	pctText := base.Foreground(theme.Muted).Render(fmt.Sprintf("%3d%%", pct))
	return bar + " " + pctText
}

// composeScopeBody summarises the staged changeset that is about to be
// committed: file count + total +adds / -dels pulled from
// `git diff --staged --numstat`. Falls back to a hint when nothing is
// staged so the user knows there is no commit to build yet.
func composeScopeBody(model *Model) string {
	numstat, err := git.GetStagedNumstat()
	if err != nil || len(numstat) == 0 {
		return "no staged changes · stage files with git add"
	}
	adds, dels := 0, 0
	for _, ns := range numstat {
		if ns.Adds > 0 {
			adds += ns.Adds
		}
		if ns.Dels > 0 {
			dels += ns.Dels
		}
	}
	noun := "file"
	if len(numstat) != 1 {
		noun = "files"
	}
	return fmt.Sprintf("%d %s staged · +%d −%d", len(numstat), noun, adds, dels)
}

// commitTypeDescription returns the description of the currently
// selected commit type, looked up against the resolved type list.
func commitTypeDescription(model *Model, tag string) string {
	for _, ct := range model.finalCommitTypes {
		if ct.Tag == tag {
			return ct.Description
		}
	}
	return ""
}

// composePipelineModelBody returns "<model id> · ctx Nk" for the stage
// currently under the pipeline-models cursor. Falls back to "(unset)"
// when no model is configured for the stage.
func composePipelineModelBody(model *Model) string {
	stages := composePipelineStages(model)
	if len(stages) == 0 {
		return "(no stages)"
	}
	idx := model.pipelineModelStageIndex
	if idx < 0 || idx >= len(stages) {
		idx = 0
	}
	stage := stages[idx]
	modelID := config.CurrentModelForStage(model.globalConfig, stage.stage)
	if modelID == "" {
		return stage.label + " · (unset)"
	}
	ctx := lookupModelContext(model, modelID)
	if ctx == 0 {
		return fmt.Sprintf("%s · %s", stage.label, modelID)
	}
	return fmt.Sprintf("%s · %s · %dk ctx", stage.label, modelID, ctx/1000)
}

// lookupModelContext returns the cached context window for modelID, or 0
// when the cache is empty / the id is not in it. Read-only call — no
// fetch is triggered from here.
func lookupModelContext(model *Model, modelID string) int {
	cached, _, err := model.db.LoadModelsCache()
	if err != nil {
		return 0
	}
	for _, m := range cached {
		if m.ID == modelID {
			return m.ContextWindow
		}
	}
	return 0
}

// composeAISuggestionBody describes the AI panel: its char count when
// populated, an "(empty)" hint otherwise.
func composeAISuggestionBody(model *Model) string {
	if strings.TrimSpace(model.commitTranslate) == "" {
		return "(empty) · press ^W to generate"
	}
	return fmt.Sprintf("%d chars · ↵ to accept", len(model.commitTranslate))
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
	entries := helpEntriesForState(model.state, model.AppMode)
	if model.state == stateChoosingCommit {
		cycleLabel := "cycle keypoint"
		if model.historyView.modeBar.Mode() == ModeStagesResponse {
			cycleLabel = "cycle stage"
		}
		// Insert the cycle entry next to the navigation keys and append the
		// "more" hint as the last item so the trail of essentials stays
		// readable while the popup remains discoverable.
		extra := []helpEntry{
			{"^]/^[", cycleLabel},
			{"^m", "swap inspect"},
			{"?", "more"},
		}
		entries = append(entries, extra...)
	}
	return model.renderHelpEntries(entries)
}

// renderHelpEntries is the shared formatter: keys in primary, descriptions
// in muted, dot separators between entries. When the entries don't fit the
// current terminal width, trailing items are dropped — except the "?" entry
// (when present), which is pinned to the end so the popup stays
// discoverable at every width.
func (model *Model) renderHelpEntries(entries []helpEntry) string {
	theme := model.Theme
	base := theme.AppStyles().Base
	keyStyle := base.Foreground(theme.Primary)
	descStyle := base.Foreground(theme.Muted)
	sepStyle := descStyle

	const sep = "  ·  " // "  " + "·" + "  "
	sepW := lipgloss.Width(sep)

	entryW := func(e helpEntry) int {
		return lipgloss.Width(e.key) + 1 + lipgloss.Width(e.desc)
	}

	// View.go wraps the help line in `Padding(0, 2)` => 4 cells of chrome.
	avail := model.width - 4
	if avail < 1 {
		avail = 1
	}

	// Pull the pinned "?" entry (if any) out of the flow so we can guarantee
	// it always renders at the right edge.
	pinnedIdx := -1
	for i, e := range entries {
		if e.key == "?" {
			pinnedIdx = i
			break
		}
	}
	var pinned *helpEntry
	flow := entries
	if pinnedIdx >= 0 {
		p := entries[pinnedIdx]
		pinned = &p
		flow = append([]helpEntry{}, entries[:pinnedIdx]...)
		flow = append(flow, entries[pinnedIdx+1:]...)
	}

	pinnedW := 0
	if pinned != nil {
		// Pinned entry plus its leading separator (only paid for when other
		// entries render before it).
		pinnedW = entryW(*pinned)
	}

	visible := make([]helpEntry, 0, len(flow)+1)
	used := 0
	for i, e := range flow {
		w := entryW(e)
		if i > 0 {
			w += sepW
		}
		// Reserve room for the pinned entry and its leading separator so
		// we never render a flow entry that would push "?" off-screen.
		reserved := 0
		if pinned != nil {
			reserved = pinnedW + sepW
		}
		if used+w+reserved > avail {
			break
		}
		used += w
		visible = append(visible, e)
	}
	if pinned != nil {
		visible = append(visible, *pinned)
	}

	parts := make([]string, 0, len(visible)*3)
	for i, e := range visible {
		if i > 0 {
			parts = append(parts, "  ", sepStyle.Render("·"), "  ")
		}
		parts = append(parts,
			keyStyle.Render(e.key),
			" ",
			descStyle.Render(e.desc),
		)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
