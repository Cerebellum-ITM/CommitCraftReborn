package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"

	"commit_craft_reborn/internal/commit"
)

const (
	localConfigName  = ".commitcraft.toml"
	GlobalConfigDir  = ".config/CommitCraft"
	globalConfigName = "config.toml"
)

// NOTE The following comments tells embed which path to look in from the file path

//go:embed prompts/change_analyzer.prompt.tmpl
var defaultChangeAnalyzerPrompt string

//go:embed prompts/commit_body_generator.prompt.tmpl
var defaultCommitBodyGeneratorPrompt string

//go:embed prompts/commit_title_generator.prompt.tmpl
var defaultCommitTitleGeneratorPrompt string

//go:embed prompts/only_translate.prompt.tmpl
var defaultOnlyTranslateFormatPrompt string

//go:embed prompts/release.prompt.tmpl
var defaultReleaseFormatPrompt string

//go:embed prompts/changelog_refiner.prompt.tmpl
var defaultChangelogRefinerPrompt string

func PopulateCommitTypeColors(cfg *Config, commitTypes []commit.CommitType) {
	if cfg.CommitFormat.CommitTypeColors == nil {
		cfg.CommitFormat.CommitTypeColors = make(map[string]string)
	}
	for _, ct := range commitTypes {
		cfg.CommitFormat.CommitTypeColors[ct.Tag] = ct.Color
	}
}

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
		case "change_analyzer":
			defaultPromptContent = defaultChangeAnalyzerPrompt
		case "commit_body_generator":
			defaultPromptContent = defaultCommitBodyGeneratorPrompt
		case "commit_title_generator":
			defaultPromptContent = defaultCommitTitleGeneratorPrompt
		case "only_translate":
			defaultPromptContent = defaultOnlyTranslateFormatPrompt
		case "release":
			defaultPromptContent = defaultReleaseFormatPrompt
		case "changelog_refiner":
			defaultPromptContent = defaultChangelogRefinerPrompt
		}
		parentDir := filepath.Dir(fullPath)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			return "", fmt.Errorf("could not create prompts directory at %s: %w", parentDir, err)
		}

		if err := os.WriteFile(fullPath, []byte(defaultPromptContent), 0o644); err != nil {
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
	changeAnalyzerPrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.ChangeAnalyzerPromptFile,
	)
	if err != nil {
		return err
	}

	commitBodyGeneratorPrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.CommitBodyGeneratorPromptFile,
	)
	if err != nil {
		return err
	}

	commitTitleGeneratorPrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.CommitTitleGeneratorPromptFile,
	)
	if err != nil {
		return err
	}

	onlyTranslatePrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.OnlyTranslatePromptFile,
	)
	if err != nil {
		return err
	}

	releasePrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.ReleasePromptFIle,
	)
	if err != nil {
		return err
	}

	globalConfig.Prompts.ChangeAnalyzerPrompt = changeAnalyzerPrompt
	globalConfig.Prompts.CommitBodyGeneratorPrompt = commitBodyGeneratorPrompt
	globalConfig.Prompts.CommitTitleGeneratorPrompt = commitTitleGeneratorPrompt
	globalConfig.Prompts.OnlyTranslatePrompt = onlyTranslatePrompt
	globalConfig.Prompts.ReleasePrompt = releasePrompt

	if globalConfig.Changelog.PromptFile != "" {
		changelogPrompt, err := createOrLoadPromptFile(
			configDir,
			globalConfig.Changelog.PromptFile,
		)
		if err != nil {
			return err
		}
		globalConfig.Changelog.Prompt = changelogPrompt
	}
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
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
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

		if err := os.WriteFile(filePath, content, 0o644); err != nil {
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

func ResolveReleaseConfig(
	globalCfg *Config, localCfg Config,
) {
	globalCfg.ReleaseConfig = localCfg.ReleaseConfig
}

// ResolveTUIConfig merges the local TUI overrides on top of the global
// config. Only fields explicitly set in the local file override the global
// one — leaving zero values keeps the global default in place.
func ResolveTUIConfig(globalCfg *Config, localCfg Config) {
	if localCfg.TUI.Theme != "" {
		globalCfg.TUI.Theme = localCfg.TUI.Theme
	}
}
