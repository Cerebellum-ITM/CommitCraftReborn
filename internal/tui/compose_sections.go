package tui

import (
	"fmt"
	"hash/fnv"
	"image/color"
	"regexp"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/tui/statusbar"
	"commit_craft_reborn/internal/tui/styles"
)

// mentionRenderRegex matches `@token` chips (filename, path, identifier)
// that should be highlighted with the success-pill palette inside any
// surface we render ourselves: the saved key-points list and the
// AI-suggestion panel. Liberal on the chars allowed inside the token so
// it covers paths like `internal/tui/view.go` and snake_case names.
var mentionRenderRegex = regexp.MustCompile(`@[\w./-]+`)

// styleMentions wraps every `@token` substring of s with the success
// pill, applying textStyle to the surrounding prose. Returns plain
// textStyle.Render(s) when no mention is present.
func styleMentions(s string, textStyle lipgloss.Style) string {
	if s == "" {
		return ""
	}
	matches := mentionRenderRegex.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		return textStyle.Render(s)
	}
	var b strings.Builder
	cursor := 0
	for _, m := range matches {
		if m[0] > cursor {
			b.WriteString(textStyle.Render(s[cursor:m[0]]))
		}
		b.WriteString(statusbar.RenderMentionPill(s[m[0]:m[1]]))
		cursor = m[1]
	}
	if cursor < len(s) {
		b.WriteString(textStyle.Render(s[cursor:]))
	}
	return b.String()
}

// renderComposeTypeRow renders the "commit type" section as a single
// horizontal line: the section label followed by the chip of the
// currently selected commit type. Other types are not displayed here —
// the picker popup is the place to switch.
func (model *Model) renderComposeTypeRow(width int, focused bool) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	label := theme.SectionPill(focused).Render("commit type")

	if len(model.finalCommitTypes) == 0 {
		empty := base.Foreground(theme.Muted).Italic(true).Render("(no commit types configured)")
		return lipgloss.JoinHorizontal(lipgloss.Center, label, " ", empty)
	}

	chip := model.commitTypeChip(model.commitType, "", true)
	return lipgloss.JoinHorizontal(lipgloss.Center, label, " ", chip)
}

