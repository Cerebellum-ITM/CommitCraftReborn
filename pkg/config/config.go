package config

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

// CommitType defines the structure for a single commit type in the config.
type CommitType struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	Color       string `toml:"color"`
}

// CommitTypesConfig holds the configuration related to commit types.
type CommitTypesConfig struct {
	Replace bool         `toml:"replace"`
	Types   []CommitType `toml:"types"`
}

// Config is the top-level configuration structure.
type Config struct {
	CommitTypes CommitTypesConfig `toml:"commit_types"`
}

// DefaultCommitTypes returns the default list of commit types.
func DefaultCommitTypes() []CommitType {
	return []CommitType{
		{Name: "[FIX]", Description: "Bug fixes"},
		{Name: "[REF]", Description: "Refactoring"},
		{Name: "[ADD]", Description: "Adding new modules/features"},
		{Name: "[REM]", Description: "Removing resources"},
		{Name: "[REV]", Description: "Reverting commits"},
		{Name: "[MOV]", Description: "Moving files or code"},
		{Name: "[REL]", Description: "Release commits"},
		{Name: "[IMP]", Description: "Incremental improvements"},
		{Name: "[MERGE]", Description: "Merge commits"},
		{Name: "[CLA]", Description: "Signing the Contributor License"},
		{Name: "[I18N]", Description: "Translation changes"},
		{Name: "[PERF]", Description: "Performance patches"},
		{Name: "[WIP]", Description: "Work in progress"},
	}
}

// LoadConfig loads configuration from default, global, and local sources.
func LoadConfig() (*Config, error) {
	// 1. Start with default config
	mergedConfig := &Config{
		CommitTypes: CommitTypesConfig{
			Types: DefaultCommitTypes(),
		},
	}

	// 2. Load and merge global config
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user home directory")
	}
	globalConfigPath := filepath.Join(home, ".commitcraft", "config.toml")
	globalConfig, err := loadFromFile(globalConfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load global config from %s", globalConfigPath)
	}
	if globalConfig != nil {
		mergedConfig = merge(mergedConfig, globalConfig)
	}

	// 3. Load and merge local config
	localConfigPath := ".commitcraft.toml"
	localConfig, err := loadFromFile(localConfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load local config from %s", localConfigPath)
	}
	if localConfig != nil {
		mergedConfig = merge(mergedConfig, localConfig)
	}


	// Sort final list alphabetically by name for consistency
	sort.Slice(mergedConfig.CommitTypes.Types, func(i, j int) bool {
		return mergedConfig.CommitTypes.Types[i].Name < mergedConfig.CommitTypes.Types[j].Name
	})

	return mergedConfig, nil
}

// loadFromFile reads a TOML config file from a given path.
func loadFromFile(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil // Return nil if file doesn't exist, not an error
	}

	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}


// merge combines a base config with an override config.
func merge(base, override *Config) *Config {
	// If override is nil or has no types, return base
	if override == nil || len(override.CommitTypes.Types) == 0 {
		return base
	}

	// If override has replace=true, it becomes the new base
	if override.CommitTypes.Replace {
		return override
	}

	// Otherwise, merge the types
	finalConfig := &Config{
		CommitTypes: CommitTypesConfig{
			// Start with a copy of base types
			Types: append([]CommitType(nil), base.CommitTypes.Types...),
		},
	}

	typeMap := make(map[string]int)
	for i, t := range finalConfig.CommitTypes.Types {
		typeMap[t.Name] = i
	}

	for _, t := range override.CommitTypes.Types {
		if idx, exists := typeMap[t.Name]; exists {
			// Update existing type
			finalConfig.CommitTypes.Types[idx] = t
		} else {
			// Add new type
			finalConfig.CommitTypes.Types = append(finalConfig.CommitTypes.Types, t)
		}
	}

	return finalConfig
}
