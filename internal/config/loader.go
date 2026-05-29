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

// HasLocalConfig reports whether a .commitcraft.toml file exists in the
// given working directory. Used by the TUI to surface a status-bar pill
// signalling that project-local settings are overriding the global
// config. Errors other than "not exist" are treated as "not present"
// because the caller is purely informational.
func HasLocalConfig(pwd string) bool {
	_, err := os.Stat(filepath.Join(pwd, localConfigName))
	return err == nil
}

// NOTE The following comments tells embed which path to look in from the file path

//go:embed prompts/change_analyzer.prompt.tmpl
var defaultChangeAnalyzerPrompt string

//go:embed prompts/commit_body_generator.prompt.tmpl
var defaultCommitBodyGeneratorPrompt string

//go:embed prompts/commit_title_generator.prompt.tmpl
var defaultCommitTitleGeneratorPrompt string

//go:embed prompts/only_translate.prompt.tmpl
var defaultOnlyTranslateFormatPrompt string

//go:embed prompts/release_body.prompt.tmpl
var defaultReleaseBodyPrompt string

//go:embed prompts/release_title.prompt.tmpl
var defaultReleaseTitlePrompt string

//go:embed prompts/release_refine.prompt.tmpl
var defaultReleaseRefinePrompt string

//go:embed prompts/changelog_refiner.prompt.tmpl
var defaultChangelogRefinerPrompt string

// PopulateCommitTypePalettes builds the per-tag palette overlay from the
// resolved commit types. Only tags with at least one non-empty color slot
// are stored; tags with no overrides keep their built-in palette in the
// styles package. Tags are upper-cased so lookups match the renderer.
func PopulateCommitTypePalettes(cfg *Config, commitTypes []commit.CommitType) {
	if cfg.CommitFormat.CommitTypePalettes == nil {
		cfg.CommitFormat.CommitTypePalettes = make(map[string]CommitTypePalette)
	}
	for _, ct := range commitTypes {
		if ct.BgBlock == "" && ct.FgBlock == "" && ct.BgMsg == "" && ct.FgMsg == "" {
			continue
		}
		cfg.CommitFormat.CommitTypePalettes[strings.ToUpper(ct.Tag)] = CommitTypePalette{
			BgBlock: ct.BgBlock,
			FgBlock: ct.FgBlock,
			BgMsg:   ct.BgMsg,
			FgMsg:   ct.FgMsg,
		}
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
		case "release_body":
			defaultPromptContent = defaultReleaseBodyPrompt
		case "release_title":
			defaultPromptContent = defaultReleaseTitlePrompt
		case "release_refine":
			defaultPromptContent = defaultReleaseRefinePrompt
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

	releaseBodyPrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.ReleaseBodyPromptFile,
	)
	if err != nil {
		return err
	}

	releaseTitlePrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.ReleaseTitlePromptFile,
	)
	if err != nil {
		return err
	}

	releaseRefinePrompt, err := createOrLoadPromptFile(
		configDir,
		globalConfig.Prompts.ReleaseRefinePromptFile,
	)
	if err != nil {
		return err
	}

	globalConfig.Prompts.ChangeAnalyzerPrompt = changeAnalyzerPrompt
	globalConfig.Prompts.CommitBodyGeneratorPrompt = commitBodyGeneratorPrompt
	globalConfig.Prompts.CommitTitleGeneratorPrompt = commitTitleGeneratorPrompt
	globalConfig.Prompts.OnlyTranslatePrompt = onlyTranslatePrompt
	globalConfig.Prompts.ReleaseBodyPrompt = releaseBodyPrompt
	globalConfig.Prompts.ReleaseTitlePrompt = releaseTitlePrompt
	globalConfig.Prompts.ReleaseRefinePrompt = releaseRefinePrompt

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

	// Two-slot Groq key resolution. GROQ_ACTIVE_KEY selects which slot is
	// live; the active slot's key becomes GroqAPIKey. No silent fallback:
	// if the active slot is empty, GroqAPIKey stays empty and the usual
	// "API key not provided" error fires — `commitcraft ai key show`
	// surfaces the state. Absent/unknown GROQ_ACTIVE_KEY ⇒ user slot, so
	// existing single-key setups behave exactly as before.
	userKey := os.Getenv(EnvGroqUserKey)
	aiKey := os.Getenv(EnvGroqAIKey)
	active := NormalizeKeySlot(os.Getenv(EnvGroqActive))
	chosen := userKey
	if active == KeySlotAI {
		chosen = aiKey
	}
	globalCfg.TUI.ActiveKeySlot = active
	globalCfg.TUI.UserKeySet = userKey != ""
	globalCfg.TUI.AIKeySet = aiKey != ""
	globalCfg.TUI.GroqAPIKey = chosen
	globalCfg.TUI.IsAPIKeySet = chosen != ""

	// GH_TOKEN was previously persisted as a field inside each
	// .commitcraft.toml. Per-repo configs were ending up committed to
	// open-source repos with a live token inside. Moved to ~/.config/CommitCraft/.env
	// (joined with GROQ_API_KEY) and on first read the legacy values
	// in both the global and local TOMLs are migrated into the env
	// file and stripped from disk so they never leak again.
	if token := os.Getenv("GH_TOKEN"); token != "" {
		globalCfg.ReleaseConfig.GhToken = token
		globalCfg.ReleaseConfig.IsGhTokenSet = true
	} else {
		migrated := migrateLegacyGhToken(globalPath, envPath)
		if migrated == "" {
			migrated = migrateLegacyGhToken(localConfigName, envPath)
		}
		if migrated != "" {
			os.Setenv("GH_TOKEN", migrated)
			globalCfg.ReleaseConfig.GhToken = migrated
			globalCfg.ReleaseConfig.IsGhTokenSet = true
		}
	}

	return globalCfg, localCfg, nil
}

