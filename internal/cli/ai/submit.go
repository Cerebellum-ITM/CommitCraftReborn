package ai

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/storage"
)

// flexStrings unmarshals a JSON value that may be either a single string or an
// array of strings, so `scope`/`keypoints` accept both shapes from the agent.
type flexStrings []string

func (f *flexStrings) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "" {
		return nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var arr []string
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		out := arr[:0]
		for _, s := range arr {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		*f = out
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s = strings.TrimSpace(s); s != "" {
		*f = flexStrings{s}
	}
	return nil
}

// submitInput is the JSON payload `ai submit` reads from stdin (or
// --input-file). It carries the message the delegate agent produced plus the
// inputs needed to persist it as a draft.
type submitInput struct {
	Kind             string      `json:"kind"`   // "commit" (default) | "release"
	Action           string      `json:"action"` // informational
	ID               int         `json:"id"`     // >0 updates an existing draft
	Tag              string      `json:"tag"`    // commit type, e.g. "[ADD]"
	Type             string      `json:"type"`   // release type: "MERGE" | "RELEASE"
	Scope            flexStrings `json:"scope"`
	KeyPoints        flexStrings `json:"keypoints"`
	Title            string      `json:"title"`
	Body             string      `json:"body"`
	Summary          string      `json:"summary"`
	ChangelogEntry   string      `json:"changelog_entry"`
	ChangelogMention string      `json:"changelog_mention"`
	Branch           string      `json:"branch"`
	Version          string      `json:"version"`
	CommitList       string      `json:"commit_list"`
	Workspace        string      `json:"workspace"`
}

// runSubmit ingests an agent-produced commit/release message (delegate mode),
// persists it as a draft, runs the deterministic verifier, and prints the
// standard envelope with the verify report attached. Exit 0 on a successful
// persist regardless of verify findings — the draft is saved and recoverable;
// the agent reads `verify.has_errors` to decide whether to edit before
// promoting.
func runSubmit(args []string) int {
	fs := flagSet("ai submit")
	inputFile := fs.String(
		"input-file",
		"-",
		"Path to a JSON file with the submission payload. `-` (default) reads stdin.",
	)
	kindOverride := fs.String(
		"kind",
		"",
		"Override the payload's kind: 'commit' | 'release'. Optional.",
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}

	raw, err := readSubmitPayload(*inputFile)
	if err != nil {
		printErrorJSON("invalid_input", err.Error())
		return 2
	}
	var in submitInput
	if err := json.Unmarshal(raw, &in); err != nil {
		printErrorJSON("invalid_input", fmt.Sprintf("malformed JSON payload: %v", err))
		return 2
	}

	kind := strings.ToLower(strings.TrimSpace(in.Kind))
	if *kindOverride != "" {
		kind = strings.ToLower(strings.TrimSpace(*kindOverride))
	}
	if kind == "" {
		kind = kindCommit
	}

	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.Body) == "" {
		printErrorJSON("invalid_input", "both `title` and `body` are required")
		return 2
	}

	bs, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer bs.db.Close()

	switch kind {
	case kindCommit:
		return submitCommit(bs, in)
	case kindRelease:
		return submitRelease(bs, in)
	default:
		printErrorJSON("invalid_input", fmt.Sprintf("unknown kind %q (want commit|release)", kind))
		return 2
	}
}

func readSubmitPayload(path string) ([]byte, error) {
	if path == "" || path == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		if strings.TrimSpace(string(data)) == "" {
			return nil, errors.New("empty payload on stdin (pipe the submission JSON)")
		}
		return data, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return data, nil
}

