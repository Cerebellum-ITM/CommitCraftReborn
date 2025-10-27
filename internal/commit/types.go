package commit

type CommitType struct {
	Tag         string
	Description string
	Color       string
}

func GetDefaultCommitTypes() []CommitType {
	return []CommitType{
		{
			Tag:         "IMP",
			Description: "Improvements to the implementation",
			Color:       "39",
		},
		{
			Tag:         "FIX",
			Description: "Bug fixes",
			Color:       "197",
		},
		{
			Tag:         "ADD",
			Description: "Feature additions",
			Color:       "47",
		},
		{
			Tag:         "REM",
			Description: "Feature removals",
			Color:       "202",
		},
		{
			Tag:         "REF",
			Description: "Code refactoring",
			Color:       "141",
		},

		{
			Tag:         "MOV",
			Description: "File moves or renames",
			Color:       "221",
		},
		{
			Tag:         "REL",
			Description: "Release-related commits",
			Color:       "81",
		},
		{
			Tag:         "WIP",
			Description: "Work in progress",
			Color:       "15",
		},
		{
			Tag:         "DOC",
			Description: "Documentation updates",
			Color:       "254",
		},
	}
}

func GetDefaultLocalCommitExamplesTypes() []CommitType {
	return []CommitType{
		{
			Tag:         "TEST",
			Description: "Adding, updating, or fixing tests",
			Color:       "#FF8C00",
		},
		{
			Tag:         "CHORE",
			Description: "Routine tasks, maintenance, or minor adjustments that don't affect code logic (e.g., build process, dependency updates, configuration changes)",
			Color:       "#B0C4DE",
		},
		{
			Tag:         "PERF",
			Description: "Performance improvements",
			Color:       "#7CFC00",
		},
		{
			Tag:         "STYLE",
			Description: "Code style changes (e.g., formatting, semicolons, whitespace), without changing logic",
			Color:       "#ADD8E6",
		},
		{
			Tag:         "CI",
			Description: "Changes to CI/CD configuration files and scripts",
			Color:       "#1E90FF",
		},
		{
			Tag:         "BUILD",
			Description: "Changes that affect the build system or external dependencies (e.g., gulp, broccli, npm)",
			Color:       "#9932CC",
		},
		{
			Tag:         "REVERT",
			Description: "Reverts a previous commit",
			Color:       "#DC143C",
		},
		{
			Tag:         "SEC",
			Description: "Security improvements or vulnerability fixes",
			Color:       "#8B0000",
		},
	}
}
