package aiengine

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"commit_craft_reborn/internal/changelog"
	"commit_craft_reborn/internal/config"
)

// Delegate mode turns the pipeline inside-out: instead of calling Groq, the
// CLI fills the same prompts and hands them to the already-running AI agent,
// which produces the message itself and returns it through `commitcraft ai
// submit`. This file builds those prompt bundles. No Groq call ever happens
// here — these are pure, deterministic string builders.

// DelegateStage is one filled prompt the agent should act on: a system prompt
// plus the user-content it applies to. In "single" strategy a bundle carries
// one DelegateStage (Unified); in "staged" strategy it carries the original
// per-stage prompts.
type DelegateStage struct {
	Stage  string `json:"stage"`
	System string `json:"system"`
	User   string `json:"user"`
}

// BundleInputs echoes the raw inputs back to the agent so it can copy them
// verbatim into the `ai submit` payload without re-deriving anything.
type BundleInputs struct {
	Tag             string   `json:"tag,omitempty"`
	Scope           string   `json:"scope,omitempty"`
	KeyPoints       []string `json:"keypoints,omitempty"`
	ChangelogActive bool     `json:"changelog_active"`
	Branch          string   `json:"branch,omitempty"`
	Version         string   `json:"version,omitempty"`
	// CommitList is the canonical, storage-ready serialization of the commit
	// range (release bundles only). The agent copies it verbatim into the
	// `ai submit` payload so the persisted release keeps its traceability
	// without re-deriving the range.
	CommitList string `json:"commit_list,omitempty"`
}

// DelegateBundle is the JSON envelope emitted by `generate/regenerate/merge/
// release --agent`. Mode is always "delegate"; Kind is "commit" or "release";
// Action is "generate" | "regenerate" | "merge" | "release". Exactly one of
// Unified / Stages is populated per Strategy.
type DelegateBundle struct {
	Mode          string          `json:"mode"`
	Kind          string          `json:"kind"`
	Action        string          `json:"action"`
	Strategy      string          `json:"strategy"`
	ID            int             `json:"id,omitempty"`
	Inputs        BundleInputs    `json:"inputs"`
	Unified       *DelegateStage  `json:"unified,omitempty"`
	Stages        []DelegateStage `json:"stages,omitempty"`
	Instructions  string          `json:"instructions"`
	SubmitExample string          `json:"submit_example"`
}

const (
	commitInstructions = "CommitCraft delegate mode. Do NOT call any external " +
		"API — YOU produce the commit message. Use the prompt(s) below as your " +
		"system instructions and the staged diff as context. The commit title " +
		"AND body MUST be written in English regardless of the working language. " +
		"When you are done, submit the result by piping a JSON object to " +
		"`commitcraft ai submit` on stdin (see submit_example). Copy `tag`, " +
		"`scope`, and `keypoints` from `inputs` verbatim. Set `id` only for a " +
		"regenerate (it updates that draft). After submit, check the returned " +
		"`verify` block; fix issues with `ai edit` before `ai promote`."

	releaseInstructions = "CommitCraft delegate mode. Do NOT call any external " +
		"API — YOU produce the release note. Use the prompt(s) below as your " +
		"system instructions and the commit list as context. The title AND body " +
		"MUST be written in English. When done, pipe a JSON object to " +
		"`commitcraft ai submit` on stdin (see submit_example) with " +
		"`kind:\"release\"`. Copy `type`, `branch`/`version` from `inputs`. Set " +
		"`id` only to update an existing release draft."

	commitSubmitExample = `echo '{"kind":"commit","action":"generate",` +
		`"tag":"ADD","scope":["cli"],"keypoints":["..."],` +
		`"title":"add agent delegate mode","body":"...","summary":"...",` +
		`"changelog_entry":"...","changelog_mention":"- Updated CHANGELOG.md ..."}' ` +
		`| commitcraft ai submit`

	releaseSubmitExample = `echo '{"kind":"release","type":"MERGE",` +
		`"branch":"feat/foo","title":"...","body":"..."}' | commitcraft ai submit`
)

// BuildCommitBundle assembles the delegate bundle for the commit pipeline.
// strategy is "single" (unified prompt) or "staged" (per-stage prompts);
// action is "generate" or "regenerate"; id is the draft id for regenerate
// (0 for a fresh generate). No Groq call.
func BuildCommitBundle(deps Deps, in Input, strategy, action string, id int) DelegateBundle {
	strategy = config.NormalizeAgentStrategy(strategy)
	pc := deps.Cfg.Prompts
	developerPoints := stripMentions(strings.Join(in.KeyPoints, "\n"))

	clog := changelogContext(deps, in.ChangelogActive)

	b := DelegateBundle{
		Mode:     config.AgentModeDelegate,
		Kind:     "commit",
		Action:   action,
		Strategy: strategy,
		ID:       id,
		Inputs: BundleInputs{
			Tag:             in.Type,
			Scope:           in.Scope,
			KeyPoints:       in.KeyPoints,
			ChangelogActive: clog != "",
		},
		Instructions:  commitInstructions,
		SubmitExample: commitSubmitExample,
	}

	if strategy == config.AgentStrategyStaged {
		b.Stages = commitStages(pc, deps.Cfg.Changelog, in, developerPoints, clog)
		return b
	}

	user := fmt.Sprintf(
		"TAG:\n%s\nMODULE:\n%s\nDEVELOPER_POINTS:\n%s\nGIT_CHANGES:\n%s",
		in.Type, in.Scope, developerPoints, in.Diff,
	)
	if clog != "" {
		user += "\nCHANGELOG_CONTEXT:\n" + clog
	}
	b.Unified = &DelegateStage{
		Stage:  "commit",
		System: pc.AgentCommitPrompt,
		User:   user,
	}
	return b
}

