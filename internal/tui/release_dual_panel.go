package tui

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"
)

// releaseEntryHashLen is the bare slice length used for short hashes in the
// inspect list. 7 chars is the conventional git short hash; we slice
// directly (no TruncateString) so the rendering never appends "…".
const releaseEntryHashLen = 7

// releaseTagPattern captures the leading `[TAG]` cue commits in this repo
// use ("[ADD] internal: …", "[UI] tui: …"). Only uppercase letters are
// matched so a shell-style `[$VAR]` doesn't get mistaken for a type chip.
var releaseTagPattern = regexp.MustCompile(`^\s*\[([A-Z]+)\]\s*(.*)$`)

// extractCommitTag splits `[TAG] rest` into ("TAG", "rest"). Returns
// ("", subject) when the leading bracketed token isn't present.
func extractCommitTag(subject string) (string, string) {
	m := releaseTagPattern.FindStringSubmatch(subject)
	if len(m) == 3 {
		return m[1], m[2]
	}
	return "", subject
}

// releaseInspectEntry is one row in the left list of the Commits/Body
// inspect mode. The first row in the list is the synthetic "[output]"
// entry — `isRelease` flags it so the right viewport renders the
// release body instead of a commit message. The second row is a
// non-selectable spacer (`isSeparator`) so the output entry reads as a
// distinct band from the rest of the commits list.
type releaseInspectEntry struct {
	isRelease   bool
	isSeparator bool
	hash        string
	subject     string
	body        string
	// tag is the bracketed prefix extracted from the subject (e.g. "ADD"
	// from "[ADD] internal: foo"). Empty when the subject doesn't start
	// with a `[X]` token. tagless is the subject with the prefix
	// stripped — used so the list row doesn't print the tag twice.
	tag     string
	tagless string
}

// ReleaseDualPanel mirrors HistoryDualPanel for releases. Mode 1
// (Commits/Body) shows the commits selected for the release on the left
// and the corresponding content on the right — release body when the
// special `[release]` row is active, individual commit subject otherwise.
// Mode 2 (Stages/Response) is the AI-side view: a single "Release
// Builder" stage today, with the same telemetry strip layout as the
// commits side. Persistence for the AI side is wired through the
// stageStats array; the create-release flow doesn't flush yet.
type ReleaseDualPanel struct {
	theme  *styles.Theme
	mode   HistoryDualMode
	render DualPanelRenderFunc

	width, height int

	release    storage.Release
	hasRelease bool

	entries     []releaseInspectEntry
	entryIndex  int
	bodyVP      viewport.Model
	stages      []historyStage
	stageIndex  int
	stageVP     viewport.Model
	stageStats  [4]pipelineStage
	releaseBody string
}

func NewReleaseDualPanel(theme *styles.Theme) ReleaseDualPanel {
	return ReleaseDualPanel{
		theme:   theme,
		bodyVP:  viewport.New(),
		stageVP: viewport.New(),
	}
}

func (p *ReleaseDualPanel) SetMode(mode HistoryDualMode)      { p.mode = mode }
func (p *ReleaseDualPanel) SetRenderer(r DualPanelRenderFunc) { p.render = r }

// SetSize budgets rows for each mode. Stages/Response keeps the same 6-row
// chrome the commits inspect uses (header + blank + rule + blank +
// telemetry around the viewport) so the visual hierarchy reads identical
// across screens.
func (p *ReleaseDualPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
	rightW := width - dualPanelLeftWidth - 1
	if rightW < 10 {
		rightW = 10
	}
	bodyH := height - 1
	if bodyH < 1 {
		bodyH = 1
	}
	stageH := height - 6
	if stageH < 1 {
		stageH = 1
	}
	p.bodyVP.SetWidth(rightW)
	p.bodyVP.SetHeight(bodyH)
	p.stageVP.SetWidth(rightW)
	p.stageVP.SetHeight(stageH)
	p.refreshContent()
}

