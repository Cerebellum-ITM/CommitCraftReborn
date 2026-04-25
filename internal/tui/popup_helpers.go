package tui

import (
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
)

// FilterState mirrors bubbles list filter states so we can refer to them
// from outside the bubbles package without aliasing every call site.
type FilterState int

const (
	Unfiltered    FilterState = iota
	Filtering                 // user is actively setting a filter
	FilterApplied             // a filter is applied and user is not editing filter
)

// calculatePopupPosition returns the top-left coordinates that center
// popupView inside a (modelWidth x modelHeight) viewport.
func calculatePopupPosition(modelWidth, modelHeight int, popupView string) (startX, startY int) {
	popupWidth := lipgloss.Width(popupView)
	popupHeight := lipgloss.Height(popupView)
	startX = (modelWidth - popupWidth) / 2
	startY = (modelHeight - popupHeight) / 2
	return startX, startY
}

// ResetAndActiveFilterOnList wipes any active filter on a bubbles list and
// activates the filter input so the user can immediately start typing.
func ResetAndActiveFilterOnList(l *list.Model) {
	if l != nil {
		l.ResetFilter()
		l.SetFilterText("")
		l.SetFilterState(list.FilterState(Filtering))
	}
}
