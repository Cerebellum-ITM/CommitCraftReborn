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
