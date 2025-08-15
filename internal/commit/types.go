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
