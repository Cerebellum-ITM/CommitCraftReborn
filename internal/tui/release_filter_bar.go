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

// ReleaseFilterMode picks which release field the workspace list filter
// matches against. Cycled via ctrl+f from the filter bar — same pattern
// the workspace history uses for commits.
type ReleaseFilterMode int

const (
	ReleaseFilterModeTitle ReleaseFilterMode = iota
	ReleaseFilterModeType
	ReleaseFilterModeVersion
	ReleaseFilterModeBranch
)

var releaseFilterModeOrder = []ReleaseFilterMode{
	ReleaseFilterModeTitle,
	ReleaseFilterModeType,
	ReleaseFilterModeVersion,
	ReleaseFilterModeBranch,
}

// releaseFilterModeMeta maps each mode to its pill label and the
// commit-type palette tag whose colors paint the pill — picked so the
// active mode stands apart from neighboring modes at a glance.
var releaseFilterModeMeta = map[ReleaseFilterMode]struct {
	label string
	tag   string
}{
	ReleaseFilterModeTitle:   {"TITLE", "ADD"},     // green
	ReleaseFilterModeType:    {"TYPE", "STYLE"},    // purple
	ReleaseFilterModeVersion: {"VERSION", "FIX"},   // amber
	ReleaseFilterModeBranch:  {"BRANCH", "REVERT"}, // pink/red
}

// currentReleaseFilterMode mirrors `currentMainFilterMode` for releases.
// HistoryReleaseItem.FilterValue reads it so cycling re-evaluates the
// list filter without rebuilding items.
var currentReleaseFilterMode = ReleaseFilterModeTitle

// CurrentReleaseFilterMode exposes the active mode to FilterValue.
func CurrentReleaseFilterMode() ReleaseFilterMode { return currentReleaseFilterMode }

// ReleasePickerFilterMode is the mode set for the workspace commit
// picker (Compose tab in Release Mode). The picker filters commits, so
// the meaningful axes (title / hash / type / tag) differ from the
// release-history axes (title / type / version / branch). Kept in a
// dedicated package var so cycling on one screen never bleeds into the
// other — same separation `currentMainFilterMode` enforces for the
// commit-mode workspace history.
type ReleasePickerFilterMode int

const (
	ReleasePickerFilterModeTitle ReleasePickerFilterMode = iota
	ReleasePickerFilterModeHash
	ReleasePickerFilterModeType
	ReleasePickerFilterModeTag
)

var releasePickerFilterModeOrder = []ReleasePickerFilterMode{
	ReleasePickerFilterModeTitle,
	ReleasePickerFilterModeHash,
	ReleasePickerFilterModeType,
	ReleasePickerFilterModeTag,
}

var releasePickerFilterModeMeta = map[ReleasePickerFilterMode]struct {
	label string
	tag   string
}{
	ReleasePickerFilterModeTitle: {"TITLE", "ADD"},  // green
	ReleasePickerFilterModeHash:  {"HASH", "STYLE"}, // purple
	ReleasePickerFilterModeType:  {"TYPE", "FIX"},   // amber
	ReleasePickerFilterModeTag:   {"TAG", "REVERT"}, // pink/red
}

var currentReleasePickerFilterMode = ReleasePickerFilterModeTitle

// CurrentReleasePickerFilterMode exposes the picker's active mode to
// WorkspaceCommitItem.FilterValue.
func CurrentReleasePickerFilterMode() ReleasePickerFilterMode {
	return currentReleasePickerFilterMode
}

// ReleaseFilterBarKind selects which mode set/state the bar reads from.
// History bars cycle the release-history modes; picker bars cycle the
// commit-picker modes. Keeping a single component with two kinds
// preserves the layout/styling logic without forcing a copy.
type ReleaseFilterBarKind int

const (
	ReleaseFilterBarHistory ReleaseFilterBarKind = iota
	ReleaseFilterBarPicker
)