// SetRelease re-hydrates both modes against a release record. `messages`
// maps each commit hash from `r.CommitList` to the subject + body
// resolved via `git show` — pre-loaded by the caller so the dual panel
// stays git-call-free during cursor moves. `calls` is the release's AI
// telemetry payload (currently always empty; the create-release flow
// doesn't flush ai_calls yet, but the renderer is ready).
func (p *ReleaseDualPanel) SetRelease(
	r storage.Release,
	messages map[string]git.CommitMessage,
	calls []storage.AICall,
) {
	p.release = r
	p.hasRelease = true
	p.releaseBody = r.Body

	// Build the inspect entries:
	//   index 0 → synthetic row labelled by record type ("release" or
	//             "merge"), with an "output" suffix so the user reads
	//             it as the AI output for that record.
	//   index 1 → blank separator (skipped by the cursor cycle).
	//   index 2+ → one row per hash from CommitList, enriched with the
	//              subject + body resolved via git.LookupCommitMessages.
	releaseLabel := "release"
	if strings.EqualFold(strings.TrimSpace(r.Type), "MERGE") {
		releaseLabel = "merge"
	}
	p.entries = []releaseInspectEntry{
		{
			isRelease: true,
			hash:      releaseLabel,
			subject:   r.Title,
			body:      r.Body,
		},
		{
			isSeparator: true,
		},
	}
	for _, raw := range strings.Split(r.CommitList, ",") {
		h := strings.TrimSpace(raw)
		if h == "" {
			continue
		}
		msg := messages[h]
		tag, tagless := extractCommitTag(msg.Subject)
		p.entries = append(p.entries, releaseInspectEntry{
			hash:    h,
			subject: msg.Subject,
			body:    msg.Body,
			tag:     tag,
			tagless: tagless,
		})
	}
	if p.entryIndex < 0 || p.entryIndex >= len(p.entries) {
		p.entryIndex = 0
	}

	// Reset and rebuild per-stage telemetry — same path the commits dual
	// panel uses. Stages without a stored row stay zeroed so
	// renderStageStatsLine returns "" for them.
	for i := range p.stageStats {
		p.stageStats[i] = pipelineStage{ID: stageID(i), Status: statusIdle}
	}
	for _, call := range calls {
		id, ok := stageIDFromDBLabel(call.Stage)
		if !ok || int(id) < 0 || int(id) >= len(p.stageStats) {
			continue
		}
		st := &p.stageStats[id]
		st.HasStats = true
		st.StatsModel = call.Model
		st.Model = call.Model
		st.PromptTokens = call.PromptTokens
		st.CompletionTokens = call.CompletionTokens
		st.TotalTokens = call.TotalTokens
		st.QueueTime = msToDuration(call.QueueTimeMs)
		st.PromptTime = msToDuration(call.PromptTimeMs)
		st.CompletionTime = msToDuration(call.CompletionTimeMs)
		st.APITotalTime = msToDuration(call.TotalTimeMs)
		st.RequestID = call.RequestID
		st.TPMLimitAtCall = call.TPMLimitAtCall
		st.Latency = st.APITotalTime
		st.Status = statusDone
		st.Progress = 1
	}

	// Single AI stage today: "Release Builder". The output text is the
	// release body itself — that's what the AI produced. A future
	// refactor will split distinct stages (notes / changelog / etc) and
	// each will get its own row.
	p.stages = []historyStage{
		{idx: 1, name: "Release Builder", output: r.Body},
	}
	if p.stageIndex < 0 || p.stageIndex >= len(p.stages) {
		p.stageIndex = 0
	}
	p.refreshContent()
}

func (p *ReleaseDualPanel) Clear() {
	p.release = storage.Release{}
	p.hasRelease = false
	p.entries = nil
	p.entryIndex = 0
	p.stages = nil
	p.stageIndex = 0
	p.releaseBody = ""
	for i := range p.stageStats {
		p.stageStats[i] = pipelineStage{ID: stageID(i), Status: statusIdle}
	}
	p.bodyVP.SetContent("")
	p.stageVP.SetContent("")
}

