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
//
// TPMLimitAtCall is the model's per-minute token budget at the moment the
// call was made (`x-ratelimit-limit-tokens`). Stored alongside the call
// so the per-stage TPM-consumption bar in the pipeline view stays stable
// across reloads even if Groq later changes the model's limit.
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
	TPMLimitAtCall   int
	CreatedAt        time.Time
}

// ModelRateLimits mirrors the latest `x-ratelimit-*` snapshot we have for
// a given Groq model. Persisted so the in-memory cache can be hydrated on
// startup (the bars in compose / picker would otherwise show "no data
// yet" for any model not called in the current session).
type ModelRateLimits struct {
	ModelID           string
	LimitRequests     int
	RemainingRequests int
	ResetRequestsMs   int
	LimitTokens       int
	RemainingTokens   int
	ResetTokensMs     int
	CapturedAt        time.Time
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
