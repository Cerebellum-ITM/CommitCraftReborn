# Unit 23: verify-generic-title

## Goal

Add a `generic_title` warning rule to `VerifyFinalMessage` that flags title
text like `"update docs"`, `"add feature"`, or `"fix bug"` — patterns where
a generic action verb + 1-2 vague words signals the model didn't anchor on
the keypoints and produced a near-content-free title.

Severity: **warning** (not error). A generic title is bad practice but
doesn't break `git commit`. The agent can decide to patch or accept.

## Design

The title text to check is the portion **after** the `[TAG] scope: ` prefix.
That text is extracted with `titleTextPattern` (added here). If the title
doesn't have the full `[TAG] scope: text` shape, `titleTextPattern` returns
no match and the rule is skipped (another rule already warned about the
malformed title).

The heuristic: title text has **≤ 3 words** AND the **first word** is in the
generic-verb list below.

**Why ≤ 3 words**: a 4-word title like `"add --model flag to ai context"` has
enough specificity to pass. `"update README"` (2 words) does not.

**Why first-word-only**: multi-word verb phrases (`"clean up"`, `"set up"`)
share the same leading verb. Checking only `words[0]` catches both without
needing phrase matching.

**Generic-verb list** (conservative — prefer false negatives over false positives):

```
update, add, remove, fix, improve, document, refactor,
implement, create, change, modify, cleanup, delete, rename
```

`clean` is intentionally absent (too likely to produce false positives in
legitimate titles like `"clean auth flow"`). Add to the list only when
we encounter a new pattern in the wild.

### Rule slug and location

- `rule`: `"generic_title"`
- `severity`: `"warning"`
- `location`: `"title"`

## Implementation

File: `internal/aiengine/verify.go`

1. Add `titleTextPattern` regexp (captures group 1 = text after `[TAG] scope: `):
   ```go
   var titleTextPattern = regexp.MustCompile(`^\[[A-Z]+\]\s+\S+:\s+(.+)$`)
   ```
2. Add `genericTitleVerbs` slice.
3. Add `checkGenericTitle(title string) *VerifyFinding`.
4. Call it in `VerifyFinalMessage` after `checkTitleLength`.

File: `internal/aiengine/verify_test.go`

Add two tests:
- `TestVerifyFinalMessage_GenericTitle_Flagged`: `[ADD] ai: update docs` → warning, rule `generic_title`.
- `TestVerifyFinalMessage_GenericTitle_NotFlagged`: `[ADD] ai: expose model context assessment` → no `generic_title` finding (4 words).

## Dependencies

- Unit 16 (`ai verify`) — already shipped; this unit adds one rule.

## Verify when done

- [ ] `go test ./internal/aiengine/...` passes (both new tests green).
- [ ] `go build ./...` + `go vet ./...` clean.
- [ ] `commitcraft ai verify --id <any_clean_draft>` still exits 0.
- [ ] Manually run `commitcraft ai verify` against a draft whose title is
      `[FIX] context: fix bug` → `generic_title` warning appears.
- [ ] Update `context/progress-tracker.md` to mark unit 23 complete.
