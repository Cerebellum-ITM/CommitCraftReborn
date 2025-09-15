package config

import "commit_craft_reborn/internal/commit"

type PromptsConfig struct {
	SummaryPromptFile        string `toml:"summary_prompt_file"`
	SummaryPromptModel       string `toml:"summary_prompt_model"`
	SummaryPromptMaxDiffsize int    `toml:"summary_prompt_max_diff_size"`
	SummaryPrompt            string `toml:"-"`
	CommitBuilderPromptFile  string `toml:"commit_builder_prompt_file"`
	CommitBuilderPromptModel string `toml:"commit_builder_prompt_model"`
	CommitBuilderPrompt      string `toml:"-"`
	OutputFormatPromptFile   string `toml:"outformat_prompt_file"`
	OutputFormatPromptModel  string `toml:"outformat_prompt_model"`
	OutputFormatPrompt       string `toml:"-"`
	OnlyTranslatePromptFile  string `toml:"only_translate_prompt_file"`
	OnlyTranslatePromptModel string `toml:"only_translate_prompt_model"`
	OnlyTranslatePrompt      string `toml:"-"`
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
	TypeFormat       string            `toml:"type_format"`
	CommitTypeColors map[string]string `toml:"-"` // Map of commit type tag to color
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
			SummaryPromptFile:        "prompts/summary.prompt",
			SummaryPromptModel:       "meta-llama/llama-4-scout-17b-16e-instruct",
			SummaryPromptMaxDiffsize: 80000,
			CommitBuilderPromptFile:  "prompts/commit_builder.prompt",
			CommitBuilderPromptModel: "llama-3.1-8b-instant",
			OutputFormatPromptFile:   "prompts/output_format.prompt",
			OutputFormatPromptModel:  "llama-3.1-8b-instant",
			OnlyTranslatePromptFile:   "prompts/only_translate.prompt",
			OnlyTranslatePromptModel:  "llama-3.1-8b-instant",
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
