package aiengine

import (
	"strings"
	"testing"
)

func TestVerifyFinalMessage_Clean(t *testing.T) {
	msg := "[ADD] ai: introduce verify subcommand\n\nDocumenta y aplica el chequeo determinístico de residuos AI sobre el final_message del draft."
	r := VerifyFinalMessage(msg)
	if r.HasErrors || r.HasWarnings {
		t.Fatalf("expected clean report, got %+v", r)
	}
	if len(r.Findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(r.Findings))
	}
}

func TestVerifyFinalMessage_AIResidue(t *testing.T) {
	msg := "[ADD] ai: nueva feature\n\nHere is the commit message for the change.\nDetalle real."
	r := VerifyFinalMessage(msg)
	if !r.HasErrors {
		t.Fatalf("expected HasErrors=true")
	}
	if !findRule(r, "ai_residue_phrase") {
		t.Fatalf("missing ai_residue_phrase finding, got %+v", r.Findings)
	}
}

func TestVerifyFinalMessage_TemplatePlaceholder(t *testing.T) {
	msg := "[ADD] ai: <title>\n\nCuerpo normal."
	r := VerifyFinalMessage(msg)
	if !findRule(r, "template_placeholder") {
		t.Fatalf("missing template_placeholder finding, got %+v", r.Findings)
	}
}

func TestVerifyFinalMessage_CodeFenceTitle(t *testing.T) {
	msg := "```\n\nCuerpo."
	r := VerifyFinalMessage(msg)
	if !findRule(r, "code_fence_wrapper") {
		t.Fatalf("missing code_fence_wrapper finding, got %+v", r.Findings)
	}
}

func TestVerifyFinalMessage_TitleMissingTag(t *testing.T) {
	msg := "feature: do thing\n\nbody"
	r := VerifyFinalMessage(msg)
	if !findRule(r, "title_format_missing_tag") {
		t.Fatalf("missing title_format_missing_tag finding")
	}
}

func TestVerifyFinalMessage_TitleMissingScope(t *testing.T) {
	msg := "[ADD] something without scope shape\n\nbody"
	r := VerifyFinalMessage(msg)
	if !findRule(r, "title_format_missing_scope") {
		t.Fatalf("missing title_format_missing_scope finding, got %+v", r.Findings)
	}
	if r.HasErrors {
		t.Fatalf("missing-scope alone should be a warning, got HasErrors=true")
	}
}

func TestVerifyFinalMessage_TitleTooLongSoft(t *testing.T) {
	long := "[ADD] ai: " + strings.Repeat("a", 80)
	r := VerifyFinalMessage(long + "\n\nbody")
	if !findRule(r, "title_too_long_soft") {
		t.Fatalf("missing title_too_long_soft finding")
	}
}

func TestVerifyFinalMessage_TitleTooLongHard(t *testing.T) {
	long := "[ADD] ai: " + strings.Repeat("a", 110)
	r := VerifyFinalMessage(long + "\n\nbody")
	if !findRule(r, "title_too_long_hard") {
		t.Fatalf("missing title_too_long_hard finding")
	}
}

func TestVerifyFinalMessage_EmptyTitle(t *testing.T) {
	r := VerifyFinalMessage("\n\nsolo body")
	if !findRule(r, "empty_title") {
		t.Fatalf("missing empty_title finding")
	}
}

func TestVerifyFinalMessage_TitleEqualsBody(t *testing.T) {
	msg := "[ADD] ai: same line\n\n[ADD] ai: same line"
	r := VerifyFinalMessage(msg)
	if !findRule(r, "title_equals_body") {
		t.Fatalf("missing title_equals_body finding")
	}
}

func TestVerifyFinalMessage_DuplicateLine(t *testing.T) {
	msg := "[ADD] ai: ok\n\nUpdated CHANGELOG.md\n\nOther stuff.\n\nUpdated CHANGELOG.md"
	r := VerifyFinalMessage(msg)
	if !findRule(r, "duplicate_line_in_body") {
		t.Fatalf("missing duplicate_line_in_body finding, got %+v", r.Findings)
	}
	if r.HasErrors {
		t.Fatalf("duplicate-line is a warning, should not flip HasErrors")
	}
}

func findRule(r VerifyReport, rule string) bool {
	for _, f := range r.Findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}
