package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// renderComposeTypeRow renders the "commit type" section: label on top,
// horizontal pills underneath that wrap to additional rows when they
// don't fit the column width.
func (model *Model) renderComposeTypeRow(width int, focused bool) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	label := theme.SectionPill(focused).Render("commit type")

	if len(model.finalCommitTypes) == 0 {
		empty := base.Foreground(theme.Muted).Italic(true).Render("(no commit types configured)")
		return lipgloss.JoinVertical(lipgloss.Left, label, empty)
	}

	pills := make([]string, 0, len(model.finalCommitTypes))
	for _, ct := range model.finalCommitTypes {
		active := ct.Tag == model.commitType
		pills = append(pills, model.commitTypeChip(ct.Tag, ct.Color, active))
	}

	// Pills wrap onto multiple rows when the joined width exceeds the
	// available column. Without wrapping the row would overflow and get
	// truncated by the panel's hard right edge.
	rowsText := wrapPillsToRows(pills, width, " ")

	return lipgloss.JoinVertical(lipgloss.Left, append([]string{label, ""}, rowsText...)...)
}

// commitTypeChip renders a single commit-type pill. Active gets a filled
// background using the type's configured color (with a primary fallback);
// inactive pills sit in a thin border so the row reads as a control bar
// even when nothing is selected yet.
func (model *Model) commitTypeChip(tag string, hex string, active bool) string {
	theme := model.Theme
	base := theme.AppStyles().Base
	if active {
		bg := lipgloss.Color(hex)
		if hex == "" {
			bg = colorOrFallback(theme.Primary)
		}
		return base.
			Foreground(theme.BG).
			Background(bg).
			Bold(true).
			Padding(0, 1).
			Render(tag)
	}
	return base.
		Foreground(theme.Muted).
		Border(lipgloss.NormalBorder(), false, false, false, false).
		Padding(0, 1).
		Render(tag)
}

// wrapPillsToRows splits an ordered list of already-rendered pills into
// rows that fit within maxWidth. The separator is included between pills
// on the same row but not at the start or end.
func wrapPillsToRows(pills []string, maxWidth int, sep string) []string {
	if len(pills) == 0 {
		return nil
	}
	if maxWidth <= 0 {
		return []string{lipgloss.JoinHorizontal(lipgloss.Top, joinWithSep(pills, sep)...)}
	}
	sepW := lipgloss.Width(sep)
	var rows []string
	var current []string
	currentW := 0
	for _, p := range pills {
		w := lipgloss.Width(p)
		needed := w
		if len(current) > 0 {
			needed += sepW
		}
		if currentW+needed > maxWidth && len(current) > 0 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, joinWithSep(current, sep)...))
			current = nil
			currentW = 0
			needed = w
		}
		current = append(current, p)
		currentW += needed
	}
	if len(current) > 0 {
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, joinWithSep(current, sep)...))
	}
	return rows
}

// renderComposeScopeRow renders the "scope" section: label on top, the
// single scope chip (or nothing) and an "edit" button. Scope is a
// single value — when the user picks a new scope the chip is replaced.
func (model *Model) renderComposeScopeRow(width int, focused bool) string {
	theme := model.Theme

	labelText := "scope"
	label := theme.SectionPill(focused).Render(labelText)

	var cells []string
	if len(model.commitScopes) > 0 {
		cells = append(cells, model.scopeChip(model.commitScopes[0], focused))
	}
	cells = append(cells, model.scopeEditButton(focused && len(model.commitScopes) == 0))
	row := lipgloss.JoinHorizontal(lipgloss.Top, joinWithSep(cells, " ")...)

	return lipgloss.JoinVertical(lipgloss.Left,
		label,
		"",
		row,
	)
}

