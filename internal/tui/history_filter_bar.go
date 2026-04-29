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

// HistoryFilterBar wraps a textinput so the History view can render a single
// content row inside the surrounding frame:
//
//	› filter [_____________]                         5 / 8
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
	ti.Placeholder = "message - id - type - scope..."
	ti.SetVirtualCursor(true)
	return HistoryFilterBar{
		input: ti,
		theme: theme,
	}
}

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
	prefixColor := f.theme.Muted
	if f.focused {
		prefixColor = f.theme.Primary
	}

	prefix := lipgloss.NewStyle().Foreground(prefixColor).Render("> filter ")
	counterText := fmt.Sprintf("%d / %d", f.visible, f.total)
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