// submitCommit persists an agent-produced commit message. id>0 updates the
// existing draft (regenerate path), preserving its stored diff; id==0 creates
// a fresh draft and snapshots the current staged diff.
func submitCommit(bs *bootstrap, in submitInput) int {
	var c storage.Commit
	if in.ID > 0 {
		existing, err := bs.db.GetCommitByID(in.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				printErrorJSON("not_found", fmt.Sprintf("commit with id=%d not found", in.ID))
				return 1
			}
			printErrorJSON("db_error", err.Error())
			return 1
		}
		c = existing
		if t := strings.TrimSpace(in.Tag); t != "" {
			c.Type = t
		}
		if len(in.Scope) > 0 {
			c.Scope = strings.Join(in.Scope, "\n")
		}
		if len(in.KeyPoints) > 0 {
			c.KeyPoints = in.KeyPoints
		}
	} else {
		if strings.TrimSpace(in.Tag) == "" {
			printErrorJSON("invalid_input", "`tag` is required for a new submission")
			return 2
		}
		if len(in.Scope) == 0 {
			printErrorJSON("invalid_input", "`scope` is required for a new submission")
			return 2
		}
		if len(in.KeyPoints) == 0 {
			printErrorJSON("invalid_input", "`keypoints` is required for a new submission")
			return 2
		}
		diff, err := validateAndStageDiff(bs.cfg.Prompts.ChangeAnalyzerMaxDiffSize)
		if err != nil {
			printErrorJSON("no_staged_diff", err.Error())
			return 1
		}
		ws := strings.TrimSpace(in.Workspace)
		if ws == "" {
			ws = bs.pwd
		}
		c = storage.Commit{
			Type:      strings.TrimSpace(in.Tag),
			Scope:     strings.Join(in.Scope, "\n"),
			KeyPoints: in.KeyPoints,
			Workspace: ws,
			Diff_code: diff,
			Source:    "ai",
		}
	}

	if !tagIsKnown(c.Type, bs.finalCommitTypes) {
		printErrorJSON(
			"invalid_input",
			fmt.Sprintf(
				"unknown tag %q — run `commitcraft ai list-tags` to see valid tags",
				c.Type,
			),
		)
		return 2
	}

	c.IaSummary = in.Summary
	c.IaCommitRaw = in.Body
	c.IaTitle = in.Title
	c.IaChangelog = in.ChangelogEntry
	c.MessageEN = aiengine.ComposeFinalMessage(
		in.Title,
		in.Body,
		strings.TrimSpace(in.ChangelogMention),
	)

	if err := bs.db.SaveDraft(&c); err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}

	saved, err := bs.db.GetCommitByID(c.ID)
	if err != nil {
		saved = c
		saved.Status = "draft"
	}
	cj, err := commitToJSON(saved, nil, bs.cfg.CommitFormat.TypeFormat)
	if err != nil {
		printErrorJSON("incomplete_commit", err.Error())
		return 1
	}

	final, ferr := commit.FormatFinalMessage(
		bs.cfg.CommitFormat.TypeFormat, saved.Type, saved.Scope, saved.MessageEN,
	)
	if ferr == nil {
		report := aiengine.VerifyFinalMessage(final)
		cj.Verify = &report
	}

	printCommitJSON(cj)
	return 0
}

// submitRelease persists an agent-produced [MERGE]/[RELEASE] note to the
// releases table. id>0 updates an existing release draft.
func submitRelease(bs *bootstrap, in submitInput) int {
	relType := strings.ToUpper(strings.TrimSpace(in.Type))
	if relType != "MERGE" && relType != "RELEASE" {
		printErrorJSON("invalid_input", "`type` must be MERGE or RELEASE for a release submission")
		return 2
	}
	if relType == "MERGE" && strings.TrimSpace(in.Branch) == "" {
		printErrorJSON("invalid_input", "`branch` is required for a MERGE submission")
		return 2
	}
	if relType == "RELEASE" && strings.TrimSpace(in.Version) == "" {
		printErrorJSON("invalid_input", "`version` is required for a RELEASE submission")
		return 2
	}

	ws := strings.TrimSpace(in.Workspace)
	if ws == "" {
		ws = bs.pwd
	}

	r := storage.Release{
		ID:         in.ID,
		Type:       relType,
		Title:      in.Title,
		Body:       in.Body,
		Branch:     strings.TrimSpace(in.Branch),
		Version:    strings.TrimSpace(in.Version),
		CommitList: in.CommitList,
		Workspace:  ws,
		Source:     "ai",
		Status:     "draft",
	}
	if err := bs.db.SaveReleaseDraft(&r); err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}

	saved, err := bs.db.GetReleaseByID(r.ID)
	if err != nil {
		saved = r
	}
	cj, err := releaseToJSON(saved, bs.cfg.CommitFormat.TypeFormat)
	if err != nil {
		printErrorJSON("format_error", err.Error())
		return 1
	}

	if final, ferr := composeReleaseFinalMessage(saved, bs.cfg.CommitFormat.TypeFormat); ferr == nil {
		report := aiengine.VerifyFinalMessage(final)
		cj.Verify = &report
	}

	printCommitJSON(cj)
	return 0
}
