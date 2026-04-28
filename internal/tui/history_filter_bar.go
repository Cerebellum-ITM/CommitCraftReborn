package tui

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// HistoryFilterBar wraps a textinput so the History view can render a
// dedicated filter row with the layout
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
	ti.Placeholder = "message · id · type · scope…"
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

func (f HistoryFilterBar) View() string {
	borderColor := f.theme.Subtle
	prefixColor := f.theme.Muted
	if f.focused {
		borderColor = f.theme.Primary
		prefixColor = f.theme.Primary
	}

	prefix := lipgloss.NewStyle().Foreground(prefixColor).Render("› filter ")
	counterText := fmt.Sprintf("%d / %d", f.visible, f.total)
	counter := lipgloss.NewStyle().Foreground(f.theme.Muted).Render(counterText)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	// Inner width = total width - border frame - padding - prefix - counter -
	// a single-space gap on each side of the input.
	innerWidth := f.width - border.GetHorizontalFrameSize() - border.GetHorizontalPadding() -
		lipgloss.Width(prefix) - lipgloss.Width(counter) - 2
	if innerWidth < 8 {
		innerWidth = 8
	}
	f.input.SetWidth(innerWidth)

	// Right-align the counter by padding between the input and the counter.
	inputView := f.input.View()
	gap := f.width - border.GetHorizontalFrameSize() - border.GetHorizontalPadding() -
		lipgloss.Width(prefix) - lipgloss.Width(inputView) - lipgloss.Width(counter)
	if gap < 1 {
		gap = 1
	}

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		prefix,
		inputView,
		lipgloss.NewStyle().Width(gap).Render(""),
		counter,
	)
	return border.Width(f.width - border.GetHorizontalFrameSize()).Render(row)
}
