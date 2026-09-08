package ai

import (
	"testing"

	"commit_craft_reborn/internal/storage"
)

func TestCommitToJSON_PendingDraftHasEmptyFinalMessage(t *testing.T) {
	c := storage.Commit{
		ID:        7,
		Type:      "ADD",
		Scope:     "ai",
		KeyPoints: []string{"tolerate pending drafts in ai show"},
		Status:    "draft",
		Source:    "agent",
	}
	cj, err := commitToJSON(c, nil, "[%s]")
	if err != nil {
		t.Fatalf("expected no error for a pending draft, got %v", err)
	}
	if cj.FinalMessage != "" {
		t.Fatalf("expected empty final_message, got %q", cj.FinalMessage)
	}
	if cj.Type != "ADD" || cj.Scope != "ai" || cj.Status != "draft" {
		t.Fatalf("envelope lost tag/scope/status: %+v", cj)
	}
	if len(cj.KeyPoints) != 1 {
		t.Fatalf("expected keypoints to survive, got %v", cj.KeyPoints)
	}
}

func TestCommitToJSON_CompleteDraftFormatsFinalMessage(t *testing.T) {
	c := storage.Commit{
		ID:        8,
		Type:      "ADD",
		Scope:     "ai",
		MessageEN: "tolerate pending drafts in ai show",
		Status:    "draft",
	}
	cj, err := commitToJSON(c, nil, "[%s]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "[ADD] ai: tolerate pending drafts in ai show"
	if cj.FinalMessage != want {
		t.Fatalf("final_message = %q, want %q", cj.FinalMessage, want)
	}
}

func TestCommitToJSON_MissingScopeStillErrors(t *testing.T) {
	c := storage.Commit{ID: 9, Type: "ADD", MessageEN: "no scope here"}
	if _, err := commitToJSON(c, nil, "[%s]"); err == nil {
		t.Fatal("expected an error when scope is missing")
	}
}
