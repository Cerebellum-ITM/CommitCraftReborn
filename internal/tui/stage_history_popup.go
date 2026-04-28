package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	glamourstyles "charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/statusbar"
	"commit_craft_reborn/internal/tui/styles"
)

// stageHistoryApplyMsg is fired when the user picks an entry in the
// history popup. The Update loop swaps that entry's text and stats
// onto the live pipeline state and dismisses the popup.
type stageHistoryApplyMsg struct {
	stage stageID
	index int
}

// closeStageHistoryMsg dismisses the popup without changing anything.
type closeStageHistoryMsg struct{}

// stageHistoryPopupModel lets the user browse every AI generation
// captured for a single stage during the current session and switch
// the active one. Lives only while the popup is on screen.
type stageHistoryPopupModel struct {
	stage         stageID
	stageLabel    string
	entries       []stageHistoryEntry
	activeIndex   int
	cursor        int
	width, height int
	theme         *styles.Theme
	preview       viewport.Model
	previewWidth  int
}

func newStageHistoryPopup(
	stage stageID,
	stageLabel string,
	entries []stageHistoryEntry,
	activeIndex int,
	width, height int,
	theme *styles.Theme,
) stageHistoryPopupModel {
	cursor := activeIndex
	if cursor < 0 || cursor >= len(entries) {
		cursor = len(entries) - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	m := stageHistoryPopupModel{
		stage:        stage,
		stageLabel:   stageLabel,
		entries:      entries,
		activeIndex:  activeIndex,
		cursor:       cursor,
		width:        width,
		height:       height,
		theme:        theme,
		preview:      viewport.New(),
		previewWidth: max(20, width-6),
	}
	m.preview.SetWidth(m.previewWidth)
	m.refreshPreviewContent()
	return m
}

// refreshPreviewContent re-renders the preview viewport for the current
// cursor entry. Called whenever the cursor moves so View() stays
// stateless. Resets the scroll position so each entry shows from the
// top — pgup/pgdn within the same entry still scroll normally.
func (m *stageHistoryPopupModel) refreshPreviewContent() {
	if len(m.entries) == 0 {
		m.preview.SetContent("")
		return
	}
	width := m.previewWidth
	if width <= 0 {
		width = 40
	}
	entry := m.entries[m.cursor]
	m.preview.SetContent(renderStageHistoryPreview(entry.Text, width))
	m.preview.GotoTop()
}

func (m stageHistoryPopupModel) Init() tea.Cmd { return nil }

func (m stageHistoryPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc", "q":
			return m, func() tea.Msg { return closeStageHistoryMsg{} }
		// Rendering is newest-first (top row = highest index), so visual
		// "down" means a lower index, and "up" means a higher one. Wrap
		// around at the edges so navigation feels like the rest of the
		// charm-style lists in the app.
		case "up", "k":
			if len(m.entries) == 0 {
				return m, nil
			}
			if m.cursor >= len(m.entries)-1 {
				m.cursor = 0
			} else {
				m.cursor++
			}
			m.refreshPreviewContent()
			return m, nil
		case "down", "j":
			if len(m.entries) == 0 {
				return m, nil
			}
			if m.cursor <= 0 {
				m.cursor = len(m.entries) - 1
			} else {
				m.cursor--
			}
			m.refreshPreviewContent()
			return m, nil
		case "home", "g":
			if len(m.entries) > 0 {
				m.cursor = len(m.entries) - 1
			}
			m.refreshPreviewContent()
			return m, nil
		case "end", "G":
			m.cursor = 0
			m.refreshPreviewContent()
			return m, nil
		case "pgup":
			m.preview.HalfPageUp()
			return m, nil
		case "pgdown":
			m.preview.HalfPageDown()
			return m, nil
		case "enter":
			if len(m.entries) == 0 {
				return m, func() tea.Msg { return closeStageHistoryMsg{} }
			}
			stage := m.stage
			idx := m.cursor
			return m, func() tea.Msg {
				return stageHistoryApplyMsg{stage: stage, index: idx}
			}
		}
	}
	var cmd tea.Cmd
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}

func (m stageHistoryPopupModel) View() tea.View {
	box := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	innerW := max(40, m.width-box.GetHorizontalFrameSize())
	innerH := max(12, m.height-box.GetVerticalFrameSize())

	base := m.theme.AppStyles().Base
	title := base.Foreground(m.theme.Secondary).Bold(true).Render(
		fmt.Sprintf("History · %s", m.stageLabel),
	)
	count := base.Foreground(m.theme.Muted).Render(
		fmt.Sprintf("· %d version%s", len(m.entries), pluralS(len(m.entries))),
	)
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, " ", count)

	hint := renderPopupHelpLine(m.theme, []helpEntry{
		{"↑↓/jk", "navigate"},
		{"pgup/pgdn", "scroll preview"},
		{"↵", "apply"},
		{"esc", "cancel"},
	})

	// Layout budget: header + spacer + list + spacer + divider + spacer +
	// preview + spacer + hint. The list takes ~1 line per entry capped to
	// half of the inner height; the preview owns whatever is left.
	listMaxRows := max(3, innerH/3)
	if listMaxRows > len(m.entries) {
		listMaxRows = len(m.entries)
	}
	if listMaxRows < 1 {
		listMaxRows = 1
	}

	listView := m.renderList(innerW, listMaxRows)
	listH := strings.Count(listView, "\n") + 1

	divider := base.Foreground(m.theme.Subtle).
		Render(strings.Repeat("─", innerW))

	hintH := lipgloss.Height(hint)
	previewH := innerH - lipgloss.Height(header) - 1 - listH - 1 - 1 - 1 - hintH
	if previewH < 3 {
		previewH = 3
	}
	previewView := m.renderPreview(innerW, previewH)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		listView,
		"",
		divider,
		"",
		previewView,
		"",
		hint,
	)
	return tea.NewView(box.Render(body))
}

