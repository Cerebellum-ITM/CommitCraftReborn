package tui

import (
	tea "charm.land/bubbletea/v2"
)

// programQuitMsg is dispatched by popup models (which have no Model
// reference) to request a graceful program exit. The root Update
// intercepts it and routes through quitWithAutodraft so the autodraft
// hook runs even when the quit originates inside a popup.
type programQuitMsg struct{}

func programQuitCmd() tea.Cmd {
	return func() tea.Msg { return programQuitMsg{} }
}

// populateCurrentCommitFromBuffers copies the live compose/pipeline
// buffers into model.currentCommit so it can be persisted via
// db.SaveDraft. Shared by the manual Save Draft handler (Ctrl+D) and
// the autodraft-on-quit hook to keep the two paths in sync.
func populateCurrentCommitFromBuffers(model *Model) {
	model.currentCommit.KeyPoints = model.keyPoints
	model.currentCommit.MessageEN = model.commitTranslate
	model.currentCommit.Type = model.commitType
	model.currentCommit.Scope = model.commitScope
	model.currentCommit.Workspace = model.pwd
	model.currentCommit.Diff_code = model.diffCode
	model.currentCommit.IaSummary = model.iaSummaryOutput
	model.currentCommit.IaCommitRaw = model.iaCommitRawOutput
	model.currentCommit.IaTitle = model.iaTitleRawOutput
	model.currentCommit.IaChangelog = model.iaChangelogEntry
}

// hasAutodraftableContent reports whether there is anything worth
// persisting before quitting. Returns true if there is already a
// linked DB row (so SaveDraft will UPDATE it) or any of the
// user-facing buffers carry content.
func hasAutodraftableContent(model *Model) bool {
	if model.currentCommit.ID != 0 {
		return true
	}
	if model.commitType != "" || model.commitScope != "" {
		return true
	}
	if len(model.keyPoints) > 0 {
		return true
	}
	if model.commitMsg != "" || model.commitTranslate != "" {
		return true
	}
	if model.iaSummaryOutput != "" || model.iaCommitRawOutput != "" {
		return true
	}
	if model.iaTitleRawOutput != "" || model.iaChangelogEntry != "" {
		return true
	}
	return false
}

// autodraftIfNeeded persists the in-memory compose/pipeline buffers as
// a draft when the user is leaving from COMPOSE or PIPELINE and has
// meaningful unsaved data. Errors are logged and swallowed so a DB
// failure can never trap the user inside the TUI.
func autodraftIfNeeded(model *Model) {
	// User completed the flow (printed a final commit/release message) —
	// nothing to draft. Without this guard the exit from stateOutput,
	// stateConfirming, etc. would still match TabCompose and emit a
	// misleading "Exit in Compose — draft saved" notice.
	if model.FinalMessage != "" {
		return
	}
	tab := tabForState(model.state)
	if tab != TabCompose && tab != TabPipeline {
		return
	}
	if !hasAutodraftableContent(model) {
		return
	}
	populateCurrentCommitFromBuffers(model)
	if err := model.db.SaveDraft(&model.currentCommit); err != nil {
		model.log.Error("autodraft on quit failed", "error", err)
		return
	}
	persistPipelineAICalls(model, model.currentCommit.ID)
	model.AutodraftedID = model.currentCommit.ID
	model.AutodraftedTab = tabLabel(tab)
}

// quitWithAutodraft is the canonical "user is leaving the TUI" exit.
// Use it instead of returning tea.Quit directly from any handler that
// has access to the Model. Popup models that lack a Model reference
// should emit programQuitCmd() instead.
func quitWithAutodraft(model *Model) (tea.Model, tea.Cmd) {
	autodraftIfNeeded(model)
	return model, tea.Quit
}
