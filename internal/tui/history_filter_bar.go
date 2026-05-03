package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"commit_craft_reborn/internal/tui/styles"
)

// MainFilterMode picks which commit field the workspace list filter
// matches against. Cycled via ctrl+f from the filter bar.
type MainFilterMode int

const (
	FilterModeTitle MainFilterMode = iota
	FilterModeID
	FilterModeType
	FilterModeScope
)

// mainFilterModeOrder is the cycle order used by CycleMode and the
// definitive list of supported modes (used for the modulo bound).
var mainFilterModeOrder = []MainFilterMode{
	FilterModeTitle,
	FilterModeID,
	FilterModeType,
	FilterModeScope,
}

// mainFilterModeMeta stores the visible label and the commit-type
// palette tag whose `msg` (dim) colors paint the mode pill. Each mode
// gets a visually distinct palette so the active mode reads at a
// glance.
var mainFilterModeMeta = map[MainFilterMode]struct {
	label string
	tag   string
}{
	FilterModeTitle: {"TITLE", "ADD"},  // greenish
	FilterModeID:    {"ID", "WIP"},     // amber
	FilterModeType:  {"TYPE", "STYLE"}, // purple
	FilterModeScope: {"SCOPE", "SEC"},  // pink/red
}

// currentMainFilterMode is read by HistoryCommitItem.FilterValue so
// switching the mode applies live to the list's filter pass without
// having to rebuild items. Mutated only via HistoryFilterBar.CycleMode.
// Explicitly initialised to FilterModeTitle so the default startup mode
// is unambiguous (Go's zero value would also resolve to TITLE today,
// but the explicit assignment guards against future iota reorders).
var currentMainFilterMode = FilterModeTitle

// CurrentMainFilterMode exposes the current mode to package callers
// (notably the FilterValue implementation in main_list.go).
func CurrentMainFilterMode() MainFilterMode { return currentMainFilterMode }

// HistoryFilterBar wraps a textinput so the History view can render a single
// content row inside the surrounding frame:
//
//	[TITLE]  > [_____________]                       5 / 8
//
// The bar manages its own focus and value; the parent decides when to feed
// keys into Update via Focus/Blur. The right-hand "n / total" counter is
// supplied at render time via View.
type HistoryFilterBar struct {
	input   textinput.Model
	theme   *styles.Theme
	focused bool
	width   int
	visible int
	total   int
}

func NewHistoryFilterBar(theme *styles.Theme) HistoryFilterBar {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "type to filter…"
	ti.SetVirtualCursor(true)
	return HistoryFilterBar{
		input: ti,
		theme: theme,
	}
}

// CycleMode advances the active filter mode to the next one in the
// canonical order. Wraps around at the end so cycling is endless.
func (f *HistoryFilterBar) CycleMode() {
	currentMainFilterMode = mainFilterModeOrder[(int(currentMainFilterMode)+1)%len(mainFilterModeOrder)]
}

// Mode returns the active filter mode.
func (f HistoryFilterBar) Mode() MainFilterMode { return currentMainFilterMode }

func (f *HistoryFilterBar) SetSize(width int) { f.width = width }
func (f *HistoryFilterBar) SetCounts(visible, total int) {
	f.visible = visible
	f.total = total
}
func (f HistoryFilterBar) Value() string   { return f.input.Value() }
func (f HistoryFilterBar) IsFocused() bool { return f.focused }

func (f *HistoryFilterBar) Focus() tea.Cmd {
	f.focused = true
	return f.input.Focus()
}

func (f *HistoryFilterBar) Blur() {
	f.focused = false
	f.input.Blur()
}

func (f *HistoryFilterBar) Reset() {
	f.input.SetValue("")
}

// Update only consumes keys when the bar has focus. The parent should still
// call Update for non-focused frames so the cursor blink ticks reach the
// embedded textinput.
func (f HistoryFilterBar) Update(msg tea.Msg) (HistoryFilterBar, tea.Cmd) {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}

// View renders a single content row of width exactly f.width. The
// surrounding HistoryView owns the outer frame. The composition is built
// manually (instead of via Style.Width / lipgloss.Place) so we can hard-cap
// the input view's width with ansi.Truncate when the placeholder/cursor
// renders wider than the budget — `lipgloss.Place` is a no-op once
// content already exceeds the requested width, which is exactly the
// failure mode that pushes "214" onto a second line.
func (f HistoryFilterBar) View() string {
	meta, ok := mainFilterModeMeta[currentMainFilterMode]
	if !ok {
		// Defensive: an unmapped mode would render as an empty pill,
		// which is what looks like a "missing initial value" in the UI.
		// Fall back to TITLE so the bar is always labelled.
		meta = mainFilterModeMeta[FilterModeTitle]
	}
	modePill := styles.CommitTypeMsgStyle(f.theme, meta.tag).
		Bold(true).
		Padding(0, 1).
		Render(meta.label)

	arrowColor := f.theme.Muted
	if f.focused {
		arrowColor = f.theme.Primary
	}
	arrow := lipgloss.NewStyle().Foreground(arrowColor).Render(" > ")

	prefix := modePill + arrow
	// Right-side commits counter: glyph + visible/total + label. Replaces
	// the bottom statusbar's "X commits" item so the count lives in one
	// place at the top of the History view.
	noun := "commits"
	if f.total == 1 {
		noun = "commit"
	}
	counterText := fmt.Sprintf(
		"%s %d/%d %s",
		f.theme.AppSymbols().GitCommit,
		f.visible,
		f.total,
		noun,
	)
	counter := lipgloss.NewStyle().Foreground(f.theme.Muted).Render(counterText)

	prefixW := lipgloss.Width(prefix)
	counterW := lipgloss.Width(counter)

	// Reserve 4 extra cells of slack so width-counting drift on glyphs
	// like `›`, `·` or `…` (which some terminals render at 2 cells while
	// ansi.StringWidth counts as 1) never pushes the final row over
	// f.width. The trailing hard-cap below catches anything that still
	// slips through.
	inputBudget := f.width - prefixW - counterW - 4
	if inputBudget < 8 {
		inputBudget = 8
	}
	f.input.SetWidth(inputBudget)
	inputView := ansi.Truncate(f.input.View(), inputBudget, "")
	inputW := lipgloss.Width(inputView)

	gap := f.width - prefixW - inputW - counterW
	if gap < 1 {
		gap = 1
	}
	row := prefix + inputView + strings.Repeat(" ", gap) + counter

	// Hard-cap to exactly f.width visible cells. Belt-and-suspenders: if
	// any width measurement above was off, this guarantees the row never
	// pushes content past the outer frame's right border.
	rowW := lipgloss.Width(row)
	if rowW > f.width {
		row = ansi.Truncate(row, f.width, "")
	} else if rowW < f.width {
		row = row + strings.Repeat(" ", f.width-rowW)
	}
	return row
}
