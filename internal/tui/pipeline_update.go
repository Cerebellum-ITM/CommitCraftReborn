package tui

import (
	"fmt"
	"time"

	"commit_craft_reborn/internal/tui/statusbar"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// updatePipeline is the Pipeline tab's per-state Update handler. It owns
// keyboard shortcuts (retry / accept / cancel / panel switch) and forwards
// progress / spinner / file-list ticks to the right sub-component.
func updatePipeline(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(m, model.keys.Toggle): // r — full retry
			return model, model.pipelineStartFullRun()
		case key.Matches(m, model.keys.RerunStage1):
			return model, model.pipelineRetryStage(stageSummary)
		case key.Matches(m, model.keys.RerunStage2):
			return model, model.pipelineRetryStage(stageBody)
		case key.Matches(m, model.keys.RerunStage3):
			return model, model.pipelineRetryStage(stageTitle)
		case key.Matches(m, model.keys.Up):
			model.pipeline.diffViewport.ScrollUp(1)
			return model, nil
		case key.Matches(m, model.keys.Down):
			model.pipeline.diffViewport.ScrollDown(1)
			return model, nil
		case key.Matches(m, model.keys.PgUp):
			vp := model.stageViewportModel(model.pipeline.focusedStage)
			if vp != nil {
				vp.HalfPageUp()
			}
			return model, nil
		case key.Matches(m, model.keys.PgDown):
			vp := model.stageViewportModel(model.pipeline.focusedStage)
			if vp != nil {
				vp.HalfPageDown()
			}
			return model, nil
		case key.Matches(m, model.keys.FileUp):
			if len(model.pipelineDiffList.Items()) > 0 {
				model.pipelineDiffList.CursorUp()
				setDiffFromSelectedFile(model)
			}
			return model, nil
		case key.Matches(m, model.keys.FileDown):
			if len(model.pipelineDiffList.Items()) > 0 {
				model.pipelineDiffList.CursorDown()
				setDiffFromSelectedFile(model)
			}
			return model, nil
		case key.Matches(m, model.keys.NextField):
			model.pipeline.cycleFocus()
			return model, nil
		case key.Matches(m, model.keys.Esc):
			if model.pipeline.anyRunning() {
				return model, model.pipelineCancel()
			}
			// Outside of a running pipeline, fall through to the global
			// tab-switch hint by leaving state unchanged.
			return model, nil
		case key.Matches(m, model.keys.Enter):
			if model.pipeline.allDone() {
				return createCommit(model)
			}
			return model, nil
		}

	case spinner.TickMsg:
		if model.pipeline.anyRunning() {
			var spCmd tea.Cmd
			model.pipeline.spinner, spCmd = model.pipeline.spinner.Update(m)
			cmds = append(cmds, spCmd)
		}

	case progress.FrameMsg:
		// Forward progress animation frames to every bar — they discard
		// frames that don't match their internal id.
		for i := range model.pipeline.progress {
			updated, pCmd := model.pipeline.progress[i].Update(msg)
			model.pipeline.progress[i] = updated
			cmds = append(cmds, pCmd)
		}

	case tickPulseMsg:
		if model.pipeline.anyRunning() {
			model.pipeline.pulsePhase++
			cmds = append(cmds, tickPulse())
		}

	case tickFlashMsg:
		// flashExpiresAt is checked in the View; nothing else to do here
		// other than triggering a redraw, which already happens because
		// the message itself causes Update→View.
		_ = m

	case tickFadeMsg:
		if m.frame > model.pipeline.fadeFrame {
			model.pipeline.fadeFrame = m.frame
		}
		if m.frame < 2 {
			cmds = append(cmds, tickFade(m.frame+1))
		}

	case tickShakeMsg:
		st := &model.pipeline.stages[m.id]
		if m.frame >= 3 {
			st.shakeFrame = 0
		} else {
			st.shakeFrame = m.frame
			cmds = append(cmds, tickShake(m.id, m.frame+1))
		}
	}

	return model, tea.Batch(cmds...)
}

// pipelineStartFullRun resets every stage and kicks off the regular
// commit-builder command (same one Ctrl+W uses on Compose).
func (model *Model) pipelineStartFullRun() tea.Cmd {
	if len(model.commitScopes) == 0 {
		model.WritingStatusBar.Level = statusbar.LevelError
		model.WritingStatusBar.Content = "Scope is required before requesting the AI. Add at least one scope."
		return nil
	}
	if len(model.keyPoints) == 0 {
		model.WritingStatusBar.Level = statusbar.LevelError
		model.WritingStatusBar.Content = "At least one key point is required before requesting the AI."
		return nil
	}
	model.pipeline.resetAll(time.Now())
	model.commitTranslate = ""
	model.iaSummaryOutput = ""
	model.iaCommitRawOutput = ""
	model.iaTitleRawOutput = ""
	model.pipelineViewport1.SetContent("")
	model.pipelineViewport2.SetContent("")
	model.pipelineViewport3.SetContent("")
	if !model.useDbCommmit {
		refreshPipelineNumstat(model)
		applyPipelineFilesDelegate(model)
	}

	model.WritingStatusBar.Content = "pipeline started · stage 1/3"
	model.WritingStatusBar.Level = statusbar.LevelInfo

	return tea.Batch(
		model.WritingStatusBar.StartSpinner(),
		model.pipeline.spinner.Tick,
		tickPulse(),
		callIaCommitBuilderCmd(model),
	)
}

