package commit

// GetCommitTypes returns the list of allowed commit types.
// Later, this could be read from a database or a config file.
func GetCommitTypes() []string {
	return []string{
		"[ADD] - For new features.",
		"[FIX] - For bug fixes.",
		"[IMP] - For improvements to existing features.",
		"[REF] - For code refactoring.",
		"[DOC] - For documentation changes.",
		"[REM] - For removing code or files.",
		"[MOV] - For moving files or code.",
	}
}
