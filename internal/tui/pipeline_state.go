package tui

import (
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"

	"commit_craft_reborn/internal/git"
)

// stageID identifies one of the three AI stages displayed on the Pipeline
// tab. The numeric values are reused as indices into the
// pipelineModel.stages / .progress arrays — keep them contiguous.
type stageID int

const (
	stageSummary   stageID = iota // Stage 1 — change analyzer
	stageBody                     // Stage 2 — commit body generator
	stageTitle                    // Stage 3 — commit title generator
	stageChangelog                // Stage 4 — changelog refiner (optional)
)

// stageStatus is the per-stage lifecycle state rendered as a status pill
// and used to drive animations (running → pulse, done → flash, …).
type stageStatus int

const (
	statusIdle stageStatus = iota
	statusRunning
	statusDone
	statusFailed
	statusCancelled
)

// pipelineStage is the per-stage record displayed in the right panel.
// flashExpiresAt holds the post-completion green-flash deadline; rendering
// checks time.Now() against it. shakeFrame drives the failure shake (0 =
// centered, 1 = shifted left, 2 = shifted right).
type pipelineStage struct {
	ID               stageID
	Title            string
	Model            string
	Status           stageStatus
	Progress         float64
	Latency          time.Duration
	StartedAt        time.Time
	Err              error
	flashExpiresAt   time.Time
	shakeFrame       int
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	QueueTime        time.Duration
	PromptTime       time.Duration
	CompletionTime   time.Duration
	APITotalTime     time.Duration
	RequestID        string
	StatsModel       string
	HasStats         bool
	TPMLimitAtCall   int
	// History keeps every successful AI response for this stage during
	// the current session so the user can compare alternatives via the
	// stage history popup (key `H`). Append-only; cleared on commit
	// finalize. ActiveHistoryIndex points at the entry currently mirrored
	// onto the live fields above (-1 when empty).
	History            []stageHistoryEntry
	ActiveHistoryIndex int
}

// stageHistoryEntry is a snapshot of one AI generation for a stage —
// the raw text plus the per-call telemetry that was on screen when the
// entry was captured. Lives only in memory; never persisted to SQLite.
type stageHistoryEntry struct {
	Text             string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	QueueTime        time.Duration
	PromptTime       time.Duration
	CompletionTime   time.Duration
	APITotalTime     time.Duration
	RequestID        string
	TPMLimitAtCall   int
	CapturedAt       time.Time
}

// pipelineModel groups every Pipeline-tab-specific piece of state on the
// main Model. It lives as a single embedded field so resets and animation
// scheduling stay local to this file.
type pipelineModel struct {
	stages   [4]pipelineStage
	progress [4]progress.Model
	spinner  spinner.Model
	// diffViewport renders the current selected file's staged diff in the
	// last sub-block of the right panel. Updated whenever the user moves
	// the file cursor (j/k).
	diffViewport viewport.Model
	// numstat is the cached `git diff --staged --numstat` map keyed by
	// path, refreshed on tab enter / pipeline re-run. Used by the file
	// list rows and the footer totals.
	numstat map[string]git.FileNumstat
	width   int
	height  int
	// focusedStage tracks which of the 3 stage cards is currently grown
	// (config.StageFocusedHeight). Tab cycles s1 → s2 → s3 → s1.
	focusedStage stageID
	cancelling   bool
	// fadeFrame drives the final-commit fade-in. 0 = hidden, 1 = Muted,
	// 2 = AcceptDim, 3 = Success (final). Reset on every full re-run.
	fadeFrame int
	// pulsePhase is incremented on every tickPulseMsg while at least one
	// stage is Running. Used to animate the indeterminate progress bars.
	pulsePhase int
	// activeStages is the number of stages currently part of the run. The
	// optional changelog refiner only counts when the project's CHANGELOG
	// was detected at pipeline start. Defaults to 3.
	activeStages int
	// focusedFinal flips on when Tab cycles past the last stage onto the
	// final-commit card slot. Only meaningful while the final card is
	// visible (allDone + commitTranslate set).
	focusedFinal bool
}

