package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ConfigScope selects whether SaveModelForStage writes to the user-wide
// global config (~/.config/CommitCraft/config.toml) or the per-repo
// .commitcraft.toml in the working directory.
type ConfigScope int

const (
	ScopeGlobal ConfigScope = iota
	ScopeLocal
)

// ModelStage is the identifier of a configurable AI stage. The TOML
// table + key it maps to is encapsulated in stageMapping.
type ModelStage string

const (
	StageChangeAnalyzer ModelStage = "change_analyzer"
	StageCommitBody     ModelStage = "commit_body"
	StageCommitTitle    ModelStage = "commit_title"
	StageOnlyTranslate  ModelStage = "only_translate"
	StageRelease        ModelStage = "release"
	StageChangelog      ModelStage = "changelog"
)

type stageMapping struct {
	table string
	key   string
}

var stageMappings = map[ModelStage]stageMapping{
	StageChangeAnalyzer: {"prompts", "change_analyzer_prompt_model"},
	StageCommitBody:     {"prompts", "commit_body_generator_prompt_model"},
	StageCommitTitle:    {"prompts", "commit_title_generator_prompt_model"},
	StageOnlyTranslate:  {"prompts", "only_translate_prompt_model"},
	StageRelease:        {"prompts", "release_prompt_model"},
	StageChangelog:      {"changelog", "prompt_model"},
}

// SaveModelForStage rewrites the model id for the given stage in the
// chosen scope's TOML file. Reads the file as a generic table so unrelated
// keys survive untouched; for local scope the file is created on first
// use containing only the modified table.
func SaveModelForStage(stage ModelStage, modelID string, scope ConfigScope) error {
	mapping, ok := stageMappings[stage]
	if !ok {
		return fmt.Errorf("unknown stage %q", stage)
	}

	path, err := scopePath(scope)
	if err != nil {
		return err
	}

	data := map[string]any{}
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, &data); err != nil {
			return fmt.Errorf("decoding %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	table, _ := data[mapping.table].(map[string]any)
	if table == nil {
		table = map[string]any{}
	}
	table[mapping.key] = modelID
	data[mapping.table] = table

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(data); err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// ApplyModelToConfig mirrors SaveModelForStage in memory so the running
// Model can keep using the new value without re-running LoadConfigs.
func ApplyModelToConfig(cfg *Config, stage ModelStage, modelID string) {
	switch stage {
	case StageChangeAnalyzer:
		cfg.Prompts.ChangeAnalyzerPromptModel = modelID
	case StageCommitBody:
		cfg.Prompts.CommitBodyGeneratorPromptModel = modelID
	case StageCommitTitle:
		cfg.Prompts.CommitTitleGeneratorPromptModel = modelID
	case StageOnlyTranslate:
		cfg.Prompts.OnlyTranslatePromptModel = modelID
	case StageRelease:
		cfg.Prompts.ReleasePromptModel = modelID
	case StageChangelog:
		cfg.Changelog.PromptModel = modelID
	}
}

// CurrentModelForStage returns the model id currently assigned to the
// given stage in cfg, or "" when unset.
func CurrentModelForStage(cfg Config, stage ModelStage) string {
	switch stage {
	case StageChangeAnalyzer:
		return cfg.Prompts.ChangeAnalyzerPromptModel
	case StageCommitBody:
		return cfg.Prompts.CommitBodyGeneratorPromptModel
	case StageCommitTitle:
		return cfg.Prompts.CommitTitleGeneratorPromptModel
	case StageOnlyTranslate:
		return cfg.Prompts.OnlyTranslatePromptModel
	case StageRelease:
		return cfg.Prompts.ReleasePromptModel
	case StageChangelog:
		return cfg.Changelog.PromptModel
	}
	return ""
}

func scopePath(scope ConfigScope) (string, error) {
	switch scope {
	case ScopeGlobal:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("user home: %w", err)
		}
		return filepath.Join(home, GlobalConfigDir, globalConfigName), nil
	case ScopeLocal:
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getwd: %w", err)
		}
		return filepath.Join(wd, localConfigName), nil
	}
	return "", fmt.Errorf("unknown scope")
}
