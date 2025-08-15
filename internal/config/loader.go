package config

import (
	"bytes"
	"commit_craft_reborn/internal/commit"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	localConfigName  = ".commitcraft.toml"
	GlobalConfigDir  = ".config/commitcraft"
	globalConfigName = "config.toml"
)

func LoadConfigs() (globalCfg, localCfg Config, err error) {
	globalCfg = NewDefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, Config{}, fmt.Errorf("could not get user home directory: %w", err)
	}

	globalDir := filepath.Join(home, GlobalConfigDir)
	globalPath := filepath.Join(globalDir, globalConfigName)

	if err := ensureGlobalConfigExists(globalDir, globalPath); err != nil {
		return Config{}, Config{}, err
	}

	if _, err := toml.DecodeFile(globalPath, &globalCfg); err != nil {
		return Config{}, Config{}, fmt.Errorf(
			"error parsing global config file at %s: %w",
			globalPath,
			err,
		)
	}

	if _, err := os.Stat(localConfigName); err == nil {
		if _, err := toml.DecodeFile(localConfigName, &localCfg); err != nil {
			return Config{}, Config{}, fmt.Errorf(
				"error parsing local config file (.commitcraft.toml): %w",
				err,
			)
		}
	} else if !os.IsNotExist(err) {
		return Config{}, Config{}, fmt.Errorf("error checking local config file: %w", err)
	}

	return globalCfg, localCfg, nil
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

func ResolveCommitTypes(
	globalCfg, localCfg Config,
) []commit.CommitType {
	finalTypes := commit.GetDefaultCommitTypes()

	globalCustomTypes := make([]commit.CommitType, len(globalCfg.CommitTypes.Types))
	for i, ct := range globalCfg.CommitTypes.Types {
		globalCustomTypes[i] = commit.CommitType{
			Tag:         ct.Tag,
			Description: ct.Description,
			Color:       ct.Color,
		}
	}

	if len(globalCustomTypes) > 0 {
		if globalCfg.CommitTypes.Behavior == "replace" {
			finalTypes = globalCustomTypes
		} else {
			finalTypes = append(finalTypes, globalCustomTypes...)
		}
	}

	localCustomTypes := make([]commit.CommitType, len(localCfg.CommitTypes.Types))
	for i, ct := range localCfg.CommitTypes.Types {
		localCustomTypes[i] = commit.CommitType{
			Tag:         ct.Tag,
			Description: ct.Description,
			Color:       ct.Color,
		}
	}

	if len(localCustomTypes) > 0 {
		if localCfg.CommitTypes.Behavior == "replace" {
			finalTypes = localCustomTypes
		} else {
			finalTypes = append(finalTypes, localCustomTypes...)
		}
	}

	return finalTypes
}
