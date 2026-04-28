package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
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
		cursor = 0
	}
	return stageHistoryPopupModel{
		stage:       stage,
		stageLabel:  stageLabel,
		entries:     entries,
		activeIndex: activeIndex,
		cursor:      cursor,
		width:       width,
		height:      height,
		theme:       theme,
	}
}

func (m stageHistoryPopupModel) Init() tea.Cmd { return nil }

func (m stageHistoryPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "esc", "q":
		return m, func() tea.Msg { return closeStageHistoryMsg{} }
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
		return m, nil
	case "home", "g":
		m.cursor = 0
		return m, nil
	case "end", "G":
		if len(m.entries) > 0 {
			m.cursor = len(m.entries) - 1
		}
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
	return m, nil
}

func (m stageHistoryPopupModel) View() tea.View {
	box := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	innerW := max(40, m.width-box.GetHorizontalFrameSize())
	innerH := max(8, m.height-box.GetVerticalFrameSize())

	base := m.theme.AppStyles().Base
	title := base.Foreground(m.theme.Secondary).Bold(true).Render(
		fmt.Sprintf("History · %s", m.stageLabel),
	)
	count := base.Foreground(m.theme.Muted).Render(
		fmt.Sprintf("· %d version%s", len(m.entries), pluralS(len(m.entries))),
	)
	header := lipgloss.JoinHorizontal(lipgloss.Top, title, " ", count)

	hint := base.Foreground(m.theme.Muted).Render(
		"↑↓/jk navigate · ↵ apply · esc cancel",
	)

	rows := m.renderRows(innerW, innerH-lipgloss.Height(header)-lipgloss.Height(hint)-2)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		rows,
		"",
		hint,
	)
	return tea.NewView(box.Render(body))
}

// renderRows builds the entry list, two lines per entry (header + preview).
func (m stageHistoryPopupModel) renderRows(width, maxLines int) string {
	if len(m.entries) == 0 {
		return m.theme.AppStyles().Base.Foreground(m.theme.Muted).Italic(true).
			Render("No history yet.")
	}
	base := m.theme.AppStyles().Base
	var b strings.Builder
	lines := 0
	// Iterate newest-first so the latest version sits at the top.
	for displayIdx := len(m.entries) - 1; displayIdx >= 0; displayIdx-- {
		if lines+2 > maxLines {
			break
		}
		entry := m.entries[displayIdx]
		isCursor := displayIdx == m.cursor
		isActive := displayIdx == m.activeIndex

		marker := "  "
		if isCursor {
			marker = base.Foreground(m.theme.Warning).Bold(true).Render("▸ ")
		}
		activeBadge := ""
		if isActive {
			activeBadge = base.Foreground(m.theme.Success).Render(" · active")
		}
		ts := entry.CapturedAt.Local().Format("15:04:05")
		stats := fmt.Sprintf(
			"[%d] %s · %s tok (in %s · out %s) · %s",
			displayIdx+1,
			ts,
			formatQuantity(entry.TotalTokens),
			formatQuantity(entry.PromptTokens),
			formatQuantity(entry.CompletionTokens),
			formatHistoryDuration(entry.APITotalTime),
		)
		statsColor := m.theme.FgMuted
		if isCursor {
			statsColor = m.theme.FG
		}
		statsStyled := base.Foreground(statsColor).Render(stats)

		preview := previewLine(entry.Text, width-4)
		previewStyled := base.Foreground(m.theme.Subtle).Italic(true).Render("    " + preview)

		b.WriteString(marker)
		b.WriteString(statsStyled)
		b.WriteString(activeBadge)
		b.WriteString("\n")
		b.WriteString(previewStyled)
		b.WriteString("\n")
		lines += 2
	}
	return strings.TrimRight(b.String(), "\n")
}

// previewLine returns the first non-empty line of text, truncated to
// fit width. Falls back to the empty string when text is blank.
func previewLine(text string, width int) string {
	if width < 8 {
		width = 8
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > width {
			trimmed = trimmed[:width-1] + "…"
		}
		return trimmed
	}
	return ""
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
	h := max(14, model.height*2/3)
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
