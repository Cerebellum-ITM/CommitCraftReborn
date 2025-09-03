package config

import "commit_craft_reborn/internal/commit"

type PromptsConfig struct {
	SummaryPromptFile string `toml:"summary_prompt_file"`
	SummaryPrompt string `toml:"-"`
}

type TUIConfig struct {
	UseNerdFonts bool   `toml:"use_nerd_fonts"`
	GroqAPIKey   string `toml:"-"`
	IsAPIKeySet  bool   `toml:"-"`
}

type Config struct {
	CommitTypes  CommitTypesConfig  `toml:"commit_types"`
	CommitFormat CommitFormatConfig `toml:"commit_format"`
	TUI          TUIConfig          `toml:"tui"`
	Prompts      PromptsConfig      `toml:"prompts"`
}

type CommitFormatConfig struct {
	TypeFormat string `toml:"type_format"`
}

type CommitTypesConfig struct {
	Behavior string             `toml:"behavior"`
	Types    []CustomCommitType `toml:"types"`
}

type CustomCommitType struct {
	Tag         string `toml:"tag"`
	Description string `toml:"description"`
	Color       string
}

func NewDefaultConfig() Config {
	return Config{
		CommitFormat: CommitFormatConfig{
			TypeFormat: "[%s]",
		},

		CommitTypes: CommitTypesConfig{
			Behavior: "append",
			Types:    []CustomCommitType{},
		},
		TUI: TUIConfig{
			UseNerdFonts: true,
		},
		Prompts: PromptsConfig{
			SummaryPromptFile: "prompts/summary.prompt",
		},
	}
}

func GetDefaultConfigWithTypes() Config {
	cfg := NewDefaultConfig()
	defaultCommits := commit.GetDefaultCommitTypes()

	cfg.CommitTypes.Types = make([]CustomCommitType, len(defaultCommits))
	for i, dc := range defaultCommits {
		cfg.CommitTypes.Types[i] = CustomCommitType{
			Tag:         dc.Tag,
			Description: dc.Description,
			Color:       dc.Color,
		}
	}
	cfg.CommitTypes.Behavior = "replace"

	return cfg
}