// scopeChip renders one scope entry styled like a bordered pill, e.g.
// `· internal/tui/layout.go ×`.
func (model *Model) scopeChip(value string, highlighted bool) string {
	theme := model.Theme
	border := theme.Subtle
	fg := theme.FG
	if highlighted {
		border = theme.Primary
		fg = theme.Primary
	}
	return theme.AppStyles().Base.
		Foreground(fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render(fmt.Sprintf("· %s ×", value))
}

// scopeEditButton mirrors the chip styling but reads "edit" — the user
// activates it with `e` to open the file picker popup.
func (model *Model) scopeEditButton(highlighted bool) string {
	theme := model.Theme
	border := theme.Subtle
	fg := theme.Muted
	if highlighted {
		border = theme.Primary
		fg = theme.Primary
	}
	return theme.AppStyles().Base.
		Foreground(fg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Render("edit")
}

// renderComposeSummaryArea renders the textarea section labelled "summary"
// where the user types short paragraphs feeding the AI.
func (model *Model) renderComposeSummaryArea(width int, focused bool) string {
	theme := model.Theme

	labelText := "summary"
	label := theme.SectionPill(focused).Render(labelText)

	model.commitsKeysInput.SetWidth(width)
	body := model.commitsKeysInput.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		label,
		"",
		body,
	)
}

// renderComposeKeypointsArea renders the bulleted key-points list with a
// "X of N" counter. N is the soft cap (5) we encourage but don't enforce.
func (model *Model) renderComposeKeypointsArea(width, height int, focused bool) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	labelText := "key points"
	label := theme.SectionPill(focused).Render(labelText)

	counter := base.Foreground(theme.Muted).Render(
		fmt.Sprintf("%d items", len(model.keyPoints)),
	)
	spacerW := width - lipgloss.Width(label) - lipgloss.Width(counter)
	if spacerW < 1 {
		overflow := 1 - spacerW
		trimmed := ansi.Truncate(labelText, max(1, lipgloss.Width(label)-overflow), "…")
		label = theme.SectionPill(focused).Render(trimmed)
		spacerW = max(1, width-lipgloss.Width(label)-lipgloss.Width(counter))
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		label,
		strings.Repeat(" ", spacerW),
		counter,
	)

	listLines := make([]string, 0, len(model.keyPoints)+1)
	for i, kp := range model.keyPoints {
		isActive := focused && i == model.keypointIndex
		markerColor := theme.Muted
		textColor := theme.FG
		removeColor := theme.Muted
		if isActive {
			markerColor = theme.Primary
			removeColor = theme.Primary
		}
		marker := base.Foreground(markerColor).Render("▸")
		remove := base.Foreground(removeColor).Render("×")

		// 4 = marker(1) + space(1) + remove(1) + minimum 1-col gap.
		maxTextW := max(1, width-4)
		shown := kp
		if ansi.StringWidth(shown) > maxTextW {
			shown = ansi.Truncate(shown, maxTextW, "…")
		}
		text := base.Foreground(textColor).Render(shown)

		left := marker + " " + text
		spacer := strings.Repeat(" ",
			max(1, width-lipgloss.Width(left)-lipgloss.Width(remove)),
		)
		listLines = append(listLines, left+spacer+remove)
	}
	// Always show the "add" placeholder at the bottom; key points have
	// no soft cap so the user can keep adding as long as they want.
	{
		marker := base.Foreground(theme.Muted).Render("▸")
		placeholder := base.Foreground(theme.Muted).Italic(true).Render("add a key point…")
		listLines = append(listLines, marker+" "+placeholder)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, listLines...)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", body)
}

// renderComposePipelineModelsArea shows a numbered list of the AI stages
// with the model name configured for each.
func (model *Model) renderComposePipelineModelsArea(width int, focused bool) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	labelText := "pipeline models"
	label := theme.SectionPill(focused).Render(labelText)

	prompts := model.globalConfig.Prompts
	stages := []struct {
		index int
		name  string
		model string
	}{
		{1, "summary", prompts.ChangeAnalyzerPromptModel},
		{2, "raw commit", prompts.CommitBodyGeneratorPromptModel},
		{3, "formatted", prompts.CommitTitleGeneratorPromptModel},
	}

	rows := make([]string, 0, len(stages))
	for _, s := range stages {
		idx := base.Foreground(theme.Muted).Render(fmt.Sprintf("[%d]", s.index))
		name := base.Foreground(theme.FG).Render(s.name)
		sep := base.Foreground(theme.Muted).Render("·")
		modelName := s.model
		if modelName == "" {
			modelName = "(unset)"
		}
		mod := base.Foreground(theme.Primary).Render(modelName)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			idx, " ", name, " ", sep, " ", mod,
		))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		label,
		"",
		strings.Join(rows, "\n"),
	)
}

