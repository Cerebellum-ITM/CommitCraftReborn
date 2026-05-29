package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"commit_craft_reborn/internal/config"
)

// UpdateLocalConfigVersion sets release_config.version inside the repo's
// .commitcraft.toml. The file is created from the default template if it
// doesn't exist yet, so the user doesn't need to bootstrap it manually.
func UpdateLocalConfigVersion(version string) error {
	if err := config.CreateLocalConfigTomlTmpl(); err != nil {
		return fmt.Errorf("ensuring local config exists: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := filepath.Join(workDir, ".commitcraft.toml")

	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return fmt.Errorf("decoding %s: %w", configPath, err)
	}
	cfg.ReleaseConfig.Version = version

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}
	return nil
}

// UpdateLocalConfigRelease writes the user-facing release fields into
// the repo's `.commitcraft.toml`. GH_TOKEN is never serialized here —
// it lives in ~/.config/CommitCraft/.env via SaveGhTokenToEnv. The
// file is created from the default template on first call so the user
// doesn't have to bootstrap it manually.
func UpdateLocalConfigRelease(
	repository, branch, version, assetsPath string,
	autoBuild bool,
	buildTool, buildTarget string,
) error {
	if err := config.CreateLocalConfigTomlTmpl(); err != nil {
		return fmt.Errorf("ensuring local config exists: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := filepath.Join(workDir, ".commitcraft.toml")

	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return fmt.Errorf("decoding %s: %w", configPath, err)
	}
	cfg.ReleaseConfig.Repository = repository
	cfg.ReleaseConfig.Branch = branch
	cfg.ReleaseConfig.Version = version
	cfg.ReleaseConfig.BinaryAssetsPath = assetsPath
	cfg.ReleaseConfig.AutoBuild = autoBuild
	cfg.ReleaseConfig.BuildTool = buildTool
	cfg.ReleaseConfig.BuildTarget = buildTarget

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}
	return nil
}

// UpdateLocalConfigChangelog writes the user-facing ChangelogConfig
// fields into the repo's `.commitcraft.toml`. The runtime-only Prompt
// field stays untouched (`toml:"-"`). Mirrors UpdateLocalConfigRelease.
func UpdateLocalConfigChangelog(
	enabled bool, path, bumpStrategy, promptFile, promptModel string,
) error {
	if err := config.CreateLocalConfigTomlTmpl(); err != nil {
		return fmt.Errorf("ensuring local config exists: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	configPath := filepath.Join(workDir, ".commitcraft.toml")

	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return fmt.Errorf("decoding %s: %w", configPath, err)
	}
	cfg.Changelog.Enabled = enabled
	cfg.Changelog.Path = path
	cfg.Changelog.BumpStrategy = bumpStrategy
	cfg.Changelog.PromptFile = promptFile
	cfg.Changelog.PromptModel = promptModel

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}
	return nil
}

// UpdateConfigTheme persists the chosen TUI theme to the global config
// (~/.config/CommitCraft/config.toml). The theme is a user-level
// preference; per-repo `.commitcraft.toml` files can still override it at
// read time but are never modified here.
func UpdateConfigTheme(theme string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	globalDir := filepath.Join(home, config.GlobalConfigDir)
	globalPath := filepath.Join(globalDir, "config.toml")
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		return err
	}

	var cfg config.Config
	if _, err := os.Stat(globalPath); err == nil {
		if _, err := toml.DecodeFile(globalPath, &cfg); err != nil {
			return fmt.Errorf("decoding %s: %w", globalPath, err)
		}
	} else {
		cfg = config.GetDefaultConfigWithTypes()
	}
	cfg.TUI.Theme = theme

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(globalPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", globalPath, err)
	}
	return nil
}

// saveEnvVar upserts a single KEY=VALUE pair in the global `.env` file.
// Thin wrapper around config.SaveEnvVar so the upsert logic lives in one
// place (the headless `commitcraft ai key` subcommand uses the same path).
func saveEnvVar(name, value string) error {
	return config.SaveEnvVar(name, value)
}

// saveAPIKeyToEnv persists the user's Groq API key in the global `.env`
// file (created at mode 0o600). Wrapper around saveEnvVar so other call
// sites can use the original ergonomic helper name.
func saveAPIKeyToEnv(key string) error {
	return saveEnvVar(config.EnvGroqUserKey, key)
}

// SaveGhTokenToEnv persists the GitHub personal-access token in the
// global `.env`. Exported because the release config popup lives in a
// different file and needs to call it after the user finishes the form.
func SaveGhTokenToEnv(token string) error {
	return saveEnvVar("GH_TOKEN", token)
}