// commitStages fills the original per-stage prompts. Stage inputs that depend
// on the agent's own upstream output carry an explicit placeholder note rather
// than a real value, since those values don't exist until the agent produces
// them.
func commitStages(
	pc config.PromptsConfig,
	clogCfg config.ChangelogConfig,
	in Input,
	developerPoints, clog string,
) []DelegateStage {
	stages := []DelegateStage{
		{
			Stage:  "summary",
			System: pc.ChangeAnalyzerPrompt,
			User: fmt.Sprintf(
				"DEVELOPER_POINTS:\n%s\nGIT_CHANGES:\n%s",
				developerPoints, in.Diff,
			),
		},
		{
			Stage:  "body",
			System: pc.CommitBodyGeneratorPrompt,
			User: fmt.Sprintf(
				"TAG:\n%s\nMODULE:\n%s\nSUMMARY_PARAGRAPHS:\n<your stage-1 (summary) output>",
				in.Type, in.Scope,
			),
		},
		{
			Stage:  "title",
			System: pc.CommitTitleGeneratorPrompt,
			User: fmt.Sprintf(
				"TAG:\n%s\nMODULE:\n%s\nCOMMIT_BODY:\n<your stage-2 (body) output>",
				in.Type, in.Scope,
			),
		},
	}
	if clog != "" {
		stages = append(stages, DelegateStage{
			Stage:  "changelog",
			System: clogCfg.Prompt,
			User: fmt.Sprintf(
				"%s\nSTAGE2_BODY:\n<your stage-2 (body) output>\nSTAGE3_TITLE:\n<your stage-3 (title) output>",
				clog,
			),
		})
	}
	return stages
}

// BuildReleaseBundle assembles the delegate bundle for the release pipeline
// (merge/release). releaseType is "MERGE" or "RELEASE". No Groq call.
func BuildReleaseBundle(
	deps Deps,
	in ReleaseInput,
	releaseType, strategy, action string,
	id int,
	branch, version, commitList string,
) DelegateBundle {
	strategy = config.NormalizeAgentStrategy(strategy)
	pc := deps.Cfg.Prompts
	commitsBlob := formatReleaseCommits(in.Commits)

	b := DelegateBundle{
		Mode:     config.AgentModeDelegate,
		Kind:     "release",
		Action:   action,
		Strategy: strategy,
		ID:       id,
		Inputs: BundleInputs{
			Tag:        fmt.Sprintf("[%s]", releaseType),
			Branch:     branch,
			Version:    version,
			CommitList: commitList,
		},
		Instructions:  releaseInstructions,
		SubmitExample: releaseSubmitExample,
	}

	if strategy == config.AgentStrategyStaged {
		b.Stages = []DelegateStage{
			{Stage: "release_body", System: pc.ReleaseBodyPrompt, User: commitsBlob},
			{
				Stage:  "release_title",
				System: pc.ReleaseTitlePrompt,
				User: fmt.Sprintf(
					"BODY:\n<your stage-1 (body) output>\n\nCOMMITS:\n%s",
					commitsBlob,
				),
			},
			{
				Stage:  "release_refine",
				System: pc.ReleaseRefinePrompt,
				User:   "TITLE:\n<your stage-2 (title) output>\n\nBODY:\n<your stage-1 (body) output>",
			},
		}
		return b
	}

	b.Unified = &DelegateStage{
		Stage:  "release",
		System: pc.AgentReleasePrompt,
		User: fmt.Sprintf(
			"TAG:\n[%s]\nSCOPE:\n%s\nCOMMITS:\n%s",
			releaseType,
			releaseScopeValue(branch, version),
			commitsBlob,
		),
	}
	return b
}

func releaseScopeValue(branch, version string) string {
	if branch != "" {
		return branch
	}
	return version
}

// changelogContext returns the filled CHANGELOG context block (FORMAT_SAMPLE,
// SUGGESTED_VERSION, DATE, BODY_BULLET_STYLE) when the changelog stage is
// active and a CHANGELOG exists, or "" otherwise. Best-effort: any detection
// failure yields "" so delegate mode degrades to title+body only.
func changelogContext(deps Deps, active bool) string {
	if !active || !deps.Cfg.Changelog.Enabled || deps.Cfg.Changelog.Prompt == "" {
		return ""
	}
	info, err := changelog.Detect(deps.Pwd, deps.Cfg.Changelog.Path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) && deps.Log != nil {
			deps.Log.Warn("delegate: changelog detect failed, skipping", "error", err)
		}
		return ""
	}
	suggested := changelog.SuggestNextVersion(info.LatestVersion, deps.Cfg.Changelog.BumpStrategy)
	return fmt.Sprintf(
		"FORMAT_SAMPLE:\n%s\nSUGGESTED_VERSION:\n%s\nDATE:\n%s\nBODY_BULLET_STYLE:\n-",
		info.FormatSample,
		suggested,
		time.Now().Format("2006-01-02"),
	)
}
