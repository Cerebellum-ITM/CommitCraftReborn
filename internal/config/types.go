package config

import "commit_craft_reborn/internal/commit"

type Config struct {
	CommitTypes  CommitTypesConfig  `toml:"commit_types"`
	CommitFormat CommitFormatConfig `toml:"commit_format"`
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
		}
	}
	cfg.CommitTypes.Behavior = "replace"

	return cfg
}
