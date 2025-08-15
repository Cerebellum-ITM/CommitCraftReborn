package commit

// GetCommitTypes returns the list of allowed commit types.
// Later, this could be read from a database or a config file.

type CommitType struct {
	Tag         string
	Description string
}

func GetDefaultCommitTypes() []CommitType {
	return []CommitType{
		{Tag: "IMP", Description: "Improvements to the implementation"},
		{Tag: "FIX", Description: "Bug fixes"},
		{Tag: "ADD", Description: "Feature additions"},
		{Tag: "REM", Description: "Feature removals"},
		{Tag: "REF", Description: "Code refactoring"},
		{Tag: "MOV", Description: "File moves or renames"},
		{Tag: "REL", Description: "Release-related commits"},
		{Tag: "WIP", Description: "Work in progress"},
		{Tag: "DOC", Description: "Documentation updates"},
	}
}
