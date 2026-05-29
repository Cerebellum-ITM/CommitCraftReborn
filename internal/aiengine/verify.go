package aiengine

import (
	"regexp"
	"strings"
)

// VerifyFinding is a single rule violation found in a draft's
// final_message. Rule slugs are stable and machine-readable so callers
// can branch deterministically.
type VerifyFinding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"` // "error" | "warning"
	Message  string `json:"message"`
	Location string `json:"location,omitempty"` // "title" | "body" | "line:N"
}

// VerifyReport is the output of VerifyFinalMessage. HasErrors is true
// when at least one finding has severity="error"; HasWarnings is true
// when at least one warning is present. Both can be true at once.
type VerifyReport struct {
	HasErrors   bool            `json:"has_errors"`
	HasWarnings bool            `json:"has_warnings"`
	Findings    []VerifyFinding `json:"findings"`
}

const (
	severityError   = "error"
	severityWarning = "warning"
)

// aiResiduePhrases are known leakage strings — phrases the model
// sometimes lets slip into the commit message. The list is intentionally
// conservative: better to under-flag than to false-positive on real
// commit prose. Add new entries as we encounter them in the wild.
var aiResiduePhrases = []string{
	"here is the commit message",
	"here's the commit message",
	"here is a commit message",
	"here is your commit message",
	"i made the following changes",
	"as an ai",
	"as a language model",
	"paragraph 1",
	"paragraph 2",
	"paragraph 3",
	"summary paragraphs",
}

// templatePlaceholderPattern catches literal template scaffolding that
// survived into the final message. Both angle and curly variants of the
// common slot names — anchored on word boundaries so we don't false-flag
// the bare words "title" or "body" in legitimate prose.
var templatePlaceholderPattern = regexp.MustCompile(
	`(?i)[<{](title|body|keypoints|summary|tag|scope)[>}]`,
)

// codeFenceWrapperPattern detects a fenced block wrapping the whole
// title or body. We look for ``` at the start of the line followed by
// an optional language label.
var codeFenceWrapperPattern = regexp.MustCompile("^```[a-zA-Z0-9_-]*\\s*$")

// titleTagPattern enforces the project's `[TAG]` prefix on the first
// line. Tags are uppercase tokens in square brackets. The optional
// trailing `scope:` is checked separately so we can warn instead of
// erroring when only the scope is missing.
var titleTagPattern = regexp.MustCompile(`^\[[A-Z]+\]`)

// titleScopePattern checks for the full `[TAG] scope:` shape that
// FormatFinalMessage produces. Anything after the colon is the title
// body and is not constrained here.
var titleScopePattern = regexp.MustCompile(`^\[[A-Z]+\]\s+\S+:\s+\S`)

// titleTextPattern extracts the free-text portion of a well-formed title
// (everything after `[TAG] scope: `). Used by the generic-title check.
var titleTextPattern = regexp.MustCompile(`^\[[A-Z]+\]\s+\S+:\s+(.+)$`)

// genericTitleVerbs are action verbs that produce near-content-free titles
// when the rest of the title is only 1-2 words. Conservative list — prefer
// false negatives over false positives. Extend only when we encounter a new
// pattern in the wild.
var genericTitleVerbs = map[string]bool{
	"update": true, "add": true, "remove": true, "fix": true,
	"improve": true, "document": true, "refactor": true,
	"implement": true, "create": true, "change": true,
	"modify": true, "cleanup": true, "delete": true, "rename": true,
}

