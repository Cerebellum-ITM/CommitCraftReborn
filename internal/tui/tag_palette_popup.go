package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/tui/styles"
)

// closeTagPalettePopupMsg dismisses the tag palette popup.
type closeTagPalettePopupMsg struct{}

// tagPalettePopupModel is a read-only viewer that shows every commit-type
// tag together with the four hex values driving its render and a worked
// example of how the chip + commit-message row look with that palette.
type tagPalettePopupModel struct {
	width, height int
	theme         *styles.Theme
	rows          []tagPaletteRow
	body          viewport.Model
}

type tagPaletteRow struct {
	tag     string
	desc    string
	palette styles.CommitTypeColors
}

func newTagPalettePopup(
	width, height int,
	theme *styles.Theme,
	types []commit.CommitType,
) tagPalettePopupModel {
	rows := make([]tagPaletteRow, len(types))
	for i, t := range types {
		rows[i] = tagPaletteRow{
			tag:     t.Tag,
			desc:    t.Description,
			palette: styles.CommitTypePalette(theme, t.Tag),
		}
	}
	vp := viewport.New()
	m := tagPalettePopupModel{
		width:  width,
		height: height,
		theme:  theme,
		rows:   rows,
		body:   vp,
	}
	m.refreshBody()
	return m
}

func (m tagPalettePopupModel) Init() tea.Cmd { return nil }

func (m tagPalettePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc", "q":
			return m, func() tea.Msg { return closeTagPalettePopupMsg{} }
		}
	}
	var cmd tea.Cmd
	m.body, cmd = m.body.Update(msg)
	return m, cmd
}

// Layout constants — single source of truth for both the header row and
// each data row so columns stay aligned. Adjust here, not in the
// per-cell renderers.
const (
	tagPaletteChipColW    = styles.CommitTypeChipInnerWidth + 2 // "[ ADD  ]" -> 10
	tagPaletteHexColW     = 9                                   // "#RRGGBB" + slack
	tagPaletteColGap      = "    "                              // gap between data columns
	tagPalettePreviewGap  = "      "                            // wider gap before preview
	tagPalettePreviewMinW = 30
)

// refreshBody rebuilds the inner table content into the viewport. Called
// once at construction; a window resize regenerates the popup wholesale.
func (m *tagPalettePopupModel) refreshBody() {
	boxFrame := 2*1 + 2*2 // border + horizontal padding (1, 2)
	inner := max(40, m.width-boxFrame)
	headerLines := 5 // title + blank + 2-line header + separator
	hintLines := 2
	bodyH := max(4, m.height-2*1-2*1-headerLines-hintLines)

	m.body.SetWidth(inner)
	m.body.SetHeight(bodyH)
	m.body.SetContent(m.renderTable(inner))
	m.body.GotoTop()
}

