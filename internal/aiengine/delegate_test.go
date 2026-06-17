package aiengine

import (
	"strings"
	"testing"

	"commit_craft_reborn/internal/config"
)

func testDeps() Deps {
	cfg := config.Config{}
	cfg.Prompts.AgentCommitPrompt = "UNIFIED-COMMIT-PROMPT"
	cfg.Prompts.AgentReleasePrompt = "UNIFIED-RELEASE-PROMPT"
	cfg.Prompts.ChangeAnalyzerPrompt = "STAGE-SUMMARY"
	cfg.Prompts.CommitBodyGeneratorPrompt = "STAGE-BODY"
	cfg.Prompts.CommitTitleGeneratorPrompt = "STAGE-TITLE"
	cfg.Prompts.ReleaseBodyPrompt = "REL-BODY"
	cfg.Prompts.ReleaseTitlePrompt = "REL-TITLE"
	cfg.Prompts.ReleaseRefinePrompt = "REL-REFINE"
	return Deps{Cfg: cfg}
}

func sampleInput() Input {
	return Input{
		KeyPoints: []string{"add greet helper"},
		Type:      "ADD",
		Scope:     "core",
		Diff:      "diff --git a/main.go",
	}
}

func TestBuildCommitBundle_Single(t *testing.T) {
	b := BuildCommitBundle(testDeps(), sampleInput(), config.AgentStrategySingle, "generate", 0)

	if b.Mode != config.AgentModeDelegate {
		t.Fatalf("mode = %q, want delegate", b.Mode)
	}
	if b.Kind != "commit" || b.Action != "generate" {
		t.Fatalf("kind/action = %q/%q", b.Kind, b.Action)
	}
	if b.Strategy != config.AgentStrategySingle {
		t.Fatalf("strategy = %q", b.Strategy)
	}
	if b.Unified == nil {
		t.Fatal("single strategy must populate Unified")
	}
	if len(b.Stages) != 0 {
		t.Fatalf("single strategy must not populate Stages, got %d", len(b.Stages))
	}
	if b.Unified.System != "UNIFIED-COMMIT-PROMPT" {
		t.Fatalf("unified system = %q", b.Unified.System)
	}
	for _, want := range []string{"TAG:\nADD", "MODULE:\ncore", "DEVELOPER_POINTS:\nadd greet helper", "GIT_CHANGES:\ndiff --git"} {
		if !strings.Contains(b.Unified.User, want) {
			t.Fatalf("unified user missing %q in:\n%s", want, b.Unified.User)
		}
	}
	if b.Inputs.ChangelogActive {
		t.Fatal("changelog must be inactive when not enabled")
	}
	if strings.Contains(b.Unified.User, "CHANGELOG_CONTEXT") {
		t.Fatal("no CHANGELOG_CONTEXT expected when changelog disabled")
	}
}

func TestBuildCommitBundle_Staged(t *testing.T) {
	b := BuildCommitBundle(testDeps(), sampleInput(), config.AgentStrategyStaged, "generate", 0)

	if b.Unified != nil {
		t.Fatal("staged strategy must not populate Unified")
	}
	gotStages := make([]string, 0, len(b.Stages))
	for _, s := range b.Stages {
		gotStages = append(gotStages, s.Stage)
	}
	want := []string{"summary", "body", "title"}
	if strings.Join(gotStages, ",") != strings.Join(want, ",") {
		t.Fatalf("stages = %v, want %v", gotStages, want)
	}
	// Stage 1 carries the real diff; downstream stages carry placeholder notes.
	if !strings.Contains(b.Stages[0].User, "GIT_CHANGES:\ndiff --git") {
		t.Fatalf("summary stage missing diff: %s", b.Stages[0].User)
	}
	if !strings.Contains(b.Stages[1].User, "<your stage-1 (summary) output>") {
		t.Fatalf("body stage missing summary placeholder: %s", b.Stages[1].User)
	}
}

func TestBuildReleaseBundle(t *testing.T) {
	in := ReleaseInput{Commits: []ReleaseCommit{{Subject: "init", Date: "2026-01-01"}}}
	b := BuildReleaseBundle(
		testDeps(),
		in,
		"MERGE",
		config.AgentStrategySingle,
		"merge",
		0,
		"feat/foo",
		"",
		"abc init",
	)

	if b.Kind != "release" || b.Action != "merge" {
		t.Fatalf("kind/action = %q/%q", b.Kind, b.Action)
	}
	if b.Inputs.Tag != "[MERGE]" || b.Inputs.Branch != "feat/foo" {
		t.Fatalf("inputs tag/branch = %q/%q", b.Inputs.Tag, b.Inputs.Branch)
	}
	if b.Inputs.CommitList != "abc init" {
		t.Fatalf("commit_list = %q", b.Inputs.CommitList)
	}
	if b.Unified == nil || b.Unified.System != "UNIFIED-RELEASE-PROMPT" {
		t.Fatal("single release bundle must use the unified release prompt")
	}
}
