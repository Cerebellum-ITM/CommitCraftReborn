package storage

import "time"

// Commit represents a single commit record in the database.
type Commit struct {
	ID          int
	Type        string
	Scope       string
	KeyPoints   []string
	MessageEN   string
	Workspace   string
	Diff_code   string
	Status      string
	IaSummary   string
	IaCommitRaw string
	IaTitle     string
	CreatedAt   time.Time
}

// AICall stores per-stage telemetry from a single Groq chat completion
// linked to a Commit row. Tokens come from the API's `usage` block; time
// fields are stored as integer milliseconds for compact storage and easy
// formatting (we never need sub-ms precision in the UI).
type AICall struct {
	ID               int
	CommitID         int
	Stage            string
	Model            string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	QueueTimeMs      int
	PromptTimeMs     int
	CompletionTimeMs int
	TotalTimeMs      int
	RequestID        string
	CreatedAt        time.Time
}

// representation of a release in the database
type Release struct {
	ID         int
	Type       string
	Title      string
	Body       string
	Branch     string
	CommitList string
	Version    string
	Workspace  string
	CreatedAt  time.Time
}
