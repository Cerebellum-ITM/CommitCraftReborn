# Unit 27: delegate-style-consistency

## Goal

Make agent-written commits (delegate mode, `single` strategy) read as if one
author wrote them. A sweep over the last 102 `source = 'ai'` drafts found the
style drifting between sessions and models:

| Symptom                                        | Count |
| ---------------------------------------------- | ----- |
| Body line over 72 columns                      | 29    |
| Title text over 50 characters                  | 12    |
| Title opens with the tag's verb (`[ADD] add`)  | 11    |
| Body is a bullet dump vs prose                 | 39/63 |
| Session narrative in body ("41 tests green")   | 2     |

Root cause: `agent_commit.prompt` (3 KB) is a lossy compression of the three
staged prompts (18 KB) and dropped the rules that produced consistency. The
verifier never measured the things the prompt asked for, so they were
followed by chance.

## Design

Two levers, both deterministic:

1. **Prompt.** Rewrite `agent_commit.prompt.tmpl` as a real merge of
   `change_analyzer`, `commit_body_generator` and `commit_title_generator`:
   process order (analysis, body, then title from the body), body tone rules
   (no actor, no meta-comments, no session narrative, no tag/module/trailers),
   a single bullet format (`- ` at column 0), 72-column hard wrap, title rules
   (50-character target, imperative, lowercase, no tag-verb restatement,
   abstract up), three worked examples and counter-examples taken from real
   drafts.
2. **Verifier.** Add three warning rules so `ai verify` and the `verify`
   block of `ai submit` report the same limits the prompt states:
   - `title_text_too_long`: text after `[TAG] scope: ` longer than 50.
   - `title_restates_tag_verb`: first word of the title text is in the tag's
     forbidden verb family (`tagVerbFamilies`), matched against the bare verb
     and its -s/-es/-ed/-d/-ing forms.
   - `body_line_too_long`: a body line over 72 columns that contains
     whitespace (an unbreakable URL or path is ignored). One finding per
     message, located at the first offending line.

All three are warnings: they never flip `has_errors`, so an agent can accept a
justified 75-character title.

Also fix the delegate bundle's `submit_example`, which omitted the pending
draft `id` while `instructions` required it.

## Implementation

- `internal/config/prompts/agent_commit.prompt.tmpl`: rewritten.
- `internal/aiengine/verify.go`: `tagVerbFamilies`, `titleTagCapture`,
  `checkTitleTextLength`, `checkTitleRestatesTag`, `checkBodyLineLength`,
  wired into `VerifyFinalMessage`.
- `internal/aiengine/verify_test.go`: one test per rule; the clean fixture
  is updated because it violated two of the new rules.
- `internal/aiengine/delegate.go`: `commitSubmitExample` and
  `releaseSubmitExample` carry `id` and use `jq -n`.

## Dependencies

- Unit 16 (`ai verify`), Unit 26 (delegate mode).

## Verify when done

- [x] `go test ./...` green, `go vet` clean.
- [x] Sweep: `commitcraft ai verify --id N` over the last 102 ai drafts
      reports 29 `body_line_too_long`, 12 `title_text_too_long`,
      11 `title_restates_tag_verb`.
- [x] `~/.config/CommitCraft/prompts/agent_commit.prompt` synced from the
      template.
- [ ] Re-run the sweep after ~30 new commits and compare counts.