// newPipelineModel builds the Pipeline tab's initial state. It does not
// allocate the file list (that lives on the parent Model as
// pipelineDiffList so it can share git status across views).
func newPipelineModel() pipelineModel {
	titles := [4]string{"Change Analyzer", "Commit Body", "Commit Title", "Changelog Refiner"}
	pm := pipelineModel{focusedStage: stageSummary, activeStages: 3}
	for i := 0; i < len(pm.stages); i++ {
		pm.stages[i] = pipelineStage{
			ID:                 stageID(i),
			Title:              titles[i],
			Status:             statusIdle,
			ActiveHistoryIndex: -1,
		}
		pm.progress[i] = progress.New(
			progress.WithDefaultBlend(),
			progress.WithoutPercentage(),
		)
	}
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	pm.spinner = sp
	pm.diffViewport = viewport.New()
	return pm
}

// cycleFocus advances the focus cursor by one slot, wrapping at the last
// *active* stage and — when showFinal is true — also through the
// final-commit card. The cycle order is: stage 1 → … → stage N → final →
// stage 1.
func (pm *pipelineModel) cycleFocus(showFinal bool) {
	active := pm.activeStages
	if active < 1 {
		active = 1
	}
	total := active
	if showFinal {
		total++
	}

	current := int(pm.focusedStage)
	if pm.focusedFinal {
		current = active // virtual slot for the final card
	}

	next := (current + 1) % total
	if next >= active {
		pm.focusedFinal = true
	} else {
		pm.focusedFinal = false
		pm.focusedStage = stageID(next)
	}
}

// resetAll marks every stage as Running with progress 0 and clears
// per-stage flashes/errors. Used by the full re-run shortcut (`r`).
func (pm *pipelineModel) resetAll(now time.Time) {
	pm.fadeFrame = 0
	pm.cancelling = false
	pm.focusedFinal = false
	for i := range pm.stages {
		if i >= pm.activeStages {
			// Inactive stages (e.g. the changelog refiner when no
			// CHANGELOG file is present) stay Idle so they don't leak
			// into the rendered card list.
			pm.stages[i].Status = statusIdle
			pm.stages[i].Progress = 0
			pm.stages[i].Err = nil
			pm.stages[i].flashExpiresAt = time.Time{}
			pm.stages[i].shakeFrame = 0
			continue
		}
		pm.stages[i].Status = statusRunning
		pm.stages[i].Progress = 0
		pm.stages[i].StartedAt = now
		pm.stages[i].Latency = 0
		pm.stages[i].Err = nil
		pm.stages[i].flashExpiresAt = time.Time{}
		pm.stages[i].shakeFrame = 0
		pm.stages[i].HasStats = false
		pm.stages[i].PromptTokens = 0
		pm.stages[i].CompletionTokens = 0
		pm.stages[i].TotalTokens = 0
		pm.stages[i].QueueTime = 0
		pm.stages[i].PromptTime = 0
		pm.stages[i].CompletionTime = 0
		pm.stages[i].APITotalTime = 0
		pm.stages[i].RequestID = ""
		pm.stages[i].StatsModel = ""
		pm.stages[i].TPMLimitAtCall = 0
	}
}

// resetFrom sets every stage from `from` (inclusive) onward to Running.
// Used by the per-stage retry shortcuts (1/2/3) — retrying stage 1
// cascades through 2 and 3, retrying stage 2 cascades through 3, and
// retrying stage 3 only resets itself.
func (pm *pipelineModel) resetFrom(from stageID, now time.Time) {
	pm.fadeFrame = 0
	pm.cancelling = false
	for i := int(from); i < len(pm.stages); i++ {
		if i >= pm.activeStages {
			pm.stages[i].Status = statusIdle
			pm.stages[i].Progress = 0
			pm.stages[i].Err = nil
			pm.stages[i].flashExpiresAt = time.Time{}
			pm.stages[i].shakeFrame = 0
			continue
		}
		pm.stages[i].Status = statusRunning
		pm.stages[i].Progress = 0
		pm.stages[i].StartedAt = now
		pm.stages[i].Latency = 0
		pm.stages[i].Err = nil
		pm.stages[i].flashExpiresAt = time.Time{}
		pm.stages[i].shakeFrame = 0
		pm.stages[i].HasStats = false
		pm.stages[i].PromptTokens = 0
		pm.stages[i].CompletionTokens = 0
		pm.stages[i].TotalTokens = 0
		pm.stages[i].QueueTime = 0
		pm.stages[i].PromptTime = 0
		pm.stages[i].CompletionTime = 0
		pm.stages[i].APITotalTime = 0
		pm.stages[i].RequestID = ""
		pm.stages[i].StatsModel = ""
		pm.stages[i].TPMLimitAtCall = 0
	}
}