// ReleaseFilterBar mirrors HistoryFilterBar but with the release-specific
// mode sets. The `kind` field decides whether CycleMode/Mode/View read
// from the release-history or picker package vars.
type ReleaseFilterBar struct {
	input   textinput.Model
	theme   *styles.Theme
	focused bool
	width   int
	visible int
	total   int
	kind    ReleaseFilterBarKind
}

func NewReleaseFilterBar(theme *styles.Theme) ReleaseFilterBar {
	return newReleaseFilterBar(theme, ReleaseFilterBarHistory)
}

// NewReleasePickerFilterBar is the picker-side constructor. Its bar
// cycles `currentReleasePickerFilterMode` (TITLE/HASH/TYPE/TAG) and the
// pill renders the picker labels.
func NewReleasePickerFilterBar(theme *styles.Theme) ReleaseFilterBar {
	return newReleaseFilterBar(theme, ReleaseFilterBarPicker)
}

func newReleaseFilterBar(theme *styles.Theme, kind ReleaseFilterBarKind) ReleaseFilterBar {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "type to filter…"
	ti.SetVirtualCursor(true)
	return ReleaseFilterBar{input: ti, theme: theme, kind: kind}
}

func (f *ReleaseFilterBar) CycleMode() {
	if f.kind == ReleaseFilterBarPicker {
		currentReleasePickerFilterMode = releasePickerFilterModeOrder[(int(currentReleasePickerFilterMode)+1)%len(releasePickerFilterModeOrder)]
		return
	}
	currentReleaseFilterMode = releaseFilterModeOrder[(int(currentReleaseFilterMode)+1)%len(releaseFilterModeOrder)]
}

func (f ReleaseFilterBar) Mode() ReleaseFilterMode { return currentReleaseFilterMode }
func (f *ReleaseFilterBar) SetSize(width int)      { f.width = width }
func (f *ReleaseFilterBar) SetCounts(visible, total int) {
	f.visible = visible
	f.total = total
}
func (f ReleaseFilterBar) Value() string   { return f.input.Value() }
func (f ReleaseFilterBar) IsFocused() bool { return f.focused }

func (f *ReleaseFilterBar) Focus() tea.Cmd {
	f.focused = true
	return f.input.Focus()
}

func (f *ReleaseFilterBar) Blur() {
	f.focused = false
	f.input.Blur()
}

func (f *ReleaseFilterBar) Reset() { f.input.SetValue("") }

func (f ReleaseFilterBar) Update(msg tea.Msg) (ReleaseFilterBar, tea.Cmd) {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}

// View mirrors HistoryFilterBar.View — same layout math, only the mode
// metadata table differs. Kept separate so future tweaks (e.g. adding a
// release-specific affix) don't have to thread through both screens.
func (f ReleaseFilterBar) View() string {
	var label, tag string
	if f.kind == ReleaseFilterBarPicker {
		meta, ok := releasePickerFilterModeMeta[currentReleasePickerFilterMode]
		if !ok {
			meta = releasePickerFilterModeMeta[ReleasePickerFilterModeTitle]
		}
		label, tag = meta.label, meta.tag
	} else {
		meta, ok := releaseFilterModeMeta[currentReleaseFilterMode]
		if !ok {
			meta = releaseFilterModeMeta[ReleaseFilterModeTitle]
		}
		label, tag = meta.label, meta.tag
	}
	modePill := styles.CommitTypeMsgStyle(f.theme, tag).
		Bold(true).
		Padding(0, 1).
		Render(label)

	arrowColor := f.theme.Muted
	if f.focused {
		arrowColor = f.theme.Primary
	}
	arrow := lipgloss.NewStyle().Foreground(arrowColor).Render(" > ")

	prefix := modePill + arrow
	counterText := fmt.Sprintf("%d / %d", f.visible, f.total)
	counter := lipgloss.NewStyle().Foreground(f.theme.Muted).Render(counterText)

	prefixW := lipgloss.Width(prefix)
	counterW := lipgloss.Width(counter)

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
	rowW := lipgloss.Width(row)
	if rowW > f.width {
		row = ansi.Truncate(row, f.width, "")
	} else if rowW < f.width {
		row = row + strings.Repeat(" ", f.width-rowW)
	}
	return row
}
