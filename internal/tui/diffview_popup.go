package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	chroma "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/charmtone"

	"commit_craft_reborn/internal/git"
	tuistyles "commit_craft_reborn/internal/tui/styles"
)

// -------------------------------------------------------------------------
// Crush base diff style — exact replica of crush's DefaultStyles() s.Diff block
// (internal/ui/styles/styles.go), variables mapped:
//   fgHalfMuted = charmtone.Smoke  (#BFBCC8)
//   bgBaseLighter = charmtone.BBQ  (#2d2c35)
//   bgBase        = charmtone.Pepper (#201F26)
//   fgMuted       = charmtone.Squid  (#858392)

type diffLineStyle struct {
	LineNumber lipgloss.Style
	Symbol     lipgloss.Style
	Code       lipgloss.Style
	// bgHex is used to drive chroma syntax highlighting background.
	bgHex string
}

type diffStyle struct {
	DividerLine diffLineStyle
	MissingLine diffLineStyle
	EqualLine   diffLineStyle
	InsertLine  diffLineStyle
	DeleteLine  diffLineStyle
	Filename    diffLineStyle
}

func crushBaseStyle() diffStyle {
	return diffStyle{
		DividerLine: diffLineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Smoke).
				Background(charmtone.BBQ),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Smoke).
				Background(charmtone.BBQ),
			bgHex: "#2d2c35",
		},
		MissingLine: diffLineStyle{
			LineNumber: lipgloss.NewStyle().
				Background(charmtone.BBQ),
			Code: lipgloss.NewStyle().
				Background(charmtone.BBQ),
			bgHex: "#2d2c35",
		},
		EqualLine: diffLineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Squid).
				Background(lipgloss.Color("#2d2c35")),
			Code: lipgloss.NewStyle().
				Background(lipgloss.Color("#2d2c35")),
			bgHex: "#2d2c35",
		},
		InsertLine: diffLineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#629657")).
				Background(lipgloss.Color("#2b322a")),
			Symbol: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#629657")).
				Background(lipgloss.Color("#323931")),
			Code: lipgloss.NewStyle().
				Background(lipgloss.Color("#323931")),
			bgHex: "#323931",
		},
		DeleteLine: diffLineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a45c59")).
				Background(lipgloss.Color("#312929")),
			Symbol: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#a45c59")).
				Background(lipgloss.Color("#383030")),
			Code: lipgloss.NewStyle().
				Background(lipgloss.Color("#383030")),
			bgHex: "#383030",
		},
		Filename: diffLineStyle{
			LineNumber: lipgloss.NewStyle().
				Foreground(charmtone.Smoke).
				Background(charmtone.BBQ),
			Code: lipgloss.NewStyle().
				Foreground(charmtone.Smoke).
				Background(charmtone.BBQ),
			bgHex: "#2d2c35",
		},
	}
}

// -------------------------------------------------------------------------
// Syntax highlighting helpers

var (
	// Strip all ANSI background codes chroma emits so we can apply our own.
	ansiBgRe = regexp.MustCompile(`\x1b\[4[89](;[0-9;]*)?m|\x1b\[48[;:][0-9;:]*m`)
	// Replace full resets \033[0m with foreground-only resets so our bg persists.
	ansiResetRe = regexp.MustCompile(`\x1b\[0m`)
)

// highlightCode applies chroma monokai foreground syntax colors to code and
// renders it on bgHex background, padded/truncated to codeWidth visible chars.
func highlightCode(code string, lexer chroma.Lexer, codeWidth int, bgHex string) string {
	if lexer == nil {
		return plainCodeBg(code, bgHex, codeWidth)
	}

	// Measure and truncate BEFORE adding any ANSI — plain text width is reliable.
	plainWidth := ansi.StringWidth(code)
	if plainWidth > codeWidth {
		code = ansi.Truncate(code, codeWidth, "")
		plainWidth = codeWidth
	}

	monoStyle := chromastyles.Get("monokai")
	if monoStyle == nil {
		monoStyle = chromastyles.Fallback
	}

	tokens, err := lexer.Tokenise(nil, code)
	if err != nil {
		return plainCodeBg(code, bgHex, codeWidth)
	}

	var buf strings.Builder
	if err := formatters.Get("terminal16m").Format(&buf, monoStyle, tokens); err != nil {
		return plainCodeBg(code, bgHex, codeWidth)
	}

	result := strings.TrimRight(buf.String(), "\n")

	// 1. Strip any background codes chroma added (we control the background).
	result = ansiBgRe.ReplaceAllString(result, "")

	// 2. Replace full resets with foreground-only resets so our bg stays active.
	result = ansiResetRe.ReplaceAllString(result, "\x1b[39;22;23m")

	// 3. Prepend our crush background.
	r, g, b := hexToRGB(bgHex)
	result = fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b) + result

	// 4. Pad to codeWidth using the reliable plain-text measurement.
	padding := codeWidth - plainWidth
	if padding > 0 {
		result += strings.Repeat(" ", padding)
	}

	// 5. Full reset at the end.
	return result + "\x1b[0m"
}

// plainCodeBg renders code as plain text on bgHex background, padded to width.
func plainCodeBg(code, bgHex string, width int) string {
	r, g, b := hexToRGB(bgHex)
	runes := []rune(code)
	if len(runes) > width {
		runes = []rune(string(runes[:width-1]) + "…")
	}
	padding := width - len(runes)
	if padding < 0 {
		padding = 0
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm%s%s\x1b[0m",
		r, g, b, string(runes), strings.Repeat(" ", padding))
}

func hexToRGB(hex string) (r, g, b uint8) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0
	}
	rv, _ := strconv.ParseUint(hex[0:2], 16, 8)
	gv, _ := strconv.ParseUint(hex[2:4], 16, 8)
	bv, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return uint8(rv), uint8(gv), uint8(bv)
}

