package tui

import (
	"time"

	"commit_craft_reborn/internal/git"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
)

// stageID identifies one of the three AI stages displayed on the Pipeline
// tab. The numeric values are reused as indices into the
// pipelineModel.stages / .progress arrays — keep them contiguous.
type stageID int

const (
	stageSummary stageID = iota // Stage 1 — change analyzer
	stageBody                   // Stage 2 — commit body generator
	stageTitle                  // Stage 3 — commit title generator
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
	ID             stageID
	Title          string
	Model          string
	Status         stageStatus
	Progress       float64
	Latency        time.Duration
	StartedAt      time.Time
	Err            error
	flashExpiresAt time.Time
	shakeFrame     int
}

// pipelineModel groups every Pipeline-tab-specific piece of state on the
// main Model. It lives as a single embedded field so resets and animation
// scheduling stay local to this file.
type pipelineModel struct {
	stages   [3]pipelineStage
	progress [3]progress.Model
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
}

// newPipelineModel builds the Pipeline tab's initial state. It does not
// allocate the file list (that lives on the parent Model as
// pipelineDiffList so it can share git status across views).
func newPipelineModel() pipelineModel {
	titles := [3]string{"Change Analyzer", "Commit Body", "Commit Title"}
	pm := pipelineModel{focusedStage: stageSummary}
	for i := 0; i < 3; i++ {
		pm.stages[i] = pipelineStage{
			ID:     stageID(i),
			Title:  titles[i],
			Status: statusIdle,
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

// cycleFocus advances the focused-stage cursor by one slot, wrapping at
// the third stage. Triggered by `tab` on the Pipeline tab.
func (pm *pipelineModel) cycleFocus() {
	pm.focusedStage = stageID((int(pm.focusedStage) + 1) % 3)
}

// resetAll marks every stage as Running with progress 0 and clears
// per-stage flashes/errors. Used by the full re-run shortcut (`r`).
func (pm *pipelineModel) resetAll(now time.Time) {
	pm.fadeFrame = 0
	pm.cancelling = false
	for i := range pm.stages {
		pm.stages[i].Status = statusRunning
		pm.stages[i].Progress = 0
		pm.stages[i].StartedAt = now
		pm.stages[i].Latency = 0
		pm.stages[i].Err = nil
		pm.stages[i].flashExpiresAt = time.Time{}
		pm.stages[i].shakeFrame = 0
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
		pm.stages[i].Status = statusRunning
		pm.stages[i].Progress = 0
		pm.stages[i].StartedAt = now
		pm.stages[i].Latency = 0
		pm.stages[i].Err = nil
		pm.stages[i].flashExpiresAt = time.Time{}
		pm.stages[i].shakeFrame = 0
	}
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

// allDone reports whether every stage finished successfully — gates the
// final-commit fade-in and the `enter` accept shortcut.
func (pm *pipelineModel) allDone() bool {
	for i := range pm.stages {
		if pm.stages[i].Status != statusDone {
			return false
		}
	}
	return true
}
