package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
)

// CreateLocalConfigTomlTmpl writes a default `.commitcraft.toml` in the
// current working directory if one doesn't already exist. No-op when the
// file is already present so it never overwrites user config.
func CreateLocalConfigTomlTmpl() error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	configPath := filepath.Join(workDir, ".commitcraft.toml")
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	cfg := config.GetDefaultLocalConfig()
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config to TOML: %w", err)
	}

	if err := os.WriteFile(configPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AppendCommitTypesToLocalConfig adds the given tags to the
// `[[commit_types.types]]` array of the workspace's `.commitcraft.toml`,
// creating the file from the default template if it doesn't exist yet.
// Tags already present (matched case-insensitively by `tag`) are
// skipped so re-running the popup is idempotent. Behavior is forced to
// `"append"` so the user's defaults remain selectable alongside the
// newly added entries.
func AppendCommitTypesToLocalConfig(types []commit.CommitType) (added int, err error) {
	if len(types) == 0 {
		return 0, nil
	}
	if err := CreateLocalConfigTomlTmpl(); err != nil {
		return 0, fmt.Errorf("ensuring local config exists: %w", err)
	}

	workDir, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	configPath := filepath.Join(workDir, ".commitcraft.toml")

	var cfg config.Config
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		return 0, fmt.Errorf("decoding %s: %w", configPath, err)
	}

	existing := make(map[string]struct{}, len(cfg.CommitTypes.Types))
	for _, ct := range cfg.CommitTypes.Types {
		existing[strings.ToUpper(ct.Tag)] = struct{}{}
	}

	for _, t := range types {
		key := strings.ToUpper(t.Tag)
		if _, dup := existing[key]; dup {
			continue
		}
		cfg.CommitTypes.Types = append(cfg.CommitTypes.Types, config.CustomCommitType{
			Tag:         t.Tag,
			Description: t.Description,
			BgBlock:     t.BgBlock,
			FgBlock:     t.FgBlock,
			BgMsg:       t.BgMsg,
			FgMsg:       t.FgMsg,
		})
		existing[key] = struct{}{}
		added++
	}

	if added == 0 {
		return 0, nil
	}
	if cfg.CommitTypes.Behavior == "" {
		cfg.CommitTypes.Behavior = "append"
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return 0, fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0o644); err != nil {
		return 0, fmt.Errorf("writing %s: %w", configPath, err)
	}
	return added, nil
}

// UpdateLocalConfigVersion sets release_config.version inside the repo's
// .commitcraft.toml. The file is created from the default template if it
// doesn't exist yet, so the user doesn't need to bootstrap it manually.
func UpdateLocalConfigVersion(version string) error {
	if err := CreateLocalConfigTomlTmpl(); err != nil {
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

// saveAPIKeyToEnv persists the user's Groq API key in the global config
// directory's `.env` file. Used by the API-key-setting state.
func saveAPIKeyToEnv(key string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, ".config", "CommitCraft")
	envPath := filepath.Join(configDir, ".env")

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	content := fmt.Sprintf("GROQ_API_KEY=%s\n", key)
	return os.WriteFile(envPath, []byte(content), 0o600)
}