func (m tagPalettePopupModel) renderTable(width int) string {
	base := m.theme.AppStyles().Base

	// Compute the preview column width so the row fills `width`.
	colsBeforePreview := tagPaletteChipColW + 4*tagPaletteHexColW +
		4*lipgloss.Width(tagPaletteColGap) + lipgloss.Width(tagPalettePreviewGap)
	previewW := width - colsBeforePreview
	if previewW < tagPalettePreviewMinW {
		previewW = tagPalettePreviewMinW
	}

	var b strings.Builder
	for i, row := range m.rows {
		palette := row.palette

		chip := chipCell(m.theme, row.tag, true)
		emptyChip := base.Width(tagPaletteChipColW).Render(" ")

		bgBlockSwatch, bgBlockHex := swatchAndHex(base, palette.BgBlock, tagPaletteHexColW)
		fgBlockSwatch, fgBlockHex := swatchAndHex(base, palette.FgBlock, tagPaletteHexColW)
		bgMsgSwatch, bgMsgHex := swatchAndHex(base, palette.BgMsg, tagPaletteHexColW)
		fgMsgSwatch, fgMsgHex := swatchAndHex(base, palette.FgMsg, tagPaletteHexColW)

		previewChip := chipCell(m.theme, row.tag, true)
		dummy := dummyMessage(row.tag)
		previewMsg := styles.CommitTypeMsgStyle(m.theme, row.tag).
			Padding(0, 1).
			Render(truncate(dummy, previewW-tagPaletteChipColW-2))
		preview := lipgloss.JoinHorizontal(lipgloss.Top, previewChip, " ", previewMsg)

		// Top half: chip · swatches · preview.
		topLine := lipgloss.JoinHorizontal(lipgloss.Top,
			chip, tagPaletteColGap,
			bgBlockSwatch, tagPaletteColGap,
			fgBlockSwatch, tagPaletteColGap,
			bgMsgSwatch, tagPaletteColGap,
			fgMsgSwatch, tagPalettePreviewGap,
			preview,
		)
		// Bottom half: hex strings under each swatch; chip + preview cells
		// stay blank so the row reads as a single block.
		bottomLine := lipgloss.JoinHorizontal(lipgloss.Top,
			emptyChip, tagPaletteColGap,
			bgBlockHex, tagPaletteColGap,
			fgBlockHex, tagPaletteColGap,
			bgMsgHex, tagPaletteColGap,
			fgMsgHex, tagPalettePreviewGap,
			"",
		)
		b.WriteString(topLine)
		b.WriteString("\n")
		b.WriteString(bottomLine)
		if i < len(m.rows)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func (m tagPalettePopupModel) View() tea.View {
	boxStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	base := m.theme.AppStyles().Base
	title := base.Foreground(m.theme.Secondary).Bold(true).Render("Tag palette")

	header := renderTagPaletteHeader(base, m.theme)
	separator := base.Foreground(m.theme.Subtle).Render(
		strings.Repeat("─", max(20, m.width-boxStyle.GetHorizontalFrameSize())),
	)

	help := m.theme.AppStyles().Help
	hint := strings.Join([]string{
		help.ShortKey.Render("↑↓ pgup/pgdn") + " " + help.ShortDesc.Render("scroll"),
		help.ShortSeparator.Render(" · "),
		help.ShortKey.Render("esc/q") + " " + help.ShortDesc.Render("close"),
	}, "")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		header,
		separator,
		m.body.View(),
		"",
		hint,
	)
	return tea.NewView(boxStyle.Render(body))
}

// renderTagPaletteHeader renders the two-line column header
// (TYPE · BG/BLOCK · FG/BLOCK · BG/MSG · FG/MSG). Stacks the qualifier
// (BLOCK/MSG) under the channel (BG/FG) so the columns stay narrow but
// stay legible — same layout as the screenshot reference.
func renderTagPaletteHeader(base lipgloss.Style, theme *styles.Theme) string {
	style := base.Foreground(theme.Muted).Bold(true)
	headerCell := func(top, bottom string, w int) string {
		t := style.Width(w).Render(top)
		bRow := style.Width(w).Render(bottom)
		return lipgloss.JoinVertical(lipgloss.Left, t, bRow)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		headerCell("TYPE", "", tagPaletteChipColW), tagPaletteColGap,
		headerCell("BG", "BLOCK", tagPaletteHexColW), tagPaletteColGap,
		headerCell("FG", "BLOCK", tagPaletteHexColW), tagPaletteColGap,
		headerCell("BG", "MSG", tagPaletteHexColW), tagPaletteColGap,
		headerCell("FG", "MSG", tagPaletteHexColW),
	)
}

// chipCell renders a commit-type chip sized to the standard chip column.
// Bold + block palette so the row's tag identity reads at a glance.
func chipCell(theme *styles.Theme, tag string, bold bool) string {
	style := styles.CommitTypeBlockStyle(theme, tag).
		Width(styles.CommitTypeChipInnerWidth).
		Padding(0, 1).
		Align(lipgloss.Center)
	if bold {
		style = style.Bold(true)
	}
	return style.Render(truncate(strings.ToUpper(tag), styles.CommitTypeChipInnerWidth))
}

// swatchAndHex returns (swatch, hex) where swatch is a small filled
// block painted with `c` and hex is the "#RRGGBB" string in muted fg.
// When `c` is nil (theme-fallback slot) both cells render as a dim "—"
// so the column still aligns and the empty state is obvious.
func swatchAndHex(base lipgloss.Style, c color.Color, width int) (string, string) {
	hex := colorToHex(c)
	if hex == "" {
		dash := base.Width(width).Render("—")
		return dash, dash
	}
	swatch := lipgloss.NewStyle().
		Background(c).
		Width(width).
		Render(" ")
	hexLabel := base.Width(width).Render(hex)
	return swatch, hexLabel
}

// colorToHex returns "#RRGGBB" for any color.Color, going through RGBA
// so it works for lipgloss.RGBColor, ANSI-indexed colors, and theme
// custom types alike. Returns an empty string when the input is nil so
// callers can render a placeholder.
func colorToHex(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

// dummyMessage returns a sample commit subject tailored to each tag so
// the preview reads like a real history row instead of lorem ipsum.
func dummyMessage(tag string) string {
	switch strings.ToUpper(tag) {
	case "FIX":
		return "resolve nil deref in commit pipeline"
	case "ADD":
		return "add palette overrides via TOML config"
	case "DOC", "DOCS":
		return "document the four-color tag scheme"
	case "WIP":
		return "wip on async stage history rewrite"
	case "STYLE":
		return "tighten compose spacing and pill widths"
	case "REFACTOR", "REF", "IMP":
		return "extract popup wiring into update.go"
	case "TEST":
		return "cover hex parsing edge cases"
	case "PERF":
		return "cache rate-limit lookups across renders"
	case "CHORE":
		return "bump dependencies and prune unused imports"
	case "DEL", "REM":
		return "drop dead commitTypeColor plumbing"
	case "BUILD", "REL":
		return "wire release build target into Makefile"
	case "CI":
		return "fail the workflow when go vet emits anything"
	case "REVERT":
		return "revert speculative pipeline reorder"
	case "SEC":
		return "redact tokens before persisting to disk"
	case "MOV":
		return "move popup helpers into tui/popups"
	case "UI":
		return "round corners on the version editor"
	default:
		return "example commit message for " + strings.ToLower(tag)
	}
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 1 {
		return "…"
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}
