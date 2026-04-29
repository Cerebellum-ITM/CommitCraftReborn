package config

import (
	"commit_craft_reborn/internal/commit"
)

type PromptsConfig struct {
	ChangeAnalyzerPromptFile        string `toml:"change_analyzer_prompt_file"`
	ChangeAnalyzerPromptModel       string `toml:"change_analyzer_prompt_model"`
	ChangeAnalyzerMaxDiffSize       int    `toml:"change_analyzer_max_diff_size"`
	ChangeAnalyzerPrompt            string `toml:"-"`
	CommitBodyGeneratorPromptFile   string `toml:"commit_body_generator_prompt_file"`
	CommitBodyGeneratorPromptModel  string `toml:"commit_body_generator_prompt_model"`
	CommitBodyGeneratorPrompt       string `toml:"-"`
	CommitTitleGeneratorPromptFile  string `toml:"commit_title_generator_prompt_file"`
	CommitTitleGeneratorPromptModel string `toml:"commit_title_generator_prompt_model"`
	CommitTitleGeneratorPrompt      string `toml:"-"`
	OnlyTranslatePromptFile         string `toml:"only_translate_prompt_file"`
	OnlyTranslatePromptModel        string `toml:"only_translate_prompt_model"`
	OnlyTranslatePrompt             string `toml:"-"`
	ReleasePromptFIle               string `toml:"release_prompt_file"`
	ReleasePromptModel              string `toml:"release_prompt_model"`
	ReleasePrompt                   string `toml:"-"`
}

type TUIConfig struct {
	UseNerdFonts bool                 `toml:"use_nerd_fonts"`
	Theme        string               `toml:"theme,omitempty"`
	GroqAPIKey   string               `toml:"-"`
	IsAPIKeySet  bool                 `toml:"-"`
	Pipeline     PipelineLayoutConfig `toml:"pipeline,omitempty"`
}

// PipelineLayoutConfig controls the per-stage card heights on the
// Pipeline tab. Heights count *body rows* (the inner content between
// borders); the panel chrome adds 2 more rows on top.
type PipelineLayoutConfig struct {
	StageDefaultHeight int `toml:"stage_default_height"`
	StageFocusedHeight int `toml:"stage_focused_height"`
	DiffMinHeight      int `toml:"diff_min_height"`
}

type ReleaseConfig struct {
	Version          string `toml:"version"`
	GhToken          string `toml:"GH_TOKEN"`
	Repository       string `toml:"repository"`
	BinaryAssetsPath string `toml:"binary_assets_path"`

	AutoBuild   bool   `toml:"auto_build"`
	BuildTool   string `toml:"build_tool"`
	BuildTarget string `toml:"build_target"`
}

// ChangelogConfig drives the optional post-pipeline step that detects the
// repository's CHANGELOG format, asks the AI to produce a matching new entry,
// and writes/stages it together with the commit. Off by default — opt-in.
type ChangelogConfig struct {
	Enabled      bool   `toml:"enabled"`
	Path         string `toml:"path"`
	BumpStrategy string `toml:"bump_strategy"`
	PromptFile   string `toml:"prompt_file"`
	PromptModel  string `toml:"prompt_model"`
	Prompt       string `toml:"-"`
}

type Config struct {
	CommitTypes   CommitTypesConfig  `toml:"commit_types"`
	CommitFormat  CommitFormatConfig `toml:"commit_format"`
	TUI           TUIConfig          `toml:"tui,omitempty"`
	Prompts       PromptsConfig      `toml:"prompts,omitempty"`
	ReleaseConfig ReleaseConfig      `toml:"release_config,omitempty"`
	Changelog     ChangelogConfig    `toml:"changelog,omitempty"`
}

type CommitFormatConfig struct {
	TypeFormat string `toml:"type_format"`
	// CommitTypePalettes holds the per-tag four-color overrides resolved
	// from `[[commit_types.types]]`. Populated at startup by
	// `PopulateCommitTypePalettes` and forwarded to the styles package
	// so the renderer's chip + message colors honor the user's TOML.
	CommitTypePalettes map[string]CommitTypePalette `toml:"-"`
}

