package storage

import "time"

// Commit represents a single commit record in the database.
type Commit struct {
	ID        int
	Type      string
	Scope     string
	MessageES string
	MessageEN string
	Workspace string
	Diff_code string
	CreatedAt time.Time
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
