// Package ai implements the headless `commitcraft ai …` subcommands so
// an external agent can drive the same multi-stage pipeline that the
// TUI runs on Ctrl+W. All output is JSON on stdout (errors go to
// stderr) so the caller can pipe results into another tool.
package ai

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"commit_craft_reborn/internal/aiengine"
	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/commit"
	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/logger"
	"commit_craft_reborn/internal/storage"
)

const usage = `Usage: commitcraft ai <subcommand> [flags]

Subcommands:
  generate     Generate a commit message from --keypoint/--tag/--scope and persist as draft.
  regenerate   Re-run the pipeline on an existing draft (--id), reusing stored inputs and diff.
  show         Print the JSON for a draft/commit by --id.
  list         List drafts/commits in the current workspace.
  promote      Mark a draft as completed (--id). Does not run git commit.
  list-tags    List the resolved commit-type tags (default + global + local) as JSON.

Run 'commitcraft ai <subcommand> -h' for the flags of each subcommand.
`

// Dispatch is the entry point invoked from cmd/cli/main.go when the
// first positional arg is "ai". Returns the process exit code.
func Dispatch(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "generate":
		return runGenerate(rest)
	case "regenerate":
		return runRegenerate(rest)
	case "show":
		return runShow(rest)
	case "list":
		return runList(rest)
	case "promote":
		return runPromote(rest)
	case "list-tags":
		return runListTags(rest)
	case "-h", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n\n%s", sub, usage)
		return 2
	}
}

// stringSlice is a flag.Value that accumulates repeated --flag values
// into a slice, trimming whitespace and dropping empties.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	v = strings.TrimSpace(v)
	if v != "" {
		*s = append(*s, v)
	}
	return nil
}

type bootstrap struct {
	cfg              config.Config
	finalCommitTypes []commit.CommitType
	db               *storage.DB
	pwd              string
	log              *logger.Logger
}

// loadBootstrap reproduces the cmd/cli/main.go config + DB + pwd setup
// without any TUI side effects. Logger is the same charm-based one but
// since we never start a TUI it just emits to its ring buffer; the
// charm-log wrapped inside writes to stderr by default which is
// exactly what we want for diagnostics.
func loadBootstrap() (*bootstrap, error) {
	log := logger.New()
	globalCfg, localCfg, err := config.LoadConfigs()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	finalTypes := config.ResolveCommitTypes(globalCfg, localCfg)
	config.PopulateCommitTypePalettes(&globalCfg, finalTypes)
	config.ResolveReleaseConfig(&globalCfg, localCfg)
	config.ResolveTUIConfig(&globalCfg, localCfg)

	pwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getwd: %w", err)
	}
	db, err := storage.InitDB()
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}
	// Hydrate rate-limit cache so per-call accounting matches what the
	// TUI would see in the same situation.
	if persisted, err := db.LoadAllModelRateLimits(); err == nil {
		for _, p := range persisted {
			api.RecordRateLimits(p.ModelID, api.RateLimits{
				LimitRequests:     p.LimitRequests,
				RemainingRequests: p.RemainingRequests,
				LimitTokens:       p.LimitTokens,
				RemainingTokens:   p.RemainingTokens,
				CapturedAt:        p.CapturedAt,
				RequestsParsed:    p.RequestsParsed,
				TokensParsed:      p.TokensParsed,
				RequestsToday:     p.RequestsToday,
				RequestsDay:       p.RequestsDay,
			})
		}
	}
	return &bootstrap{
		cfg:              globalCfg,
		finalCommitTypes: finalTypes,
		db:               db,
		pwd:              pwd,
		log:              log,
	}, nil
}

func tagIsKnown(tag string, types []commit.CommitType) bool {
	for _, t := range types {
		if t.Tag == tag {
			return true
		}
	}
	return false
}

// commitJSON is the wire shape returned by every subcommand. Field
// names are snake_case for easy consumption from any language. Stages
// are flattened into a small array sorted by stage id.
type commitJSON struct {
	ID             int         `json:"id"`
	Status         string      `json:"status"`
	Type           string      `json:"type"`
	Scope          string      `json:"scope"`
	KeyPoints      []string    `json:"keypoints"`
	Summary        string      `json:"summary"`
	Body           string      `json:"body"`
	Title          string      `json:"title"`
	ChangelogEntry string      `json:"changelog_entry,omitempty"`
	ChangelogLine  string      `json:"changelog_mention,omitempty"`
	FinalMessage   string      `json:"final_message"`
	Workspace      string      `json:"workspace"`
	CreatedAt      string      `json:"created_at"`
	Stages         []stageJSON `json:"stages,omitempty"`
}

type stageJSON struct {
	ID               int    `json:"id"`
	Stage            string `json:"stage"`
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	TotalTimeMs      int    `json:"total_time_ms"`
	RequestID        string `json:"request_id,omitempty"`
}

var stageNames = [...]string{"summary", "body", "title", "changelog"}