// pipelineRetryStage resets `from` and every downstream stage, then calls
// the matching command. The cascading reset mirrors what the underlying
// runner already does (stage 1 redoes everything, stage 2 redoes 2+3,
// stage 3 redoes only itself).
func (model *Model) pipelineRetryStage(from stageID) tea.Cmd {
	if len(model.commitScopes) == 0 {
		model.WritingStatusBar.Level = statusbar.LevelError
		model.WritingStatusBar.Content = "Scope is required before requesting the AI. Add at least one scope."
		return nil
	}
	if len(model.keyPoints) == 0 {
		model.WritingStatusBar.Level = statusbar.LevelError
		model.WritingStatusBar.Content = "At least one key point is required before requesting the AI."
		return nil
	}
	model.pipeline.resetFrom(from, time.Now())
	model.commitTranslate = ""

	switch from {
	case stageSummary:
		model.iaSummaryOutput = ""
		model.iaCommitRawOutput = ""
		model.iaTitleRawOutput = ""
	case stageBody:
		model.iaCommitRawOutput = ""
		model.iaTitleRawOutput = ""
	case stageTitle:
		model.iaTitleRawOutput = ""
	}

	model.WritingStatusBar.Content = fmt.Sprintf("pipeline retry · stage %d", int(from)+1)
	model.WritingStatusBar.Level = statusbar.LevelAI

	cmds := []tea.Cmd{
		model.WritingStatusBar.StartSpinner(),
		model.pipeline.spinner.Tick,
		tickPulse(),
	}
	switch from {
	case stageSummary:
		cmds = append(cmds, callIaSummaryCmd(model))
	case stageBody:
		cmds = append(cmds, callIaCommitBuilderStage2Cmd(model))
	case stageTitle:
		// callIaOutputFormatCmd is named for legacy reasons — today it
		// drives the commit-title generator.
		cmds = append(cmds, callIaOutputFormatCmd(model))
	}
	return tea.Batch(cmds...)
}

// pipelineCancel marks running stages as Cancelled and drains their
// progress bars to 0% over ~250ms (built-in easing).
func (model *Model) pipelineCancel() tea.Cmd {
	model.pipeline.cancelling = true
	cancelledIdx := -1
	cmds := make([]tea.Cmd, 0, 4)
	for i := range model.pipeline.stages {
		if model.pipeline.stages[i].Status == statusRunning {
			model.pipeline.stages[i].Status = statusCancelled
			model.pipeline.stages[i].Progress = 0
			cmds = append(cmds, model.pipeline.progress[i].SetPercent(0))
			if cancelledIdx < 0 {
				cancelledIdx = i
			}
		}
	}
	model.WritingStatusBar.Content = fmt.Sprintf(
		"pipeline cancelled · stage %d stopped",
		cancelledIdx+1,
	)
	model.WritingStatusBar.Level = statusbar.LevelWarning
	cmds = append(cmds, model.WritingStatusBar.StopSpinner())
	return tea.Batch(cmds...)
}

// applyPipelineResult synchronises the pipeline view's per-stage status
// when one of the existing IaXxxResultMsg arrives. Called from update.go
// alongside the existing Compose-side handlers. Also pushes the freshly
// produced output into the per-stage viewport so the user can scroll
// through the full text right after the run.
func (model *Model) applyPipelineResult(touched []stageID, err error) tea.Cmd {
	now := time.Now()
	cmds := make([]tea.Cmd, 0, len(touched)+2)
	for _, id := range touched {
		st := &model.pipeline.stages[id]
		if err != nil {
			st.Status = statusFailed
			st.Err = err
			cmds = append(cmds, tickShake(id, 1))
			continue
		}
		if st.StartedAt.IsZero() {
			st.StartedAt = now.Add(-time.Second)
		}
		st.Status = statusDone
		st.Progress = 1
		st.Latency = now.Sub(st.StartedAt)
		st.flashExpiresAt = now.Add(400 * time.Millisecond)
		cmds = append(cmds, model.pipeline.progress[id].SetPercent(1))
		cmds = append(cmds, tickFlash(id, 400*time.Millisecond))

		if vp := model.stageViewportModel(id); vp != nil {
			switch id {
			case stageSummary:
				vp.SetContent(model.iaSummaryOutput)
			case stageBody:
				vp.SetContent(model.iaCommitRawOutput)
			case stageTitle:
				vp.SetContent(model.iaTitleRawOutput)
			}
			vp.GotoTop()
		}
	}
	if err == nil && model.pipeline.allDone() {
		cmds = append(cmds, tickFade(1))
	}
	return tea.Batch(cmds...)
}