// CommitTypePalette is the wire-format mirror of `styles.CommitTypeColors`:
// raw hex strings for the chip background/foreground (`bg_block`/`fg_block`)
// and the message-row background/foreground (`bg_msg`/`fg_msg`). Kept here
// (instead of in `styles`) to avoid pulling the TUI package into config and
// creating an import cycle. Empty fields are skipped at registration time.
type CommitTypePalette struct {
	BgBlock string
	FgBlock string
	BgMsg   string
	FgMsg   string
}

type CommitTypesConfig struct {
	Behavior string             `toml:"behavior"`
	Types    []CustomCommitType `toml:"types"`
}

type CustomCommitType struct {
	Tag         string `toml:"tag"`
	Description string `toml:"description"`
	BgBlock     string `toml:"bg_block"`
	FgBlock     string `toml:"fg_block"`
	BgMsg       string `toml:"bg_msg"`
	FgMsg       string `toml:"fg_msg"`
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
			Pipeline: PipelineLayoutConfig{
				StageDefaultHeight: 4,
				StageFocusedHeight: 8,
				DiffMinHeight:      6,
			},
		},
		Prompts: PromptsConfig{
			ChangeAnalyzerPromptFile:        "prompts/change_analyzer.prompt",
			ChangeAnalyzerPromptModel:       "meta-llama/llama-4-scout-17b-16e-instruct",
			ChangeAnalyzerMaxDiffSize:       80000,
			CommitBodyGeneratorPromptFile:   "prompts/commit_body_generator.prompt",
			CommitBodyGeneratorPromptModel:  "llama-3.1-8b-instant",
			CommitTitleGeneratorPromptFile:  "prompts/commit_title_generator.prompt",
			CommitTitleGeneratorPromptModel: "llama-3.1-8b-instant",
			OnlyTranslatePromptFile:         "prompts/only_translate.prompt",
			OnlyTranslatePromptModel:        "llama-3.1-8b-instant",
		},
		Changelog: ChangelogConfig{
			Enabled:      false,
			Path:         "CHANGELOG.md",
			BumpStrategy: "patch",
			PromptFile:   "prompts/changelog_refiner.prompt",
			PromptModel:  "llama-3.1-8b-instant",
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
			BgBlock:     dc.BgBlock,
			FgBlock:     dc.FgBlock,
			BgMsg:       dc.BgMsg,
			FgMsg:       dc.FgMsg,
		}
	}
	cfg.CommitTypes.Behavior = "replace"

	return cfg
}

func GetDefaultLocalConfig() Config {
	cfg := NewDefautlLocalConfig()
	defaultCommits := commit.GetDefaultLocalCommitExamplesTypes()
	cfg.CommitTypes.Types = make([]CustomCommitType, len(defaultCommits))
	for i, dc := range defaultCommits {
		cfg.CommitTypes.Types[i] = CustomCommitType{
			Tag:         dc.Tag,
			Description: dc.Description,
			BgBlock:     dc.BgBlock,
			FgBlock:     dc.FgBlock,
			BgMsg:       dc.BgMsg,
			FgMsg:       dc.FgMsg,
		}
	}
	return cfg
}

func NewDefautlLocalConfig() Config {
	return Config{
		CommitFormat: CommitFormatConfig{
			TypeFormat: "[%s]",
		},

		CommitTypes: CommitTypesConfig{
			Behavior: "append",
			Types:    []CustomCommitType{},
		},
		ReleaseConfig: ReleaseConfig{
			Version:          "v0.2.5",
			GhToken:          "ghp_123456789dummytoken",
			Repository:       "user/repo_path",
			BinaryAssetsPath: "bin/",
			AutoBuild:        false,
			BuildTool:        "make",
			BuildTarget:      "",
		},
	}
}