func (p *ReleaseDualPanel) renderText(text string, width int) string {
	if p.render != nil {
		return p.render(text, width)
	}
	return lipgloss.NewStyle().Width(width).Render(text)
}

// JumpToRelease moves the inspect cursor to the synthetic release entry
// (always at index 0). Used by the dedicated "R" shortcut so the user can
// pop back to the release body from any commit row without holding
// ctrl+[.
func (p *ReleaseDualPanel) JumpToRelease() {
	if len(p.entries) == 0 {
		return
	}
	p.entryIndex = 0
	p.refreshContent()
}

func (p *ReleaseDualPanel) refreshContent() {
	if !p.hasRelease {
		return
	}
	// Right body in Commits/Body mode. Layout differs by entry kind:
	//   - release/merge entry → full release body, glamour-rendered.
	//   - commit entry → bold subject + blank + muted rule + blank +
	//     body (or placeholder when the commit body is empty).
	// The header row (rendered separately by View) used to spell out
	// the hash, which is noise for a preview. We keep "commit" /
	// "release" / "merge" up there now and let the viewport own the
	// content the user actually reads.
	if p.entryIndex >= 0 && p.entryIndex < len(p.entries) {
		entry := p.entries[p.entryIndex]
		var content string
		switch {
		case entry.isSeparator:
			// Defensive — the cycle skips separators so this shouldn't
			// fire, but if it does, leave the viewport blank rather
			// than crash on missing fields.
			content = ""
		case entry.isRelease:
			if strings.TrimSpace(entry.body) != "" {
				content = p.renderText(entry.body, p.bodyVP.Width())
			} else {
				content = lipgloss.NewStyle().
					Foreground(p.theme.Muted).
					Italic(true).
					Render("(release body not stored)")
			}
		case strings.TrimSpace(entry.subject) == "":
			content = lipgloss.NewStyle().
				Foreground(p.theme.Muted).
				Italic(true).
				Render(fmt.Sprintf("(commit %s not found in git history)", entry.hash))
		default:
			content = p.renderCommitPreview(entry, p.bodyVP.Width())
		}
		p.bodyVP.SetContent(content)
		p.bodyVP.GotoTop()
	}

	// Right body in Stages/Response mode.
	if p.stageIndex >= 0 && p.stageIndex < len(p.stages) {
		out := p.stages[p.stageIndex].output
		if strings.TrimSpace(out) == "" {
			out = lipgloss.NewStyle().
				Foreground(p.theme.Muted).
				Italic(true).
				Render("(stage output not stored)")
		} else {
			out = p.renderText(out, p.stageVP.Width())
		}
		p.stageVP.SetContent(out)
	}
	p.stageVP.GotoTop()
}

func (p ReleaseDualPanel) Update(msg tea.Msg) (ReleaseDualPanel, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "pgup":
			if p.mode == ModeKeyPointsBody {
				p.bodyVP.ScrollUp(p.bodyVP.Height() / 2)
			} else {
				p.stageVP.ScrollUp(p.stageVP.Height() / 2)
			}
			return p, nil
		case "pgdown":
			if p.mode == ModeKeyPointsBody {
				p.bodyVP.ScrollDown(p.bodyVP.Height() / 2)
			} else {
				p.stageVP.ScrollDown(p.stageVP.Height() / 2)
			}
			return p, nil
		}
	}
	return p, nil
}

