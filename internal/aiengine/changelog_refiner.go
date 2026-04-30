package aiengine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"commit_craft_reborn/internal/changelog"
)

// changelogRefinerOutput mirrors the JSON contract documented in
// prompts/changelog_refiner.prompt.tmpl. The refiner emits two
// independent pieces of text: the new CHANGELOG block and a single
// mention line that will be appended to the commit body.
type changelogRefinerOutput struct {
	ChangelogEntry    string `json:"changelog_entry"`
	CommitMentionLine string `json:"commit_mention_line"`
}

// RunChangelogRefiner runs the optional 4th stage. Detects the
// CHANGELOG, asks the AI for a new entry + a mention line, and writes
// the results onto out. Best-effort: any failure is logged and the run
// continues with empty changelog fields. Reads out.Body and out.Title
// as input, so callers re-running only this stage can populate those
// from an existing draft.
func RunChangelogRefiner(deps Deps, out *Output) {
	out.ChangelogEntry = ""
	out.ChangelogMentionLine = ""

	cfg := deps.Cfg.Changelog
	if !cfg.Enabled {
		return
	}

	info, err := changelog.Detect(deps.Pwd, cfg.Path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) && deps.Log != nil {
			deps.Log.Warn("Changelog detect failed, skipping refiner", "error", err)
		}
		return
	}

	suggested := changelog.SuggestNextVersion(info.LatestVersion, cfg.BumpStrategy)
	prompt := cfg.Prompt
	if prompt == "" {
		if deps.Log != nil {
			deps.Log.Warn("Changelog prompt is empty, skipping refiner")
		}
		return
	}

	bulletHint := pickBulletStyle(out.Body)
	if bulletHint == "" {
		bulletHint = "none"
	}

	userInput := fmt.Sprintf(
		"FORMAT_SAMPLE:\n%s\nSUGGESTED_VERSION:\n%s\nDATE:\n%s\nSTAGE2_BODY:\n%s\nSTAGE3_TITLE:\n%s\nBODY_BULLET_STYLE:\n%s",
		info.FormatSample,
		suggested,
		time.Now().Format("2006-01-02"),
		out.Body,
		out.Title,
		bulletHint,
	)

	response, stats, err := SendIaMessage(deps, prompt, userInput, cfg.PromptModel)
	if err != nil {
		if deps.Log != nil {
			deps.Log.Warn("Changelog refiner call failed", "error", err)
		}
		return
	}
	RecordStage(out, StageChangelog, cfg.PromptModel, stats)

	parsed, perr := parseChangelogRefinerJSON(response)
	if perr != nil {
		if deps.Log != nil {
			deps.Log.Warn("Changelog refiner JSON parse failed, using fallback", "error", perr)
		}
		out.ChangelogEntry = fallbackChangelogEntry(suggested, out.Title, out.Body)
		out.ChangelogMentionLine = fallbackMentionLine(out.Body, suggested)
		out.ChangelogTargetPath = info.Path
		out.ChangelogSuggestedVersion = suggested
		return
	}

	mention := strings.TrimSpace(parsed.CommitMentionLine)
	if mention == "" || !strings.Contains(strings.ToLower(mention), "changelog.md") {
		mention = fallbackMentionLine(out.Body, suggested)
	}

	out.ChangelogEntry = strings.TrimSpace(parsed.ChangelogEntry)
	out.ChangelogMentionLine = mention
	out.ChangelogTargetPath = info.Path
	out.ChangelogSuggestedVersion = suggested
	if deps.Log != nil {
		deps.Log.Debug(
			"Changelog refiner output",
			"entry", out.ChangelogEntry,
			"mention", out.ChangelogMentionLine,
			"version", suggested,
		)
	}
}

// parseChangelogRefinerJSON extracts the refiner's JSON payload,
// tolerating prose or a markdown code fence around it.
func parseChangelogRefinerJSON(raw string) (changelogRefinerOutput, error) {
	var out changelogRefinerOutput
	trimmed := strings.TrimSpace(raw)
	if i := strings.Index(trimmed, "{"); i >= 0 {
		if j := strings.LastIndex(trimmed, "}"); j > i {
			trimmed = trimmed[i : j+1]
		}
	}
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return out, err
	}
	if out.ChangelogEntry == "" {
		return out, fmt.Errorf("missing changelog_entry field")
	}
	return out, nil
}

// fallbackMentionLine builds a deterministic single-line mention used
// when the refiner doesn't return a usable commit_mention_line. The
// bullet character matches the existing body so the line blends in.
func fallbackMentionLine(body, version string) string {
	bullet := pickBulletStyle(body)
	if bullet == "" {
		return fmt.Sprintf("Updated CHANGELOG.md with %s entry.", version)
	}
	return fmt.Sprintf("%s Updated CHANGELOG.md with %s entry.", bullet, version)
}

// pickBulletStyle scans body for the first existing bullet and returns
// the same prefix character, or "" when the body has no bullets.
func pickBulletStyle(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) >= 2 {
			switch trimmed[0] {
			case '-', '*', '+':
				if trimmed[1] == ' ' {
					return string(trimmed[0])
				}
			}
		}
	}
	return ""
}

// fallbackChangelogEntry builds a minimal entry when the AI response
// is unusable. Heading + title + first paragraph of the body.
func fallbackChangelogEntry(version, title, body string) string {
	first := strings.SplitN(strings.TrimSpace(body), "\n\n", 2)[0]
	return fmt.Sprintf("## %s — %s\n\n%s\n\n%s",
		version,
		time.Now().Format("2006-01-02"),
		strings.TrimSpace(title),
		first,
	)
}