// commitTypeChip renders a single commit-type pill. The active tag uses
// the strong (block) palette of its commit type; the rest use the dim
// (msg) palette. Width is fixed (CommitTypeChipInnerWidth) and content
// is centered, so every pill measures the same and reads as a uniform
// row of badges.
func (model *Model) commitTypeChip(tag string, _ string, active bool) string {
	upper := strings.ToUpper(tag)
	if len(upper) > styles.CommitTypeChipInnerWidth {
		upper = upper[:styles.CommitTypeChipInnerWidth]
	}
	var chipStyle lipgloss.Style
	if active {
		chipStyle = styles.CommitTypeBlockStyle(model.Theme, tag).Bold(true)
	} else {
		chipStyle = styles.CommitTypeMsgStyle(model.Theme, tag)
	}
	return chipStyle.
		Width(styles.CommitTypeChipInnerWidth).
		Padding(0, 1).
		Align(lipgloss.Center).
		Render(upper)
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

// renderComposeScopeRow renders the "scope" section on a single
// horizontal line: section label, the file chip (when set) and the
// "edit" affordance. The chip uses the commit-type pill palette with a
// per-file deterministic colour so the scope reads as a coloured tag
// rather than a neutral box.
func (model *Model) renderComposeScopeRow(width int, focused bool) string {
	theme := model.Theme

	label := theme.SectionPill(focused).Render("scope")

	cells := []string{label}
	if len(model.commitScopes) > 0 {
		cells = append(cells, " ", model.scopeChip(model.commitScopes[0], focused))
	}
	cells = append(cells, " ", model.scopeEditButton(focused && len(model.commitScopes) == 0))

	return lipgloss.JoinHorizontal(lipgloss.Center, cells...)
}

// scopeChip renders the selected file as a flat coloured pill in the
// commit-type style. The palette is picked deterministically from the
// file path so the same file always renders with the same colour, but
// different files get visually distinguishable tags.
func (model *Model) scopeChip(value string, _ bool) string {
	tag := pickCommitTypeTag(value)
	return styles.CommitTypeBlockStyle(model.Theme, tag).
		Bold(true).
		Padding(0, 1).
		Render(value)
}

// scopeEditButton renders the "edit" affordance as a flat pill so it
// aligns visually with the scope chip on the same single-line row.
func (model *Model) scopeEditButton(highlighted bool) string {
	theme := model.Theme
	bg := theme.Surface
	fg := theme.Muted
	if highlighted {
		bg = theme.Primary
		fg = theme.BG
	}
	return theme.AppStyles().Base.
		Background(bg).
		Foreground(fg).
		Padding(0, 1).
		Render("edit")
}

// pickCommitTypeTag deterministically maps a string (typically a file
// path) to one of the commit-type palette keys, so chips for the same
// file always look identical between renders while different files get
// distinct colours.
func pickCommitTypeTag(seed string) string {
	tags := []string{
		"ADD", "FIX", "DOC", "WIP", "STYLE", "REFACTOR",
		"TEST", "PERF", "CHORE", "DEL", "BUILD", "CI", "REVERT", "SEC",
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(seed))
	return tags[int(h.Sum32())%len(tags)]
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
		// Saved items are always recognisable: the secondary brand colour
		// when the section is blurred (so they pop against the muted
		// neighbours), the primary colour when the section owns focus and
		// the cursor is on this row, and a quieter muted tone for the
		// remaining rows in the focused section so the active one stands
		// out alone.
		markerColor := theme.Secondary
		textColor := theme.FG
		removeColor := theme.Secondary
		switch {
		case isActive:
			markerColor = theme.Warning
			removeColor = theme.Warning
		case focused:
			markerColor = theme.Muted
			removeColor = theme.Muted
		}
		marker := base.Foreground(markerColor).Render("▸")
		remove := base.Foreground(removeColor).Render("×")

		// 4 = marker(1) + space(1) + remove(1) + minimum 1-col gap.
		maxTextW := max(1, width-4)
		text := styleMentions(kp, base.Foreground(textColor))
		if ansi.StringWidth(text) > maxTextW {
			text = ansi.Truncate(text, maxTextW, "…")
		}

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
// with the model name configured for each. When the section has focus
// the active stage is highlighted with a cursor and an `enter to change`
// hint appears underneath.
func (model *Model) renderComposePipelineModelsArea(width int, focused bool) string {
	theme := model.Theme
	base := theme.AppStyles().Base

	labelText := "pipeline models"
	label := theme.SectionPill(focused).Render(labelText)

	stages := composePipelineStages(model)
	if focused && model.pipelineModelStageIndex >= len(stages) {
		model.pipelineModelStageIndex = 0
	}

	// Each stage gets 3 lines: the model label, an RPD bar, and a TPM
	// bar. Bars are fed by the in-memory rate-limit cache that
	// ratelimit_cache.go hydrates from x-ratelimit-* response headers
	// on every Groq call.
	rows := make([]string, 0, len(stages)*3)
	barWidth := width - 2
	if barWidth < 24 {
		barWidth = 24
	}
	for i, s := range stages {
		isActive := focused && i == model.pipelineModelStageIndex
		cursor := base.Foreground(theme.Muted).Render(" ")
		idxColor := theme.Muted
		nameColor := theme.FG
		modColor := theme.Primary
		if isActive {
			cursor = base.Foreground(theme.Warning).Bold(true).Render("▸")
			idxColor = theme.Warning
			nameColor = theme.Warning
			modColor = theme.Warning
		}
		idx := base.Foreground(idxColor).Render(fmt.Sprintf("[%d]", i+1))
		name := base.Foreground(nameColor).Render(s.label)
		sep := base.Foreground(theme.Muted).Render("·")
		modelName := config.CurrentModelForStage(model.globalConfig, s.stage)
		modelLabel := modelName
		if modelLabel == "" {
			modelLabel = "(unset)"
		}
		mod := base.Foreground(modColor).Render(modelLabel)
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			cursor, " ", idx, " ", name, " ", sep, " ", mod,
		))

		var rpd, tpm string
		if modelName == "" {
			rpd = renderThinQuotaBar(theme, "RPD", 0, 0, barWidth)
			tpm = renderThinQuotaBar(theme, "TPM", 0, 0, barWidth)
		} else if rl, ok := api.GetRateLimits(modelName); ok {
			eff := api.EffectiveRateLimits(rl, time.Now())
			// Both RPD and TPM derive from the captured `remaining-*`
			// headers; the Parsed flags guard against the case where
			// Groq omits a header (parser would otherwise silently
			// produce `used = limit - 0 = limit` and show a 100% bar).
			if eff.RequestsParsed {
				rpd = renderThinQuotaBarLog(theme, "RPD",
					eff.LimitRequests-eff.RemainingRequests, eff.LimitRequests, barWidth)
			} else {
				rpd = renderThinQuotaBarLog(theme, "RPD", 0, 0, barWidth)
			}
			if eff.TokensParsed {
				tpm = renderThinQuotaBar(theme, "TPM",
					eff.LimitTokens-eff.RemainingTokens, eff.LimitTokens, barWidth)
			} else {
				tpm = renderThinQuotaBar(theme, "TPM", 0, 0, barWidth)
			}
		} else {
			rpd = renderThinQuotaBar(theme, "RPD", 0, 0, barWidth)
			tpm = renderThinQuotaBar(theme, "TPM", 0, 0, barWidth)
		}
		rows = append(rows, "    "+rpd, "    "+tpm)
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