// CycleLeftCursor mirrors HistoryDualPanel.CycleLeftCursor: stages list
// wraps around, commits list wraps around. In Commits/Body mode the
// cycle skips entries flagged as separators so the cursor jumps from
// `[output]` straight to the first commit (and back) without resting on
// the visual gap.
func (p *ReleaseDualPanel) CycleLeftCursor(delta int) {
	if p.mode == ModeKeyPointsBody {
		if len(p.entries) == 0 {
			return
		}
		step := 1
		if delta < 0 {
			step = -1
		}
		n := len(p.entries)
		idx := p.entryIndex
		// Move at least one slot in the requested direction, then keep
		// stepping while we land on a separator. n iterations is the
		// safety bound: if every entry is a separator we'd otherwise
		// loop forever.
		for k := 0; k < n; k++ {
			idx = ((idx+step)%n + n) % n
			if !p.entries[idx].isSeparator {
				break
			}
		}
		p.entryIndex = idx
		p.refreshContent()
		return
	}
	if len(p.stages) == 0 {
		return
	}
	n := len(p.stages)
	p.stageIndex = ((p.stageIndex+delta)%n + n) % n
	p.stageVP.SetContent(p.renderText(p.stages[p.stageIndex].output, p.stageVP.Width()))
	p.stageVP.GotoTop()
}

func (p ReleaseDualPanel) View() string {
	if p.width <= 0 || p.height <= 0 {
		return ""
	}

	leftW := dualPanelLeftWidth
	if leftW > p.width/2 {
		leftW = p.width / 2
	}
	rightW := p.width - leftW - 1

	var leftHeader, leftBody, rightHeader, rightBody string
	if p.mode == ModeKeyPointsBody {
		// Count only real commit rows for the header — the synthetic
		// `[output]` entry and the separator aren't commits.
		commitCount := 0
		for _, e := range p.entries {
			if !e.isRelease && !e.isSeparator {
				commitCount++
			}
		}
		leftHeader = p.renderHeader("commits", fmt.Sprintf("%d", commitCount), leftW)
		leftBody = p.renderEntriesBody(leftW, p.height-1)
		// Right header label: synthetic row reads "release output" /
		// "merge output" so the header echoes the inspect row identity;
		// individual commits drop the hash and just show "commit" since
		// the user reads the title + body in the viewport.
		entryName := "commit"
		if p.entryIndex >= 0 && p.entryIndex < len(p.entries) {
			e := p.entries[p.entryIndex]
			if e.isRelease {
				entryName = fmt.Sprintf("%s output", e.hash)
			}
		}
		rightHeader = p.renderHeader(entryName, "preview", rightW)
		rightBody = p.bodyVP.View()
	} else {
		leftHeader = p.renderHeader("ai stages", fmt.Sprintf("%d", len(p.stages)), leftW)
		leftBody = p.renderStagesBody(leftW, p.height-1)
		stageName := ""
		if p.stageIndex >= 0 && p.stageIndex < len(p.stages) {
			stageName = fmt.Sprintf("%d.%s", p.stages[p.stageIndex].idx, p.stages[p.stageIndex].name)
		}
		rightHeader = p.renderHeader(stageName, "output", rightW)
		rightBody = p.stageVP.View()
	}

	leftCol := lipgloss.JoinVertical(lipgloss.Left, leftHeader, leftBody)
	var rightCol string
	if p.mode == ModeStagesResponse {
		telemetry := p.renderTelemetryRow(rightW)
		rule := lipgloss.NewStyle().
			Foreground(p.theme.Subtle).
			Render(strings.Repeat("─", rightW))
		rightCol = lipgloss.JoinVertical(
			lipgloss.Left,
			rightHeader,
			"",
			rightBody,
			"",
			rule,
			"",
			telemetry,
		)
	} else {
		rightCol = lipgloss.JoinVertical(lipgloss.Left, rightHeader, rightBody)
	}

	leftStyled := lipgloss.Place(leftW, p.height, lipgloss.Left, lipgloss.Top, leftCol)
	rightStyled := lipgloss.Place(rightW, p.height, lipgloss.Left, lipgloss.Top, rightCol)

	bars := make([]string, p.height)
	barStyle := lipgloss.NewStyle().Foreground(p.theme.Subtle)
	for i := range bars {
		bars[i] = barStyle.Render("│")
	}
	dividerCol := lipgloss.JoinVertical(lipgloss.Left, bars...)
	divider := lipgloss.Place(1, p.height, lipgloss.Left, lipgloss.Top, dividerCol)

	row := lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, divider, rightStyled)
	return lipgloss.Place(p.width, p.height, lipgloss.Left, lipgloss.Top, row)
}

