package aiengine

import (
	"strings"
	"testing"
)

func TestVerifyFinalMessage_Clean(t *testing.T) {
	msg := "[ADD] ai: check drafts for AI residue offline\n\nRuns a deterministic rule set over the composed final_message so an\nagent can catch leaked phrases before promoting the draft."
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

func TestVerifyFinalMessage_TagWithDigits(t *testing.T) {
	r := VerifyFinalMessage(
		"[I18N] sale: ship the Spanish catalog\n\nThe module had no es_MX catalog, so every label showed in English.",
	)
	if r.HasErrors || r.HasWarnings {
		t.Fatalf("an I18N title must verify clean, got %+v", r.Findings)
	}
	if !findRule(
		VerifyFinalMessage("[I18N] sale: translate the catalog\n\nbody"),
		"title_restates_tag_verb",
	) {
		t.Fatalf("I18N titles must reach the tag-verb check")
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

func TestVerifyFinalMessage_GenericTitle_Flagged(t *testing.T) {
	for _, title := range []string{
		"[ADD] ai: update docs",
		"[FIX] context: fix bug",
		"[REF] storage: refactor code",
		"[ADD] ai: add feature",
	} {
		r := VerifyFinalMessage(title + "\n\nbody text here")
		if !findRule(r, "generic_title") {
			t.Errorf("expected generic_title warning for %q, got %+v", title, r.Findings)
		}
		if r.HasErrors {
			t.Errorf("generic_title should be warning only for %q", title)
		}
	}
}

func TestVerifyFinalMessage_GenericTitle_NotFlagged(t *testing.T) {
	for _, title := range []string{
		"[ADD] context: expose model context assessment",                    // 4 words
		"[ADD] ai: add --model flag to context command",                     // verb + many words
		"[UI] release_main_menu_list: release source pill indicators added", // no generic verb first
		"[FIX] storage: resolve collision in dispatch lookup",               // 5 words, specific
	} {
		r := VerifyFinalMessage(title + "\n\nbody text here")
		if findRule(r, "generic_title") {
			t.Errorf("unexpected generic_title warning for %q", title)
		}
	}
}

func TestVerifyFinalMessage_TitleTextTooLong(t *testing.T) {
	r := VerifyFinalMessage("[ADD] ai: " + strings.Repeat("a", 51) + "\n\nbody")
	if !findRule(r, "title_text_too_long") {
		t.Fatalf("missing title_text_too_long finding, got %+v", r.Findings)
	}
	r = VerifyFinalMessage("[ADD] ai: " + strings.Repeat("a", 50) + "\n\nbody")
	if findRule(r, "title_text_too_long") {
		t.Fatalf("50 characters must not be flagged")
	}
}

func TestVerifyFinalMessage_TitleRestatesTag(t *testing.T) {
	for _, title := range []string{
		"[ADD] docker: add remote ps over SSH with --from",
		"[FIX] mail: Fixes upload links in email messages",
		"[REM] api: drop deprecated v1 endpoints",
		"[DOC] deploy: document the Echo deploy flow",
	} {
		r := VerifyFinalMessage(title + "\n\nbody text here")
		if !findRule(r, "title_restates_tag_verb") {
			t.Errorf("expected title_restates_tag_verb for %q, got %+v", title, r.Findings)
		}
		if r.HasErrors {
			t.Errorf("title_restates_tag_verb must be a warning for %q", title)
		}
	}
	for _, title := range []string{
		"[ADD] docker: list containers on a linked host",
		"[FIX] contracts: route signers by stage, not by address",
		"[CHORE] importer: add the missing mailbox",
	} {
		if findRule(VerifyFinalMessage(title+"\n\nbody"), "title_restates_tag_verb") {
			t.Errorf("unexpected title_restates_tag_verb for %q", title)
		}
	}
}

func TestVerifyFinalMessage_BodyLineTooLong(t *testing.T) {
	long := strings.Repeat("word ", 20)
	r := VerifyFinalMessage("[ADD] ai: ok\n\nShort line.\n" + long)
	if !findRule(r, "body_line_too_long") {
		t.Fatalf("missing body_line_too_long finding, got %+v", r.Findings)
	}
	if r.HasErrors {
		t.Fatalf("body_line_too_long must be a warning")
	}
	url := "https://example.com/" + strings.Repeat("a", 80)
	if findRule(VerifyFinalMessage("[ADD] ai: ok\n\n"+url), "body_line_too_long") {
		t.Fatalf("an unbreakable token must not be flagged")
	}
}

func TestVerifyFinalMessage_BodyTooLong(t *testing.T) {
	body := strings.Repeat("A line of prose.\n", 15)
	if findRule(VerifyFinalMessage("[ADD] ai: ok\n\n"+body), "body_too_long") {
		t.Fatalf("15 lines must not be flagged")
	}
	paragraphs := strings.Repeat(
		"A line of prose.\n",
		8,
	) + "\n" + strings.Repeat(
		"A line of prose.\n",
		8,
	)
	r := VerifyFinalMessage("[ADD] ai: ok\n\n" + paragraphs)
	if !findRule(r, "body_too_long") {
		t.Fatalf("missing body_too_long for 16 lines, got %+v", r.Findings)
	}
	if r.HasErrors {
		t.Fatalf("body_too_long must be a warning")
	}
	if findRule(VerifyFinalMessage("[MERGE] feature/x: ship x\n\n"+paragraphs), "body_too_long") {
		t.Fatalf("merge notes must not be flagged")
	}
}

func TestVerifyFinalMessage_PlanningReference(t *testing.T) {
	for _, body := range []string{
		"The list now groups by origin, per decision R45.",
		"Implements the export from unit 8 of the plan.",
		"The spec runs with the batch of units 01 to 03.",
	} {
		r := VerifyFinalMessage("[ADD] app: group by origin\n\n" + body)
		if !findRule(r, "planning_reference") {
			t.Errorf("expected planning_reference for %q, got %+v", body, r.Findings)
		}
		if r.HasErrors {
			t.Errorf("planning_reference must be a warning for %q", body)
		}
	}
	for _, body := range []string{
		"Responses now carry an RFC and a 404 when the record is gone.",
		"Unit tests cover the parser.",
	} {
		if findRule(VerifyFinalMessage("[ADD] app: x\n\n"+body), "planning_reference") {
			t.Errorf("unexpected planning_reference for %q", body)
		}
	}
}

func TestVerifyFinalMessage_SessionNarrative(t *testing.T) {
	for _, body := range []string{
		"Verified against habitta_dev.",
		"The run took 22 s on the test Mac.",
		"The native suite passed 17/17 specs.",
		"Changed because the user asked for it.",
	} {
		if !findRule(VerifyFinalMessage("[FIX] app: x\n\n"+body), "session_narrative") {
			t.Errorf("expected session_narrative for %q", body)
		}
	}
	if findRule(
		VerifyFinalMessage("[ADD] app: x\n\nThe user picks a folder and the export lands there."),
		"session_narrative",
	) {
		t.Errorf("a product user must not be flagged")
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