// VerifyFinalMessage runs the deterministic rule set against a
// composed final_message (the same text that would go into
// `git commit`). Rules never call Groq and never read the diff — they
// catch the kind of defects that show up in the message text itself.
func VerifyFinalMessage(finalMessage string) VerifyReport {
	title, body := splitTitleBody(finalMessage)

	var findings []VerifyFinding

	findings = appendIf(findings, checkEmptyTitle(title))
	findings = appendIf(findings, checkTitleFormat(title)...)
	findings = appendIf(findings, checkTitleLength(title)...)
	findings = appendIf(findings, checkGenericTitle(title))
	findings = appendIf(findings, checkEmptyBody(title, body))
	findings = appendIf(findings, checkTitleEqualsBody(title, body))
	findings = appendIf(findings, checkCodeFence(title, body)...)
	findings = appendIf(findings, checkAIResidue(title, body)...)
	findings = appendIf(findings, checkTemplatePlaceholders(title, body)...)
	findings = appendIf(findings, checkDuplicateLines(body)...)

	report := VerifyReport{Findings: findings}
	for _, f := range findings {
		switch f.Severity {
		case severityError:
			report.HasErrors = true
		case severityWarning:
			report.HasWarnings = true
		}
	}
	return report
}

// splitTitleBody returns the first line (title) and the remainder
// after the first blank line (body). Falls back to empty body when
// the message is a single line.
func splitTitleBody(msg string) (title, body string) {
	msg = strings.TrimRight(msg, "\n")
	parts := strings.SplitN(msg, "\n\n", 2)
	title = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		body = strings.TrimSpace(parts[1])
	}
	return title, body
}

func appendIf(dst []VerifyFinding, fs ...*VerifyFinding) []VerifyFinding {
	for _, f := range fs {
		if f != nil {
			dst = append(dst, *f)
		}
	}
	return dst
}

func checkEmptyTitle(title string) *VerifyFinding {
	if title != "" {
		return nil
	}
	return &VerifyFinding{
		Rule:     "empty_title",
		Severity: severityError,
		Message:  "Title (first line) is empty.",
		Location: "title",
	}
}

func checkTitleFormat(title string) []*VerifyFinding {
	if title == "" {
		return nil
	}
	var out []*VerifyFinding
	if !titleTagPattern.MatchString(title) {
		out = append(out, &VerifyFinding{
			Rule:     "title_format_missing_tag",
			Severity: severityError,
			Message:  "Title does not start with `[TAG]` (e.g. `[ADD] scope: ...`).",
			Location: "title",
		})
		return out
	}
	if !titleScopePattern.MatchString(title) {
		out = append(out, &VerifyFinding{
			Rule:     "title_format_missing_scope",
			Severity: severityWarning,
			Message:  "Title has `[TAG]` but not the full `[TAG] scope: ...` shape.",
			Location: "title",
		})
	}
	return out
}

func checkTitleLength(title string) []*VerifyFinding {
	n := len(title)
	switch {
	case n > 100:
		return []*VerifyFinding{{
			Rule:     "title_too_long_hard",
			Severity: severityError,
			Message:  "Title is longer than 100 characters; will be truncated by most git UIs.",
			Location: "title",
		}}
	case n > 72:
		return []*VerifyFinding{{
			Rule:     "title_too_long_soft",
			Severity: severityWarning,
			Message:  "Title is longer than 72 characters (GitHub convention).",
			Location: "title",
		}}
	}
	return nil
}

// checkGenericTitle warns when the title text (the portion after
// `[TAG] scope: `) is ≤ 3 words and starts with a generic action verb.
// Titles like "update docs" or "fix bug" carry near-zero information —
// the model likely ignored the keypoints.
func checkGenericTitle(title string) *VerifyFinding {
	m := titleTextPattern.FindStringSubmatch(title)
	if m == nil {
		return nil // malformed title already caught by titleFormat rule
	}
	words := strings.Fields(m[1])
	if len(words) > 3 || len(words) == 0 {
		return nil
	}
	if !genericTitleVerbs[strings.ToLower(words[0])] {
		return nil
	}
	return &VerifyFinding{
		Rule:     "generic_title",
		Severity: severityWarning,
		Message:  "Title text is likely too generic (\"" + m[1] + "\"). Add specifics about what changed.",
		Location: "title",
	}
}