// -------------------------------------------------------------------------
// Popup model

type diffViewPopup struct {
	filePath string
	viewport viewport.Model
	theme    *tuistyles.Theme
	width    int
	height   int
}

type closeDiffViewPopupMsg struct{}

type diffFetchedMsg struct {
	filePath string
	content  string
	err      error
}

func newDiffViewPopup(
	filePath, diffText string,
	width, height int,
	theme *tuistyles.Theme,
) diffViewPopup {
	popupW := width * 85 / 100
	popupH := height * 80 / 100

	vpW := popupW - 4
	vpH := popupH - 4

	vp := viewport.New()
	vp.SetWidth(vpW)
	vp.SetHeight(vpH)
	vp.SetContent(renderDiff(diffText, vpW, filePath))

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
	footerPct := crushBaseStyle().DividerLine.LineNumber.Render(scrollPct)
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

// -------------------------------------------------------------------------
// Diff renderer — crush-style with chroma syntax highlighting

var hunkRe = regexp.MustCompile(`@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func renderDiff(raw string, width int, filePath string) string {
	st := crushBaseStyle()

	// Detect chroma lexer from file extension.
	var lexer chroma.Lexer
	if filePath != "" {
		lexer = lexers.Match(filepath.Base(filePath))
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	if raw == "" {
		return st.EqualLine.Code.Render("  (no staged changes for this file)")
	}

	// First pass: detect max line number for column width.
	maxLineNum := 1
	for _, line := range strings.Split(raw, "\n") {
		if m := hunkRe.FindStringSubmatch(line); m != nil {
			if n, err := strconv.Atoi(m[2]); err == nil && n+200 > maxLineNum {
				maxLineNum = n + 200
			}
		}
	}
	numWidth := len(strconv.Itoa(maxLineNum))
	numFmt := fmt.Sprintf("%%%dd", numWidth)

	// col widths: lineNum(numWidth) + symbol(1) + code — no separators between columns
	codeWidth := width - numWidth - 1
	if codeWidth < 10 {
		codeWidth = 10
	}

	renderNum := func(ls diffLineStyle, n int) string {
		return ls.LineNumber.Width(numWidth).Render(fmt.Sprintf(numFmt, n))
	}
	blankNum := func(ls diffLineStyle) string {
		return ls.LineNumber.Width(numWidth).Render(strings.Repeat(" ", numWidth))
	}

	var (
		sb         strings.Builder
		beforeLine int
		afterLine  int
	)

	for _, line := range strings.Split(raw, "\n") {
		switch {
		// ── git metadata & filename headers ──────────────────────────────
		case strings.HasPrefix(line, "diff ") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "deleted file") ||
			strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++"):
			sb.WriteString(blankNum(st.Filename))
			sb.WriteString(st.Filename.Symbol.Render(" "))
			sb.WriteString(st.Filename.Code.Width(codeWidth).Render(truncateLine(line, codeWidth)))
			sb.WriteString("\n")

		// ── hunk divider ─────────────────────────────────────────────────
		case strings.HasPrefix(line, "@@"):
			m := hunkRe.FindStringSubmatch(line)
			if m != nil {
				before, _ := strconv.Atoi(m[1])
				after, _ := strconv.Atoi(m[2])
				beforeLine = before
				afterLine = after
			}
			sb.WriteString(blankNum(st.DividerLine))
			sb.WriteString(st.DividerLine.LineNumber.Render(" "))
			sb.WriteString(
				st.DividerLine.Code.Width(codeWidth).Render(truncateLine(line, codeWidth)),
			)
			sb.WriteString("\n")

		// ── inserted line ─────────────────────────────────────────────────
		case strings.HasPrefix(line, "+"):
			code := line[1:]
			sb.WriteString(renderNum(st.InsertLine, afterLine))
			afterLine++
			sb.WriteString(st.InsertLine.Symbol.Render("+"))
			sb.WriteString(highlightCode(code, lexer, codeWidth, st.InsertLine.bgHex))
			sb.WriteString("\n")

		// ── deleted line ──────────────────────────────────────────────────
		case strings.HasPrefix(line, "-"):
			code := line[1:]
			sb.WriteString(renderNum(st.DeleteLine, beforeLine))
			beforeLine++
			sb.WriteString(st.DeleteLine.Symbol.Render("-"))
			sb.WriteString(highlightCode(code, lexer, codeWidth, st.DeleteLine.bgHex))
			sb.WriteString("\n")

		// ── context line ──────────────────────────────────────────────────
		case strings.HasPrefix(line, " "):
			code := line[1:]
			sb.WriteString(renderNum(st.EqualLine, afterLine))
			beforeLine++
			afterLine++
			sb.WriteString(st.EqualLine.LineNumber.Render(" "))
			sb.WriteString(highlightCode(code, lexer, codeWidth, st.EqualLine.bgHex))
			sb.WriteString("\n")

		default:
			if line != "" {
				sb.WriteString(st.EqualLine.Code.Render(truncateLine(line, width)))
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
	if maxW <= 1 {
		return string(runes[:maxW])
	}
	return string(runes[:maxW-1]) + "…"
}

func fetchDiffCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		content, err := git.GetStagedFileDiff(filePath)
		return diffFetchedMsg{filePath: filePath, content: content, err: err}
	}
}