func (p ReleaseDualPanel) renderTelemetryRow(width int) string {
	muted := lipgloss.NewStyle().Foreground(p.theme.Muted).Italic(true).PaddingLeft(1)
	if p.stageIndex < 0 || p.stageIndex >= len(p.stages) {
		return muted.Render("(no stage selected)")
	}
	id := stageID(p.stages[p.stageIndex].idx - 1)
	if int(id) < 0 || int(id) >= len(p.stageStats) {
		return muted.Render("(stage out of range)")
	}
	st := p.stageStats[id]
	innerW := width - 2
	if innerW < 1 {
		innerW = 1
	}
	line := renderStageStatsLine(p.theme, &st, innerW, false)
	if line == "" {
		return muted.Render("(no telemetry stored)")
	}
	return lipgloss.NewStyle().PaddingLeft(1).Render(line)
}

func (p ReleaseDualPanel) renderHeader(label, suffix string, width int) string {
	labelStyle := lipgloss.NewStyle().Foreground(p.theme.Primary).Bold(true)
	suffixStyle := lipgloss.NewStyle().Foreground(p.theme.Muted)
	left := labelStyle.Render(label)
	right := suffixStyle.Render(suffix)
	pad := width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if pad < 1 {
		pad = 1
	}
	return lipgloss.NewStyle().PaddingLeft(1).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			left,
			lipgloss.NewStyle().Width(pad).Render(""),
			right,
		),
	)
}

