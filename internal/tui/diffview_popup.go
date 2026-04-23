package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"commit_craft_reborn/internal/tui/styles"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type diffViewPopup struct {
	filePath string
	viewport viewport.Model
	theme    *styles.Theme
	width    int
	height   int
}

type closeDiffViewPopupMsg struct{}

type diffFetchedMsg struct {
	filePath string
	content  string
	err      error
}

func newDiffViewPopup(filePath, diffText string, width, height int, theme *styles.Theme) diffViewPopup {
	popupW := width * 85 / 100
	popupH := height * 80 / 100

	// viewport sits inside the popup border (2 sides) + header/footer lines
	vpW := popupW - 4
	vpH := popupH - 5

	vp := viewport.New()
	vp.SetWidth(vpW)
	vp.SetHeight(vpH)
	vp.SetContent(renderDiff(diffText, vpW, theme))

	return diffViewPopup{
		filePath: filePath,
		viewport: vp,
		theme:    theme,
		width:    popupW,
		height:   popupH,
	}
}

func (p diffViewPopup) Init() tea.Cmd { return nil }

func (p diffViewPopup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return p, func() tea.Msg { return closeDiffViewPopupMsg{} }
		}
	}
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p diffViewPopup) View() tea.View {
	theme := p.theme
	dimColor := lipgloss.Color("240")

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Accent).
		Bold(true).
		Padding(0, 1)

	titleRendered := titleStyle.Render(filepath.Base(p.filePath))
	headerSep := lipgloss.NewStyle().
		Foreground(dimColor).
		Render(strings.Repeat("─", max(0, p.width-lipgloss.Width(titleRendered)-4)))

	header := lipgloss.JoinHorizontal(lipgloss.Left, titleRendered, headerSep)

	scrollPct := fmt.Sprintf(" %3.f%% ", p.viewport.ScrollPercent()*100)
	footerHint := lipgloss.NewStyle().Foreground(dimColor).Render("  q/esc close  ↑↓ scroll")
	footerPct := lipgloss.NewStyle().Foreground(theme.Accent).Render(scrollPct)
	footerSep := lipgloss.NewStyle().
		Foreground(dimColor).
		Render(strings.Repeat("─", max(0, p.width-lipgloss.Width(footerHint)-lipgloss.Width(footerPct)-4)))

	footer := lipgloss.JoinHorizontal(lipgloss.Left, footerSep, footerHint, footerPct)

	inner := lipgloss.JoinVertical(lipgloss.Left,
		header,
		p.viewport.View(),
		footer,
	)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(0, 1)

	return tea.NewView(boxStyle.Render(inner))
}

