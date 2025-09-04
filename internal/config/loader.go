package config

import (
	"bytes"
	"commit_craft_reborn/internal/commit"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

const (
	localConfigName  = ".commitcraft.toml"
	GlobalConfigDir  = ".config/CommitCraft"
	globalConfigName = "config.toml"
)

// NOTE The following comments tells embed which path to look in from the file path

//go:embed prompts/summary.prompt.tmpl
var defaultSummaryPrompt string

//go:embed prompts/commit_builder.prompt.tmpl
var defaultCommitBuilderPrompt string

//go:embed prompts/output_format.prompt.tmpl
var defaultOutputFormatPrompt string

// --- Prompt Resolution Logic ---
func createOrLoadPromptFile(configDir string, fullPath string) (string, error) {
	var defaultPromptContent string
	if !filepath.IsAbs(fullPath) {
		fullPath = filepath.Join(configDir, fullPath)
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		baseFileName := filepath.Base(fullPath)
		promptName := strings.TrimSuffix(baseFileName, filepath.Ext(baseFileName))
		switch promptName {
		case "summary":
			defaultPromptContent = defaultSummaryPrompt
		case "commit_builder":
			defaultPromptContent = defaultCommitBuilderPrompt
		case "output_format":
			defaultPromptContent = defaultOutputFormatPrompt
		}
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return "", fmt.Errorf("could not create prompts directory at %s: %w", parentDir, err)
		}

		if err := os.WriteFile(fullPath, []byte(defaultPromptContent), 0644); err != nil {
			return "", fmt.Errorf("could not write default prompt file to %s: %w", fullPath, err)
		}

		return defaultPromptContent, nil
	} else if err != nil {
		return "", fmt.Errorf("error checking prompt file at %s: %w", fullPath, err)
	}
	promptBytes, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file at %s: %w", fullPath, err)
	}
	return string(promptBytes), nil
}

func loadIaPrompts(
	configDir string, globalConfig *Config,
) error {
	summaryPrompt, err := createOrLoadPromptFile(configDir, globalConfig.Prompts.SummaryPromptFile)
	if err != nil {
		return err
	}

	commitBuilderPrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.CommitBuilderPromptFile,
	)
	if err != nil {
		return err
	}

	outpurFormatPrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.OutputFormatPromptFile,
	)
	if err != nil {
		return err
	}

	globalConfig.Prompts.SummaryPrompt = summaryPrompt
	globalConfig.Prompts.CommitBuilderPrompt = commitBuilderPrompt
	globalConfig.Prompts.OutputFormatPrompt = outpurFormatPrompt
	return nil
}

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

	err = loadIaPrompts(globalDir, &globalCfg)
	if err != nil {
		return Config{}, Config{}, err
	}

	envPath := filepath.Join(globalDir, ".env")
	_ = godotenv.Load(envPath)

	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey != "" {
		globalCfg.TUI.GroqAPIKey = apiKey
		globalCfg.TUI.IsAPIKeySet = true
	} else {
		globalCfg.TUI.IsAPIKeySet = false
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
