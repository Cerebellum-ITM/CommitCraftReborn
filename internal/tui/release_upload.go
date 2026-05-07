package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/logger"
)

// UploadReleaseToGithub publishes a release to GitHub using the `gh` CLI.
// The tag, repository, and binary asset path are sourced from
// config.ReleaseConfig; the body comes from the selected stored release.
func UploadReleaseToGithub(
	selectedItem HistoryReleaseItem,
	pwd string,
	config *config.Config,
	logger *logger.Logger,
	tools Tools,
) error {
	if !tools.gh.available {
		return fmt.Errorf("The Github CLI is not available on the system")
	}

	var files []string
	assetsPath := config.ReleaseConfig.BinaryAssetsPath
	if assetsPath != "" {
		assetPath := filepath.Join(pwd, assetsPath)
		if info, statErr := os.Stat(assetPath); statErr == nil && info.IsDir() {
			err := filepath.Walk(assetPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	tmpFile, err := os.CreateTemp("", "release-notes-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for release notes: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	token := config.ReleaseConfig.GhToken
	tag := config.ReleaseConfig.Version
	repository := config.ReleaseConfig.Repository
	title := fmt.Sprintf("Release %s: %s", tag, selectedItem.release.Title)

	_, err = tmpFile.WriteString(selectedItem.release.Body)
	if err != nil {
		return fmt.Errorf("failed to write release notes to temporary file: %w", err)
	}
	tmpFile.Sync()
	notesFilePath := tmpFile.Name()

	args := []string{
		"release", "create", tag,
		"--repo", repository,
		"--title", title,
		"--notes-file", notesFilePath,
	}
	args = append(args, files...)

	logger.Debug(fmt.Sprintf("gh %s", strings.Join(args, " ")))

	cmd := exec.Command("gh", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GH_TOKEN=%s", token))
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err = cmd.Run()
	if err != nil {
		logger.Debug(err.Error())
		logger.Debug(errb.String())
		return fmt.Errorf(
			"error running command: stdout: %s, stderr: %s, err: %w",
			outb.String(),
			errb.String(),
			err,
		)
	}

	return nil
}

// execUploadRelease wraps UploadReleaseToGithub as a tea.Cmd for use inside
// the TUI message loop.
func execUploadRelease(releaseItem HistoryReleaseItem, model *Model) tea.Cmd {
	return func() tea.Msg {
		err := UploadReleaseToGithub(
			releaseItem,
			model.pwd,
			&model.globalConfig,
			model.log,
			model.ToolsInfo,
		)
		return releaseUpdloadResultMsg{Err: err}
	}
}