func commitToJSON(c storage.Commit, stages []aiengine.StageStats) commitJSON {
	cj := commitJSON{
		ID:             c.ID,
		Status:         c.Status,
		Type:           c.Type,
		Scope:          c.Scope,
		KeyPoints:      c.KeyPoints,
		Summary:        c.IaSummary,
		Body:           c.IaCommitRaw,
		Title:          c.IaTitle,
		ChangelogEntry: c.IaChangelog,
		FinalMessage:   c.MessageEN,
		Workspace:      c.Workspace,
		CreatedAt:      c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	for i, s := range stages {
		if !s.HasStats {
			continue
		}
		name := ""
		if i >= 0 && i < len(stageNames) {
			name = stageNames[i]
		}
		cj.Stages = append(cj.Stages, stageJSON{
			ID:               i,
			Stage:            name,
			Model:            firstNonEmpty(s.StatsModel, s.Model),
			PromptTokens:     s.PromptTokens,
			CompletionTokens: s.CompletionTokens,
			TotalTokens:      s.TotalTokens,
			TotalTimeMs:      int(s.APITotalTime.Milliseconds()),
			RequestID:        s.RequestID,
		})
	}
	return cj
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// loadStagesForCommit rebuilds an aiengine.StageStats slice from the
// ai_calls rows persisted for a commit. Used by show/list so callers
// see the same per-stage telemetry the pipeline produced.
func loadStagesForCommit(db *storage.DB, commitID int) []aiengine.StageStats {
	calls, err := db.GetAICallsByCommitID(commitID)
	if err != nil {
		return nil
	}
	out := make([]aiengine.StageStats, 4)
	for i := range out {
		out[i].ID = aiengine.StageID(i)
	}
	for _, c := range calls {
		idx := dbStageIndex(c.Stage)
		if idx < 0 || idx >= len(out) {
			continue
		}
		out[idx].HasStats = true
		out[idx].StatsModel = c.Model
		out[idx].Model = c.Model
		out[idx].PromptTokens = c.PromptTokens
		out[idx].CompletionTokens = c.CompletionTokens
		out[idx].TotalTokens = c.TotalTokens
		out[idx].APITotalTime = time.Duration(c.TotalTimeMs) * time.Millisecond
		out[idx].RequestID = c.RequestID
		out[idx].TPMLimitAtCall = c.TPMLimitAtCall
	}
	return out
}

func dbStageIndex(label string) int {
	for i, n := range stageNames {
		if n == label {
			return i
		}
	}
	return -1
}

// persistAICalls flushes the per-stage telemetry produced by an engine
// run to the ai_calls table, replacing any existing rows for the given
// commit so iterative regenerations don't accumulate orphan data.
func persistAICalls(db *storage.DB, commitID int, stages []aiengine.StageStats) error {
	if commitID <= 0 || db == nil {
		return nil
	}
	if err := db.DeleteAICallsByCommitID(commitID); err != nil {
		return err
	}
	for i, s := range stages {
		if !s.HasStats {
			continue
		}
		stageName := ""
		if i >= 0 && i < len(stageNames) {
			stageName = stageNames[i]
		}
		modelName := s.StatsModel
		if modelName == "" {
			modelName = s.Model
		}
		_, err := db.CreateAICall(storage.AICall{
			CommitID:         commitID,
			Stage:            stageName,
			Model:            modelName,
			PromptTokens:     s.PromptTokens,
			CompletionTokens: s.CompletionTokens,
			TotalTokens:      s.TotalTokens,
			QueueTimeMs:      int(s.QueueTime.Milliseconds()),
			PromptTimeMs:     int(s.PromptTime.Milliseconds()),
			CompletionTimeMs: int(s.CompletionTime.Milliseconds()),
			TotalTimeMs:      int(s.APITotalTime.Milliseconds()),
			RequestID:        s.RequestID,
			TPMLimitAtCall:   s.TPMLimitAtCall,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// printErrorJSON writes a structured error to stderr. The shell exit
// code is the caller's responsibility (usually 1 for runtime errors,
// 2 for usage errors).
func printErrorJSON(code, msg string) {
	enc := json.NewEncoder(os.Stderr)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]string{"error": msg, "code": code})
}

// printCommitJSON writes a commitJSON to stdout, indented for human
// readability — agents that prefer compact JSON can re-encode.
func printCommitJSON(c commitJSON) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(c)
}

// validateAndStageDiff fetches the current staged diff. Returns an
// "no_staged_diff" code when the repo has nothing staged so callers
// can present the right error.
func validateAndStageDiff(maxBytes int) (string, error) {
	diff, err := git.GetStagedDiffSummary(maxBytes)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "", errors.New("no staged changes — run `git add` before invoking ai generate")
	}
	return diff, nil
}

// flagSet builds a flag.FlagSet that prints to stderr and returns
// usage errors as plain Go errors for easier handling.
func flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}
