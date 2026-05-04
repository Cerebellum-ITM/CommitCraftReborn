package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"commit_craft_reborn/internal/commit"
)

// CreateLocalConfigTomlTmpl writes a default `.commitcraft.toml` in the
// current working directory if one doesn't already exist. No-op when the
// file is already present so it never overwrites user config.
func CreateLocalConfigTomlTmpl() error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	configPath := filepath.Join(workDir, ".commitcraft.toml")
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	cfg := GetDefaultLocalConfig()
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
// Tags already present (matched case-insensitively by `tag`) are skipped
// so re-running is idempotent. Behavior is set to `"append"` only when
// it was previously empty, preserving any explicit user choice.
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

	var cfg Config
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
		cfg.CommitTypes.Types = append(cfg.CommitTypes.Types, CustomCommitType{
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

// LocalConfigPath returns the absolute path to `.commitcraft.toml` in
// the current working directory. Useful for CLI subcommands that want to
// surface the path back to the caller after writing.
func LocalConfigPath() (string, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(workDir, ".commitcraft.toml"), nil
}