// hunkHeader parses "@@ -a,b +c,d @@" and returns beforeStart and afterStart.
var hunkRe = regexp.MustCompile(`@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func renderDiff(raw string, width int, theme *styles.Theme) string {
	if raw == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  (no staged changes for this file)")
	}

	// Colors — Tokyo Night palette
	addedBg := lipgloss.Color("#293229")
	deletedBg := lipgloss.Color("#332929")
	addedFg := lipgloss.Color("#9ece6a")
	deletedFg := lipgloss.Color("#f7768e")
	hunkFg := lipgloss.Color("#7dcfff")
	headerFg := lipgloss.Color("#565f89")
	lineNumFg := lipgloss.Color("#3b4261")
	sepFg := lipgloss.Color("#3b4261")

	lineNumStyle := lipgloss.NewStyle().Foreground(lineNumFg)
	sepStyle := lipgloss.NewStyle().Foreground(sepFg)
	addedStyle := lipgloss.NewStyle().Foreground(addedFg).Background(addedBg)
	deletedStyle := lipgloss.NewStyle().Foreground(deletedFg).Background(deletedBg)
	hunkStyle := lipgloss.NewStyle().Foreground(hunkFg)
	headerStyle := lipgloss.NewStyle().Foreground(headerFg)
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	sep := sepStyle.Render(" │ ")

	// Line number widths — detect from hunk headers first pass
	maxLineNum := 1
	for _, line := range strings.Split(raw, "\n") {
		if m := hunkRe.FindStringSubmatch(line); m != nil {
			if n, err := strconv.Atoi(m[2]); err == nil && n > maxLineNum {
				maxLineNum = n + 200
			}
		}
	}
	numWidth := len(strconv.Itoa(maxLineNum))
	numFmt := fmt.Sprintf("%%%dd", numWidth)

	var (
		sb          strings.Builder
		beforeLine  int
		afterLine   int
	)

	// code content width after line number + sep + sigil + sep
	// numWidth + 3 (sep) + 1 (sigil) + 3 (sep)
	codeWidth := width - numWidth - 7
	if codeWidth < 10 {
		codeWidth = 10
	}

	for _, line := range strings.Split(raw, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"):
			m := hunkRe.FindStringSubmatch(line)
			if m != nil {
				before, _ := strconv.Atoi(m[1])
				after, _ := strconv.Atoi(m[2])
				beforeLine = before
				afterLine = after
			}
			// Render hunk header without line number
			placeholder := strings.Repeat(" ", numWidth)
			rendered := truncateLine(line, codeWidth)
			sb.WriteString(lineNumStyle.Render(placeholder))
			sb.WriteString(sep)
			sb.WriteString(hunkStyle.Render("  "))
			sb.WriteString(sep)
			sb.WriteString(hunkStyle.Render(rendered))
			sb.WriteString("\n")

		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++"):
			placeholder := strings.Repeat(" ", numWidth)
			rendered := truncateLine(line, codeWidth)
			sb.WriteString(lineNumStyle.Render(placeholder))
			sb.WriteString(sep)
			sb.WriteString(headerStyle.Render("  "))
			sb.WriteString(sep)
			sb.WriteString(headerStyle.Render(rendered))
			sb.WriteString("\n")

		case strings.HasPrefix(line, "diff ") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "deleted file"):
			placeholder := strings.Repeat(" ", numWidth)
			rendered := truncateLine(line, codeWidth)
			sb.WriteString(lineNumStyle.Render(placeholder))
			sb.WriteString(sep)
			sb.WriteString(headerStyle.Render("  "))
			sb.WriteString(sep)
			sb.WriteString(headerStyle.Render(rendered))
			sb.WriteString("\n")

		case strings.HasPrefix(line, "+"):
			numStr := fmt.Sprintf(numFmt, afterLine)
			afterLine++
			code := truncateLine(line[1:], codeWidth)
			sb.WriteString(lineNumStyle.Render(numStr))
			sb.WriteString(sep)
			sb.WriteString(addedStyle.Render("+"))
			sb.WriteString(sep)
			sb.WriteString(addedStyle.Render(padRight(code, codeWidth)))
			sb.WriteString("\n")

		case strings.HasPrefix(line, "-"):
			numStr := fmt.Sprintf(numFmt, beforeLine)
			beforeLine++
			code := truncateLine(line[1:], codeWidth)
			sb.WriteString(lineNumStyle.Render(numStr))
			sb.WriteString(sep)
			sb.WriteString(deletedStyle.Render("-"))
			sb.WriteString(sep)
			sb.WriteString(deletedStyle.Render(padRight(code, codeWidth)))
			sb.WriteString("\n")

		case strings.HasPrefix(line, " "):
			numStr := fmt.Sprintf(numFmt, afterLine)
			beforeLine++
			afterLine++
			code := truncateLine(line[1:], codeWidth)
			sb.WriteString(lineNumStyle.Render(numStr))
			sb.WriteString(sep)
			sb.WriteString(contextStyle.Render(" "))
			sb.WriteString(sep)
			sb.WriteString(contextStyle.Render(code))
			sb.WriteString("\n")

		default:
			if line != "" {
				sb.WriteString(headerStyle.Render(truncateLine(line, width)))
				sb.WriteString("\n")
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

func truncateLine(s string, maxW int) string {
	runes := []rune(s)
	if len(runes) <= maxW {
		return s
	}
	if maxW <= 3 {
		return string(runes[:maxW])
	}
	return string(runes[:maxW-1]) + "…"
}

func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func fetchDiffCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		content, err := GetStagedFileDiff(filePath)
		return diffFetchedMsg{filePath: filePath, content: content, err: err}
	}
}