// pushStageHistory snapshots the current per-stage telemetry plus the
// freshly produced AI text into the stage's History slice and points
// ActiveHistoryIndex at the new entry. No-op for invalid ids or empty
// text (failed runs shouldn't pollute the picker).
func (pm *pipelineModel) pushStageHistory(id stageID, text string) {
	if int(id) < 0 || int(id) >= len(pm.stages) {
		return
	}
	if text == "" {
		return
	}
	st := &pm.stages[id]
	st.History = append(st.History, stageHistoryEntry{
		Text:             text,
		Model:            st.StatsModel,
		PromptTokens:     st.PromptTokens,
		CompletionTokens: st.CompletionTokens,
		TotalTokens:      st.TotalTokens,
		QueueTime:        st.QueueTime,
		PromptTime:       st.PromptTime,
		CompletionTime:   st.CompletionTime,
		APITotalTime:     st.APITotalTime,
		RequestID:        st.RequestID,
		TPMLimitAtCall:   st.TPMLimitAtCall,
		CapturedAt:       time.Now(),
	})
	st.ActiveHistoryIndex = len(st.History) - 1
}

// clearAllHistory drops every per-stage history entry. Called from the
// commit-finalize path so the next commit starts with a clean slate;
// SaveDraft never calls this (the user may keep iterating).
func (pm *pipelineModel) clearAllHistory() {
	for i := range pm.stages {
		pm.stages[i].History = nil
		pm.stages[i].ActiveHistoryIndex = -1
	}
}

// applyStageHistoryEntry mirrors a stored entry onto the live stage
// fields so the card renders the chosen version's stats.
func (pm *pipelineModel) applyStageHistoryEntry(id stageID, index int) (stageHistoryEntry, bool) {
	if int(id) < 0 || int(id) >= len(pm.stages) {
		return stageHistoryEntry{}, false
	}
	st := &pm.stages[id]
	if index < 0 || index >= len(st.History) {
		return stageHistoryEntry{}, false
	}
	entry := st.History[index]
	st.PromptTokens = entry.PromptTokens
	st.CompletionTokens = entry.CompletionTokens
	st.TotalTokens = entry.TotalTokens
	st.QueueTime = entry.QueueTime
	st.PromptTime = entry.PromptTime
	st.CompletionTime = entry.CompletionTime
	st.APITotalTime = entry.APITotalTime
	st.RequestID = entry.RequestID
	st.StatsModel = entry.Model
	st.TPMLimitAtCall = entry.TPMLimitAtCall
	st.HasStats = true
	st.ActiveHistoryIndex = index
	return entry, true
}

// anyRunning reports whether the pulse/spinner ticks should keep firing.
func (pm *pipelineModel) anyRunning() bool {
	for i := range pm.stages {
		if pm.stages[i].Status == statusRunning {
			return true
		}
	}
	return false
}

// allDone reports whether every *active* stage finished successfully — gates
// the final-commit fade-in and the `enter` accept shortcut. Inactive stages
// (the changelog refiner when disabled or no CHANGELOG file) are ignored.
func (pm *pipelineModel) allDone() bool {
	limit := pm.activeStages
	if limit <= 0 || limit > len(pm.stages) {
		limit = len(pm.stages)
	}
	for i := 0; i < limit; i++ {
		if pm.stages[i].Status != statusDone {
			return false
		}
	}
	return true
}
