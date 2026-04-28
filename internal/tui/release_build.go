package tui

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"
)

// releaseBuildResultMsg reports the outcome of running the configured
// pre-release build command (e.g. `make build_release`).
type releaseBuildResultMsg struct {
	Err    error
	Output string
}

// execReleaseBuild runs the build command declared in
// ReleaseConfig (currently only `make <target>`) before kicking off the
// GitHub release upload. The caller must have validated that AutoBuild is
// enabled and BuildTool/BuildTarget are populated.
func execReleaseBuild(model *Model) tea.Cmd {
	return func() tea.Msg {
		cfg := model.globalConfig.ReleaseConfig
		cmd := exec.Command(cfg.BuildTool, cfg.BuildTarget)
		cmd.Dir = model.pwd
		out, err := cmd.CombinedOutput()
		return releaseBuildResultMsg{Err: err, Output: string(out)}
	}
}
