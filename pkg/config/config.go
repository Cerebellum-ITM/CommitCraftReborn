package config

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

// Struct definitions
type CommitType struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	Color       string `toml:"color"`
}
type CommitTypesConfig struct {
	Replace bool         `toml:"replace"`
	Types   []CommitType `toml:"types"`
}
type APIConfig struct {
	GroqKey string `toml:"groq_key,omitempty"`
}
type Config struct {
	CommitTypes CommitTypesConfig `toml:"commit_types"`
	API         APIConfig         `toml:"api"`
}

// LoadConfig loads configuration with a clear priority: defaults -> global -> local -> env vars.
func LoadConfig() (*Config, error) {
	// --- Part 1: Determine the final list of Commit Types ---
	finalTypes := DefaultCommitTypes()

	// Load global config
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user home directory")
	}
	globalPath := filepath.Join(home, ".commitcraft", "config.toml")
	globalCfg, err := loadFromFile(globalPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load global config from %s", globalPath)
	}

	if globalCfg != nil {
		if globalCfg.CommitTypes.Replace {
			finalTypes = globalCfg.CommitTypes.Types
		} else {
			finalTypes = mergeTypes(finalTypes, globalCfg.CommitTypes.Types)
		}
	}

	// Load local config
	localPath := ".commitcraft.toml"
	localCfg, err := loadFromFile(localPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load local config from %s", localPath)
	}

	if localCfg != nil {
		if localCfg.CommitTypes.Replace {
			finalTypes = localCfg.CommitTypes.Types
		} else {
			finalTypes = mergeTypes(finalTypes, localCfg.CommitTypes.Types)
		}
	}

	// --- Part 2: Determine the final API Key ---
	finalAPIKey := ""
	if globalCfg != nil {
		finalAPIKey = globalCfg.API.GroqKey // Key from global file
	}
	if envKey := os.Getenv("GROQ_API_KEY"); envKey != "" {
		finalAPIKey = envKey // Environment variable has highest priority
	}

	// --- Part 3: Assemble the final config ---
	finalConfig := &Config{
		CommitTypes: CommitTypesConfig{
			Types: finalTypes,
		},
		API: APIConfig{
			GroqKey: finalAPIKey,
		},
	}

	sort.Slice(finalConfig.CommitTypes.Types, func(i, j int) bool {
		return finalConfig.CommitTypes.Types[i].Name < finalConfig.CommitTypes.Types[j].Name
	})

	return finalConfig, nil
}

// mergeTypes merges override types into a base list of types.
func mergeTypes(base, override []CommitType) []CommitType {
	typeMap := make(map[string]int)
	for i, t := range base {
		typeMap[t.Name] = i
	}
	for _, t := range override {
		if idx, exists := typeMap[t.Name]; exists {
			base[idx] = t
		} else {
			base = append(base, t)
		}
	}
	return base
}

// loadFromFile reads a TOML config file from a given path.
func loadFromFile(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveGroqKey saves the Groq API key to the global config file.
func SaveGroqKey(key string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "failed to get user home directory")
	}
	configDir := filepath.Join(home, ".commitcraft")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create config directory")
	}
	globalConfigPath := filepath.Join(configDir, "config.toml")
	cfg, err := loadFromFile(globalConfigPath)
	if err != nil {
		return errors.Wrap(err, "failed to load global config for saving")
	}
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.API.GroqKey = key
	file, err := os.Create(globalConfigPath)
	if err != nil {
		return errors.Wrap(err, "failed to create/truncate global config file")
	}
	defer file.Close()
	if err := toml.NewEncoder(file).Encode(cfg); err != nil {
		return errors.Wrap(err, "failed to encode config to file")
	}
	return nil
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