func (p ReleaseDualPanel) renderEntriesBody(width, height int) string {
	if len(p.entries) == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Foreground(p.theme.Muted).
			PaddingLeft(1).
			Render("(no commits in this release)")
	}
	var lines []string
	for i, e := range p.entries {
		nameStyle := lipgloss.NewStyle().Foreground(p.theme.Muted)
		glyphColor := p.theme.Muted
		if i == p.entryIndex {
			nameStyle = lipgloss.NewStyle().Foreground(p.theme.FG).Bold(true)
			glyphColor = p.theme.Secondary
		}
		var rendered string
		switch {
		case e.isSeparator:
			// Muted rule sized to the inner column width so the gap
			// reads as a visual band between the [output] entry and
			// the commits list. PaddingLeft(1) is kept off so the rule
			// spans the full inner width.
			ruleW := width - 2
			if ruleW < 1 {
				ruleW = 1
			}
			rendered = lipgloss.NewStyle().
				Foreground(p.theme.Subtle).
				PaddingLeft(1).
				Render(strings.Repeat("─", ruleW))
		case e.isRelease:
			// Render two tokens: the type-aware bracketed label ("[release]"
			// or "[merge]") followed by a muted "· output" suffix so the
			// user can tell at a glance that this row is the AI output
			// of the record, not one of the inner commits.
			glyph := lipgloss.NewStyle().Foreground(glyphColor).Bold(true).Render("✦")
			tag := nameStyle.Render(fmt.Sprintf("[%s]", e.hash))
			outputStyle := lipgloss.NewStyle().Foreground(p.theme.Muted)
			if i == p.entryIndex {
				outputStyle = outputStyle.Bold(true)
			}
			suffix := outputStyle.Render("· output")
			label := fmt.Sprintf("%s %s %s", glyph, tag, suffix)
			rendered = lipgloss.NewStyle().PaddingLeft(1).Render(label)
		default:
			label := p.renderCommitListRow(e, i == p.entryIndex)
			rendered = lipgloss.NewStyle().PaddingLeft(1).Render(label)
		}
		lines = append(lines, rendered)
		if len(lines) >= height {
			break
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderCommitListRow paints one commit row in the inspect list. Format:
//
//	·  abc1234
//
// The hash is the only payload; when the subject begins with a `[TAG]`
// cue the tag's palette is applied to the hash itself (same chrome the
// History list uses for commit type chips), so the row reads as a
// colored pill instead of carrying a separate badge. Hashes without a
// resolved tag fall back to the muted Accent color.
func (p ReleaseDualPanel) renderCommitListRow(
	e releaseInspectEntry,
	selected bool,
) string {
	glyphColor := p.theme.Muted
	if selected {
		glyphColor = p.theme.Secondary
	}
	glyph := lipgloss.NewStyle().Foreground(glyphColor).Render("·")

	short := e.hash
	if len(short) > releaseEntryHashLen {
		short = short[:releaseEntryHashLen]
	}

	var hashStyle lipgloss.Style
	switch {
	case e.tag != "" && selected:
		hashStyle = styles.CommitTypeBlockStyle(p.theme, e.tag).Bold(true)
	case e.tag != "":
		hashStyle = styles.CommitTypeMsgStyle(p.theme, e.tag)
	case selected:
		hashStyle = lipgloss.NewStyle().Foreground(p.theme.Accent).Bold(true)
	default:
		hashStyle = lipgloss.NewStyle().Foreground(p.theme.Accent)
	}
	hashRendered := hashStyle.Padding(0, 1).Render(short)

	return fmt.Sprintf("%s %s", glyph, hashRendered)
}

// renderCommitPreview builds the right viewport content for a commit
// entry: bold subject (with optional [TAG] pill restored at the front
// so the preview reads identical to a styled commit header), a blank,
// a muted horizontal rule, another blank, then the body. Empty bodies
// fall back to a muted "(no body)" so the preview never collapses.
func (p ReleaseDualPanel) renderCommitPreview(e releaseInspectEntry, width int) string {
	if width < 1 {
		width = 1
	}
	titleStyle := lipgloss.NewStyle().Foreground(p.theme.Primary).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(p.theme.Muted).Italic(true)

	subject := strings.TrimSpace(e.tagless)
	if e.tag == "" {
		subject = strings.TrimSpace(e.subject)
	}
	if subject == "" {
		subject = "(no subject)"
	}

	var titleRow string
	if e.tag != "" {
		pill := styles.CommitTypeMsgStyle(p.theme, e.tag).Padding(0, 1).Render(e.tag)
		titleRow = pill + " " + titleStyle.Render(subject)
	} else {
		titleRow = titleStyle.Render(subject)
	}

	rule := lipgloss.NewStyle().
		Foreground(p.theme.Subtle).
		Render(strings.Repeat("─", width))

	body := strings.TrimSpace(e.body)
	var bodyRendered string
	if body == "" {
		bodyRendered = mutedStyle.Render("(no body)")
	} else {
		bodyRendered = p.renderText(body, width)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleRow,
		"",
		rule,
		"",
		bodyRendered,
	)
}

func (p ReleaseDualPanel) renderStagesBody(width, height int) string {
	if len(p.stages) == 0 {
		return ""
	}
	var lines []string
	for i, s := range p.stages {
		nameStyle := lipgloss.NewStyle().Foreground(p.theme.Muted)
		glyphColor := p.theme.Muted
		if i == p.stageIndex {
			nameStyle = lipgloss.NewStyle().Foreground(p.theme.FG).Bold(true)
			glyphColor = p.theme.Secondary
		}
		glyph := lipgloss.NewStyle().Foreground(glyphColor).Bold(true).Render("✦")
		head := fmt.Sprintf("%s [%d]  %s", glyph, s.idx, nameStyle.Render(s.name))
		lines = append(lines, lipgloss.NewStyle().PaddingLeft(1).Render(head))
		if len(lines) >= height {
			break
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