// renderList draws the version list, newest-first, capped to maxRows.
// When the cursor sits beyond the visible window we scroll the slice so
// the cursor is always shown.
func (m stageHistoryPopupModel) renderList(width, maxRows int) string {
	if len(m.entries) == 0 {
		return m.theme.AppStyles().Base.Foreground(m.theme.Muted).Italic(true).
			Render("No history yet.")
	}
	base := m.theme.AppStyles().Base

	// "displayIdx" walks the entries newest-first; the visible window
	// scrolls so the cursor row is always present.
	cursorDisplay := (len(m.entries) - 1) - m.cursor // 0 = top row
	startDisplay := 0
	if cursorDisplay >= maxRows {
		startDisplay = cursorDisplay - maxRows + 1
	}
	endDisplay := startDisplay + maxRows
	if endDisplay > len(m.entries) {
		endDisplay = len(m.entries)
	}

	var b strings.Builder
	for d := startDisplay; d < endDisplay; d++ {
		entryIdx := (len(m.entries) - 1) - d
		entry := m.entries[entryIdx]
		isCursor := entryIdx == m.cursor
		isActive := entryIdx == m.activeIndex

		marker := "  "
		if isCursor {
			marker = base.Foreground(m.theme.Warning).Bold(true).Render("▸ ")
		}
		ts := entry.CapturedAt.Local().Format("15:04:05")
		head := fmt.Sprintf(
			"[%d] %s · %s tok (in %s · out %s) · %s",
			entryIdx+1,
			ts,
			formatQuantity(entry.TotalTokens),
			formatQuantity(entry.PromptTokens),
			formatQuantity(entry.CompletionTokens),
			formatHistoryDuration(entry.APITotalTime),
		)
		headColor := m.theme.FgMuted
		if isCursor {
			headColor = m.theme.FG
		}
		row := marker + base.Foreground(headColor).Render(head)
		if isActive {
			row += base.Foreground(m.theme.Success).Render("  · active")
		}
		b.WriteString(row)
		if d < endDisplay-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderPreview returns the cached viewport view sized to the budget
// the layout passed in. Sizing happens here (height changes with the
// popup) but content + scroll state were already prepared in
// refreshPreviewContent so View stays stateless.
func (m stageHistoryPopupModel) renderPreview(width, height int) string {
	if width <= 0 || height <= 0 || len(m.entries) == 0 {
		return ""
	}
	vp := m.preview
	vp.SetWidth(width)
	vp.SetHeight(height)
	return vp.View()
}

// renderStageHistoryPreview applies a glamour pass on raw text so the
// preview matches the rendering used in the stage cards. Falls back to
// the plain text when the renderer can't be built.
func renderStageHistoryPreview(raw string, width int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "(empty entry)"
	}
	if width < 4 {
		width = 4
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(glamourstyles.TokyoNightStyleConfig),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return raw
	}
	out, err := renderer.Render(raw)
	if err != nil {
		return raw
	}
	return out
}

// formatHistoryDuration prints a per-call latency in a compact form
// (Nms / N.Ns / NmNs) so the entry header stays narrow.
func formatHistoryDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// openStageHistoryPopup builds the popup for `id` and assigns it to
// model.popup. Returns a status-bar toast cmd when there is no history
// yet so the caller can short-circuit without showing an empty popup.
func openStageHistoryPopup(model *Model, id stageID) tea.Cmd {
	if int(id) < 0 || int(id) >= len(model.pipeline.stages) {
		return nil
	}
	st := &model.pipeline.stages[id]
	if len(st.History) == 0 {
		return model.WritingStatusBar.ShowMessageForDuration(
			"No history yet — run the pipeline first",
			statusbar.LevelInfo,
			2*time.Second,
		)
	}
	w := max(60, model.width*2/3)
	h := max(20, model.height*3/4)
	if w > model.width-4 {
		w = model.width - 4
	}
	if h > model.height-4 {
		h = model.height - 4
	}
	model.popup = newStageHistoryPopup(
		id, st.Title,
		append([]stageHistoryEntry(nil), st.History...),
		st.ActiveHistoryIndex,
		w, h, model.Theme,
	)
	return nil
}
