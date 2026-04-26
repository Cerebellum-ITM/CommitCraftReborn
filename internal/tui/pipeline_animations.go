package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// tickPulseMsg fires while at least one stage is running and drives the
// indeterminate progress pulse for the active stage rows.
type tickPulseMsg struct{}

// tickFlashMsg signals that the post-completion green flash should be
// re-evaluated for the given stage. The renderer reads
// pipelineStage.flashExpiresAt to decide whether to keep painting it.
type tickFlashMsg struct{ id stageID }

// tickFadeMsg advances the final-commit fade-in by one frame.
type tickFadeMsg struct{ frame int }

// tickShakeMsg advances the failure-shake animation. frame goes 1, 2, 3
// (centered) before the row stops moving.
type tickShakeMsg struct {
	id    stageID
	frame int
}

// tickPulse schedules the next indeterminate-bar pulse. The cadence
// (80ms) matches the eye-perceived smoothness for a triangle wave.
func tickPulse() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
		return tickPulseMsg{}
	})
}

// tickFlash schedules a re-render after `after` so the flash window can
// expire. The renderer checks time.Now() against flashExpiresAt directly.
func tickFlash(id stageID, after time.Duration) tea.Cmd {
	return tea.Tick(after, func(time.Time) tea.Msg {
		return tickFlashMsg{id: id}
	})
}

// tickFade is the 3-frame fade-in for the final-commit block. The
// renderer interprets fadeFrame as 0=hidden, 1=Muted, 2=AcceptDim,
// 3=Success.
func tickFade(frame int) tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return tickFadeMsg{frame: frame}
	})
}

// tickShake schedules the next frame of the failure shake. Frames 1 and
// 2 shift the row left/right by one column; frame 3 recenters it and
// ends the animation.
func tickShake(id stageID, frame int) tea.Cmd {
	return tea.Tick(60*time.Millisecond, func(time.Time) tea.Msg {
		return tickShakeMsg{id: id, frame: frame}
	})
}