// migrateLegacyGhToken scans `tomlPath` for a single `GH_TOKEN = "..."`
// line. If present, the token is written into `envPath` (preserving any
// other env vars already there) and the line is removed from the TOML
// file. Returns the migrated token, or empty string if nothing was
// found / migration failed. Best-effort: I/O errors are silently
// swallowed so a half-broken file never blocks startup.
func migrateLegacyGhToken(tomlPath, envPath string) string {
	raw, err := os.ReadFile(tomlPath)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(raw), "\n")
	var token string
	out := lines[:0]
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "GH_TOKEN") {
			// Accept GH_TOKEN = "value", GH_TOKEN="value", GH_TOKEN = value
			if eq := strings.IndexByte(trim, '='); eq > 0 {
				v := strings.TrimSpace(trim[eq+1:])
				v = strings.Trim(v, `"' `)
				if v != "" && v != "ghp_123456789dummytoken" {
					token = v
				}
				continue
			}
		}
		out = append(out, line)
	}
	if token == "" {
		return ""
	}
	if err := appendEnvVar(envPath, "GH_TOKEN", token); err != nil {
		return ""
	}
	if err := os.WriteFile(tomlPath, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return ""
	}
	return token
}

// appendEnvVar upserts a KEY=VALUE pair inside envPath, creating the
// file at 0o600 if missing. Same upsert semantics as the TUI's
// `saveEnvVar` (config can't import tui, so we keep a small duplicate
// here just for the migration).
func appendEnvVar(envPath, name, value string) error {
	if err := os.MkdirAll(filepath.Dir(envPath), 0o755); err != nil {
		return err
	}
	existing := map[string]string{}
	order := []string{}
	if raw, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(raw), "\n") {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			eq := strings.IndexByte(line, '=')
			if eq <= 0 {
				continue
			}
			k := strings.TrimSpace(line[:eq])
			v := line[eq+1:]
			if _, seen := existing[k]; !seen {
				order = append(order, k)
			}
			existing[k] = v
		}
	}
	if _, ok := existing[name]; !ok {
		order = append(order, name)
	}
	existing[name] = value

	var buf bytes.Buffer
	for _, k := range order {
		fmt.Fprintf(&buf, "%s=%s\n", k, existing[k])
	}
	return os.WriteFile(envPath, buf.Bytes(), 0o600)
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

		header := "# CommitCraft Global Configuration\n" +
			"# This file was auto-generated. You can customize it.\n" +
			"#\n" +
			"# Custom commit-type colors\n" +
			"# -------------------------\n" +
			"# Each [[commit_types.types]] entry accepts four optional hex\n" +
			"# colors that drive how the tag renders in the TUI:\n" +
			"#   bg_block / fg_block  -> the chip pill (history, popup, pills)\n" +
			"#   bg_msg   / fg_msg    -> the commit message row background/text\n" +
			"# Empty values fall through to the built-in palette (or to the\n" +
			"# active theme's neutral colors if the tag has no built-in entry).\n" +
			"# Hex values must start with '#' (e.g. #264653); other formats are\n" +
			"# ignored with a warning on startup.\n" +
			"#\n" +
			"# Example:\n" +
			"#   [[commit_types.types]]\n" +
			"#   tag         = \"EXP\"\n" +
			"#   description = \"Experimental work\"\n" +
			"#   bg_block    = \"#264653\"\n" +
			"#   fg_block    = \"#e9f5db\"\n" +
			"#   bg_msg      = \"#1b2f37\"\n" +
			"#   fg_msg      = \"#a8c5b3\"\n\n"
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
			BgBlock:     ct.BgBlock,
			FgBlock:     ct.FgBlock,
			BgMsg:       ct.BgMsg,
			FgMsg:       ct.FgMsg,
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
			BgBlock:     ct.BgBlock,
			FgBlock:     ct.FgBlock,
			BgMsg:       ct.BgMsg,
			FgMsg:       ct.FgMsg,
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
	rc := &globalCfg.ReleaseConfig
	if !rc.AutoBuild {
		return
	}
	if rc.BuildTool == "" {
		rc.BuildTool = "make"
	}
	if rc.BuildTool != "make" {
		fmt.Fprintf(
			os.Stderr,
			"warning: release_config.build_tool=%q is not supported (only \"make\"); disabling auto_build\n",
			rc.BuildTool,
		)
		rc.AutoBuild = false
		return
	}
	if rc.BuildTarget == "" {
		fmt.Fprintln(
			os.Stderr,
			"warning: release_config.auto_build=true but build_target is empty; disabling auto_build",
		)
		rc.AutoBuild = false
	}
}

// ResolveTUIConfig merges the local TUI overrides on top of the global
// config. Only fields explicitly set in the local file override the global
// one — leaving zero values keeps the global default in place.
func ResolveTUIConfig(globalCfg *Config, localCfg Config) {
	if localCfg.TUI.Theme != "" {
		globalCfg.TUI.Theme = localCfg.TUI.Theme
	}
}
