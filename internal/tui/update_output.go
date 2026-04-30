package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// updateOutput handles input for the Output review screen. Tab cycles
// focus between the report (left) and the content viewer (right). Enter
// prints the assembled final message to stdout (via tea.Quit and
// cmd/cli/main.go); Esc returns to the history list.
func updateOutput(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	m, ok := msg.(tea.KeyMsg)
	if !ok {
		return model, nil
	}
	switch {
	case key.Matches(m, model.keys.Enter):
		model.FinalMessage = outputCommitMessageOrFallback(model, model.currentCommit)
		return quitWithAutodraft(model)
	case key.Matches(m, model.keys.Esc):
		return model.cancelProcess(stateChoosingCommit)
	case key.Matches(m, model.keys.NextField):
		toggleOutputFocus(model, true)
		return model, nil
	case key.Matches(m, model.keys.PrevField):
		toggleOutputFocus(model, false)
		return model, nil
	}

	if model.focusedElement == focusOutputReport {
		switch {
		case key.Matches(m, model.keys.Up):
			model.outputReportViewport.ScrollUp(1)
		case key.Matches(m, model.keys.Down):
			model.outputReportViewport.ScrollDown(1)
		case key.Matches(m, model.keys.PgUp):
			model.outputReportViewport.ScrollUp(model.outputReportViewport.Height() - 1)
		case key.Matches(m, model.keys.PgDown):
			model.outputReportViewport.ScrollDown(model.outputReportViewport.Height() - 1)
		}
		return model, nil
	}

	// focusOutputContent
	switch {
	case key.Matches(m, model.keys.Left):
		cycleOutputSegment(model, false)
	case key.Matches(m, model.keys.Right):
		cycleOutputSegment(model, true)
	case key.Matches(m, model.keys.Up):
		model.iaViewport.ScrollUp(1)
	case key.Matches(m, model.keys.Down):
		model.iaViewport.ScrollDown(1)
	case key.Matches(m, model.keys.PgUp):
		model.iaViewport.ScrollUp(model.iaViewport.Height() - 1)
	case key.Matches(m, model.keys.PgDown):
		model.iaViewport.ScrollDown(model.iaViewport.Height() - 1)
	}
	return model, nil
}

// toggleOutputFocus flips between the report (left) and the content
// viewer (right) panes. Both panes are scrollable; the right one also
// owns the segmented selector.
func toggleOutputFocus(model *Model, _ bool) {
	if model.focusedElement == focusOutputReport {
		model.focusedElement = focusOutputContent
		return
	}
	model.focusedElement = focusOutputReport
}

// openOutputViewFromHistory hydrates the Model with a previously stored
// commit (already in model.currentCommit at popup-open time) and lands
// on stateOutput so the user can review telemetry, swap segments and
// re-print the message without going through the Compose flow.
func openOutputViewFromHistory(model *Model) (tea.Model, tea.Cmd) {
	model.popup = nil
	commit := model.currentCommit
	model.loadScopesFromString(commit.Scope)
	model.commitType = commit.Type
	model.keyPoints = commit.KeyPoints
	model.diffCode = commit.Diff_code
	model.commitTranslate = commit.MessageEN
	model.iaSummaryOutput = commit.IaSummary
	model.iaCommitRawOutput = commit.IaCommitRaw
	model.iaTitleRawOutput = commit.IaTitle
	model.iaChangelogEntry = commit.IaChangelog
	loadPipelineAICalls(model, commit.ID)
	model.state = stateOutput
	model.keys = outputViewKeys()
	model.outputSegment = outSegFinal
	model.focusedElement = focusOutputContent
	model.outputReportViewport.GotoTop()
	model.iaViewport.GotoTop()
	model.WritingStatusBar.Content = "Output review · enter to print · esc to history"
	return model, nil
}

// cycleOutputSegment advances the right-pane segment selector; resets
// the iaViewport scroll so the new content starts at the top.
func cycleOutputSegment(model *Model, forward bool) {
	defs := model.outputSegmentDefs()
	if len(defs) == 0 {
		return
	}
	cur := 0
	for i, d := range defs {
		if d.id == model.outputSegment {
			cur = i
			break
		}
	}
	if forward {
		cur = (cur + 1) % len(defs)
	} else {
		cur = (cur - 1 + len(defs)) % len(defs)
	}
	model.outputSegment = defs[cur].id
	model.iaViewport.GotoTop()
}
