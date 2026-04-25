package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/logger"

	tea "charm.land/bubbletea/v2"
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
	assetPath := fmt.Sprintf("%s/%s", pwd, config.ReleaseConfig.BinaryAssetsPath)
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

	filesStr := strings.Join(files, " ")
	tmpFile, err := os.CreateTemp("", "release-notes-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temporary file for release notes: %w", err)
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	logger.Debug(filesStr)
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

	createCommand := fmt.Sprintf(
		"export GH_TOKEN=\"%s\" && gh release create \"%s\" --repo \"%s\" --title \"%s\" --notes-file \"%s\" %s",
		token,
		tag,
		repository,
		title,
		notesFilePath,
		filesStr,
	)
	cmd := exec.Command("sh", "-c", createCommand)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err = cmd.Run()
	if err != nil {
		logger.Debug(createCommand)
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
