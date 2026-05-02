package ai

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"commit_craft_reborn/internal/aiengine"
)

// runEdit applies direct overrides to a draft's title/body/changelog
// (and optionally tag/scope) without re-running any AI stage. The
// motivation is iteration: agents that detect issues in the generated
// text used to call `regenerate --stage …` and pay for another model
// round-trip; with `edit` they patch the offending field in place,
// recompose the final message, and persist. ai_calls telemetry from
// the previous run is left untouched on purpose — no new stage ran.
//
// Flag semantics: omitted means "keep stored value". A literal "-"
// reads the value from stdin (consumed once; later "-" flags reuse
// the buffered content). Empty strings ("") are rejected for
// title/body/tag/scope since FormatFinalMessage needs them populated;
// --changelog accepts the sentinel "CLEAR" to drop the entry.
func runEdit(args []string) int {
	fs := flagSet("ai edit")
	id := fs.Int("id", 0, "Draft ID to edit (required)")
	title := fs.String("title", "",
		"Replace the AI title. Use '-' to read from stdin.")
	body := fs.String("body", "",
		"Replace the AI body. Use '-' to read from stdin.")
	changelog := fs.String("changelog", "",
		"Replace the changelog entry. Use '-' to read from stdin, "+
			"or 'CLEAR' to drop the entry.")
	tag := fs.String("tag", "", "Override the commit type tag")
	scope := fs.String("scope", "", "Override the commit scope")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		printErrorJSON("invalid_input", err.Error())
		return 2
	}
	titleSet, bodySet, changelogSet, tagSet, scopeSet := false, false, false, false, false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "title":
			titleSet = true
		case "body":
			bodySet = true
		case "changelog":
			changelogSet = true
		case "tag":
			tagSet = true
		case "scope":
			scopeSet = true
		}
	})
	if *id <= 0 {
		printErrorJSON("invalid_input", "--id is required")
		return 2
	}
	if !titleSet && !bodySet && !changelogSet && !tagSet && !scopeSet {
		printErrorJSON("invalid_input",
			"nothing to edit — pass at least one of --title/--body/--changelog/--tag/--scope")
		return 2
	}

	bs, err := loadBootstrap()
	if err != nil {
		printErrorJSON("bootstrap_error", err.Error())
		return 1
	}
	defer bs.db.Close()

	c, err := bs.db.GetCommitByID(*id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			printErrorJSON("not_found", fmt.Sprintf("commit with id=%d not found", *id))
			return 1
		}
		printErrorJSON("db_error", err.Error())
		return 1
	}

	// Capture pre-edit identity so the mention-line heuristic can match
	// the previous final_message before we overwrite the fields.
	oldTitle := c.IaTitle
	oldBody := c.IaCommitRaw
	oldFinal := c.MessageEN

	// Resolve "-" placeholders against stdin lazily. Single-shot cache
	// so multiple "-" flags share the same stdin buffer.
	var stdinCache *string
	readStdin := func() (string, error) {
		if stdinCache != nil {
			return *stdinCache, nil
		}
		raw, rerr := io.ReadAll(os.Stdin)
		if rerr != nil {
			return "", rerr
		}
		s := string(raw)
		stdinCache = &s
		return s, nil
	}
	resolve := func(v string) (string, error) {
		if v == "-" {
			return readStdin()
		}
		return v, nil
	}

	if titleSet {
		v, rerr := resolve(*title)
		if rerr != nil {
			printErrorJSON("invalid_input", fmt.Sprintf("read --title: %s", rerr.Error()))
			return 2
		}
		v = strings.TrimSpace(v)
		if v == "" {
			printErrorJSON("invalid_input", "--title cannot be empty")
			return 2
		}
		c.IaTitle = v
	}
	if bodySet {
		v, rerr := resolve(*body)
		if rerr != nil {
			printErrorJSON("invalid_input", fmt.Sprintf("read --body: %s", rerr.Error()))
			return 2
		}
		v = strings.TrimRight(v, " \n\t")
		if strings.TrimSpace(v) == "" {
			printErrorJSON("invalid_input", "--body cannot be empty")
			return 2
		}
		c.IaCommitRaw = v
	}
	if changelogSet {
		v, rerr := resolve(*changelog)
		if rerr != nil {
			printErrorJSON("invalid_input", fmt.Sprintf("read --changelog: %s", rerr.Error()))
			return 2
		}
		if strings.TrimSpace(v) == "CLEAR" {
			c.IaChangelog = ""
		} else {
			c.IaChangelog = strings.TrimRight(v, " \n\t")
		}
	}
	if tagSet {
		t := strings.TrimSpace(*tag)
		if t == "" {
			printErrorJSON("invalid_input", "--tag cannot be empty")
			return 2
		}
		if !tagIsKnown(t, bs.finalCommitTypes) {
			printErrorJSON("invalid_input", fmt.Sprintf("unknown tag %q", t))
			return 2
		}
		c.Type = t
	}
	if scopeSet {
		s := strings.TrimSpace(*scope)
		if s == "" {
			printErrorJSON("invalid_input", "--scope cannot be empty")
			return 2
		}
		c.Scope = s
	}

	// Recompose MessageEN, preserving the changelog mention line that
	// the refiner appended on the previous run when possible.
	mention := extractMentionLine(oldFinal, oldTitle, oldBody)
	c.MessageEN = aiengine.ComposeFinalMessage(c.IaTitle, c.IaCommitRaw, mention)

	if err := bs.db.SaveDraft(&c); err != nil {
		printErrorJSON("db_error", err.Error())
		return 1
	}

	saved, err := bs.db.GetCommitByID(c.ID)
	if err != nil {
		saved = c
	}
	stages := loadStagesForCommit(bs.db, saved.ID)
	cj, err := commitToJSON(saved, stages, bs.cfg.CommitFormat.TypeFormat)
	if err != nil {
		printErrorJSON("incomplete_commit", err.Error())
		return 1
	}
	printCommitJSON(cj)
	return 0
}

// extractMentionLine recovers the changelog mention tail from the
// previous final_message. ComposeFinalMessage formats as
// "<title>\n\n<body>[\n\n<mention>]"; if the previous final still
// matches the pre-edit (oldTitle, oldBody) prefix, the leftover is the
// mention. Otherwise, fall back to scanning for "\n\nChangelog:" since
// the refiner always emits that token. Returns "" when nothing
// confidently extracts.
func extractMentionLine(prevFinal, oldTitle, oldBody string) string {
	if prevFinal == "" {
		return ""
	}
	prefix := oldTitle + "\n\n" + oldBody
	if rest, ok := strings.CutPrefix(prevFinal, prefix); ok {
		return strings.TrimSpace(rest)
	}
	idx := strings.LastIndex(prevFinal, "\n\nChangelog:")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(prevFinal[idx+2:])
}
