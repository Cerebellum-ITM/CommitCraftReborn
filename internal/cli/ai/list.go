package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// listEntry is a compact summary returned by `ai list`. Useful for
// agents that want to scan available drafts before picking one to
// inspect via `ai show --id`.
type listEntry struct {
	ID           int    `json:"id"`
	Status       string `json:"status"`
	Type         string `json:"type"`
	Scope        string `json:"scope"`
	TitleSnippet string `json:"title_snippet"`
	CreatedAt    string `json:"created_at"`
}

// runList prints a JSON array of drafts/commits in the current
// workspace, optionally filtered by status.
func runList(args []string) int {
	fs := flagSet("ai list")
	status := fs.String("status", "draft", "Filter by status (draft | completed)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}
	if *status != "draft" && *status != "completed" {
		printErrorJSON("invalid_input",
			fmt.Sprintf("--status must be draft or completed (got %q)", *status))
		return 2
	}

	bs, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer bs.db.Close()

	commits, err := bs.db.GetCommits(bs.pwd, *status)
	if err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}
	out := make([]listEntry, 0, len(commits))
	for _, c := range commits {
		title := strings.SplitN(strings.TrimSpace(c.IaTitle), "\n", 2)[0]
		if len(title) > 80 {
			title = title[:80] + "…"
		}
		out = append(out, listEntry{
			ID:           c.ID,
			Status:       c.Status,
			Type:         c.Type,
			Scope:        c.Scope,
			TitleSnippet: title,
			CreatedAt:    c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
	return 0
}