// renderAISuggestionContent picks between the empty-state placeholder and
// the rendered AI output, returning content meant to live inside the
// titled "ai suggestion" panel.
func (model *Model) renderAISuggestionContent(width, height int) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	if strings.TrimSpace(model.commitTranslate) == "" {
		// Empty state: centered sparkle + hint.
		sparkle := base.Foreground(theme.Primary).Render("✦")
		line1 := base.Foreground(theme.FG).Render(
			"describe your changes on the left,",
		)
		hint := base.Foreground(theme.Muted).Render("^W")
		line2 := base.Foreground(theme.FG).Render("then press ") +
			base.Foreground(theme.Primary).Render("^W") +
			base.Foreground(theme.FG).Render(" to generate a commit.")
		_ = hint
		return lipgloss.Place(width, height,
			lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center,
				sparkle,
				"",
				line1,
				line2,
			),
		)
	}

	// When populated, show the AI viewport content (already styled by the
	// caller). We pad to the panel size so the surrounding border doesn't
	// get out of sync with the content height.
	content := model.iaViewport.View()
	rendered := lipgloss.NewStyle().Width(width).Render(content)
	return rendered
}

// --- helpers ----------------------------------------------------------

// renderComposeDivider draws a thin horizontal rule that separates the
// metadata block (type + scope) from the content block (summary,
// keypoints, pipeline models) inside the compose left panel.
func (model *Model) renderComposeDivider(width int) string {
	if width <= 0 {
		return ""
	}
	return model.Theme.AppStyles().Base.
		Foreground(model.Theme.Subtle).
		Render(strings.Repeat("─", width))
}

// joinWithSep returns items interleaved with sep so JoinHorizontal can
// keep visual gaps without us having to remember to splice the separator
// at every call site.
func joinWithSep(items []string, sep string) []string {
	if len(items) == 0 {
		return items
	}
	out := make([]string, 0, len(items)*2-1)
	out = append(out, items[0])
	for _, it := range items[1:] {
		out = append(out, sep, it)
	}
	return out
}

// colorOrFallback wraps a color.Color so callers can pass theme primary
// when a hex string is empty.
func colorOrFallback(c color.Color) color.Color {
	if c == nil {
		return lipgloss.Color("#b79cf4")
	}
	return c
}

// scopeSeparator is the joiner for commitScope (the legacy joined-string
// form persisted in the DB and fed to AI prompts). Picked so the joined
// string still parses back into the same slice via splitJoinedScopes.
const scopeSeparator = ", "

// syncCommitScope rewrites commitScope from the slice. Call after every
// mutation of commitScopes so downstream code (storage, AI prompts) sees
// a consistent value.
func (m *Model) syncCommitScope() {
	m.commitScope = strings.Join(m.commitScopes, scopeSeparator)
}

// loadScopesFromString rehydrates the slice from a joined value.
func (m *Model) loadScopesFromString(joined string) {
	joined = strings.TrimSpace(joined)
	m.commitScopes = nil
	if joined == "" {
		m.syncCommitScope()
		return
	}
	for _, part := range strings.Split(joined, scopeSeparator) {
		part = strings.TrimSpace(part)
		if part != "" {
			m.commitScopes = append(m.commitScopes, part)
		}
	}
	m.syncCommitScope()
	if m.scopeChipIndex >= len(m.commitScopes) {
		m.scopeChipIndex = max(0, len(m.commitScopes)-1)
	}
}

// addScope sets the (single) scope value, replacing whatever was there
// before. The slice infrastructure remains so render code can iterate
// uniformly, but a commit only ever carries one scope.
func (m *Model) addScope(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	m.commitScopes = []string{s}
	m.syncCommitScope()
	m.scopeChipIndex = 0
}

// removeScopeAt drops the chip at index i and clamps the cursor.
func (m *Model) removeScopeAt(i int) {
	if i < 0 || i >= len(m.commitScopes) {
		return
	}
	m.commitScopes = append(m.commitScopes[:i], m.commitScopes[i+1:]...)
	m.syncCommitScope()
	if len(m.commitScopes) == 0 {
		m.scopeChipIndex = 0
	} else if m.scopeChipIndex >= len(m.commitScopes) {
		m.scopeChipIndex = len(m.commitScopes) - 1
	}
}

// resetScopes wipes both representations.
func (m *Model) resetScopes() {
	m.commitScopes = nil
	m.commitScope = ""
	m.scopeChipIndex = 0
}

// lookupTypeColor resolves a commit type tag to its configured color.
func lookupTypeColor(colors map[string]string, tag string, fallback color.Color) color.Color {
	if colors == nil {
		return fallback
	}
	if hex, ok := colors[tag]; ok && hex != "" {
		return lipgloss.Color(hex)
	}
	return fallback
}
