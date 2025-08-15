package config

import (
	"bytes"
	"commit_craft_reborn/internal/commit"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	// "github.com/charmbracelet/log"
)

const (
	localConfigName  = ".commitcraft.toml"
	globalConfigDir  = ".config/commitcraft"
	globalConfigName = "config.toml"
)

func LoadConfig() (Config, error) {
	cfg := NewDefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("could not get user home directory: %w", err)
	}

	globalDir := filepath.Join(home, globalConfigDir)
	globalPath := filepath.Join(globalDir, globalConfigName)

	if err := ensureGlobalConfigExists(globalDir, globalPath); err != nil {
		return Config{}, err
	}

	if _, err := toml.DecodeFile(globalPath, &cfg); err != nil {
		return Config{}, fmt.Errorf("error parsing global config file at %s: %w", globalPath, err)
	}

	if _, err := os.Stat(localConfigName); err == nil {
		if _, err := toml.DecodeFile(localConfigName, &cfg); err != nil {
			return Config{}, fmt.Errorf(
				"error parsing local config file (.commitcraft.toml): %w",
				err,
			)
		}
	}

	return cfg, nil
}

func ensureGlobalConfigExists(dirPath, filePath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("could not create global config directory at %s: %w", dirPath, err)
		}
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		defaultCfg := GetDefaultConfigWithTypes()

		var buf bytes.Buffer
		encoder := toml.NewEncoder(&buf)
		if err := encoder.Encode(defaultCfg); err != nil {
			return fmt.Errorf("could not encode default config to TOML: %w", err)
		}

		header := "# CommitCraft Global Configuration\n# This file was auto-generated. You can customize it.\n\n"
		content := append([]byte(header), buf.Bytes()...)

		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return fmt.Errorf("could not write default global config file to %s: %w", filePath, err)
		}
	}
	return nil
}

func ResolveCommitTypes(cfg Config) []commit.CommitType {
	defaultTypes := commit.GetDefaultCommitTypes()

	customTypes := make([]commit.CommitType, len(cfg.CommitTypes.Types))
	for i, ct := range cfg.CommitTypes.Types {
		customTypes[i] = commit.CommitType{Tag: ct.Tag, Description: ct.Description}
	}

	if cfg.CommitTypes.Behavior == "replace" {
		if len(customTypes) > 0 {
			return customTypes
		}
		return defaultTypes
	}

	return append(defaultTypes, customTypes...)
}