func checkEmptyBody(title, body string) *VerifyFinding {
	if body != "" || title == "" {
		return nil
	}
	return &VerifyFinding{
		Rule:     "empty_body",
		Severity: severityWarning,
		Message:  "Body is empty; only the title line will be committed.",
		Location: "body",
	}
}

func checkTitleEqualsBody(title, body string) *VerifyFinding {
	if title == "" || body == "" {
		return nil
	}
	if strings.TrimSpace(title) != strings.TrimSpace(body) {
		return nil
	}
	return &VerifyFinding{
		Rule:     "title_equals_body",
		Severity: severityError,
		Message:  "Title and body are byte-equal after trim.",
		Location: "body",
	}
}

func checkCodeFence(title, body string) []*VerifyFinding {
	var out []*VerifyFinding
	if codeFenceWrapperPattern.MatchString(strings.TrimSpace(title)) {
		out = append(out, &VerifyFinding{
			Rule:     "code_fence_wrapper",
			Severity: severityError,
			Message:  "Title is a code fence (```), not a commit title.",
			Location: "title",
		})
	}
	for i, line := range strings.Split(body, "\n") {
		if codeFenceWrapperPattern.MatchString(strings.TrimSpace(line)) {
			out = append(out, &VerifyFinding{
				Rule:     "code_fence_wrapper",
				Severity: severityError,
				Message:  "Body contains a stray code-fence line (```).",
				Location: lineLoc(i),
			})
			break // one finding per message is enough
		}
	}
	return out
}

func checkAIResidue(title, body string) []*VerifyFinding {
	var out []*VerifyFinding
	titleLower := strings.ToLower(title)
	bodyLower := strings.ToLower(body)
	for _, phrase := range aiResiduePhrases {
		if strings.Contains(titleLower, phrase) {
			out = append(out, &VerifyFinding{
				Rule:     "ai_residue_phrase",
				Severity: severityError,
				Message:  "Title contains AI residue phrase: " + phrase,
				Location: "title",
			})
		}
		if strings.Contains(bodyLower, phrase) {
			out = append(out, &VerifyFinding{
				Rule:     "ai_residue_phrase",
				Severity: severityError,
				Message:  "Body contains AI residue phrase: " + phrase,
				Location: "body",
			})
		}
	}
	return out
}

func checkTemplatePlaceholders(title, body string) []*VerifyFinding {
	var out []*VerifyFinding
	if templatePlaceholderPattern.MatchString(title) {
		out = append(out, &VerifyFinding{
			Rule:     "template_placeholder",
			Severity: severityError,
			Message:  "Title contains a template placeholder (e.g. `<title>`, `{body}`).",
			Location: "title",
		})
	}
	if templatePlaceholderPattern.MatchString(body) {
		out = append(out, &VerifyFinding{
			Rule:     "template_placeholder",
			Severity: severityError,
			Message:  "Body contains a template placeholder (e.g. `<title>`, `{body}`).",
			Location: "body",
		})
	}
	return out
}

// checkDuplicateLines warns when the same non-empty, non-separator line
// appears 2+ times in the body. Trailers and `---` rulers are ignored
// so we don't false-flag legitimate structure.
func checkDuplicateLines(body string) []*VerifyFinding {
	counts := map[string]int{}
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line == "---" {
			continue
		}
		counts[line]++
	}
	var out []*VerifyFinding
	for line, n := range counts {
		if n < 2 {
			continue
		}
		out = append(out, &VerifyFinding{
			Rule:     "duplicate_line_in_body",
			Severity: severityWarning,
			Message:  "Line repeated " + itoa(n) + "× in body: " + truncate(line, 80),
			Location: "body",
		})
	}
	return out
}

func lineLoc(zeroBased int) string {
	return "line:" + itoa(zeroBased+1)
}

// itoa avoids dragging strconv just for two call sites.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
