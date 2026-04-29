package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/changelog"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
)

// mentionStripRegex matches an `@token` mention preceded by start-of-line
// or whitespace, capturing the leading boundary and the bare token so we
// can drop the `@` while keeping the path text. The path itself is
// information the AI needs ("which files are referenced"); only the
// visual marker is for the human.
var mentionStripRegex = regexp.MustCompile(`(^|\s)@([\w./-]+)`)

// stripMentions removes the leading `@` from every mention in s, leaving
// the file path/identifier intact. Called on every user-supplied snippet
// (key points, summary) right before assembling an AI prompt.
func stripMentions(s string) string {
	return mentionStripRegex.ReplaceAllString(s, "$1$2")
}

func createAndSendIaMessage(
	systemPrompt string,
	userInput string,
	iaModel string,
	model *Model,
) (string, *api.CallStats, error) {
	if iaModel == "" {
		iaModel = "llama-3.1-8b-instant"
	}
	apiKey := model.globalConfig.TUI.GroqAPIKey
	messages := []api.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userInput,
		},
	}
	response, stats, err := api.GetGroqChatCompletion(apiKey, iaModel, messages)
	if err != nil {
		return "", stats, fmt.Errorf(
			"An error occurred while making the following call:\n systemPrompt: %s\n userInput: %s\n Error: %s",
			systemPrompt,
			userInput,
			err,
		)
	}
	if stats != nil {
		api.RecordRateLimits(iaModel, stats.RateLimits)
		persistRateLimits(model, iaModel, stats.RateLimits)
		logRateLimits(model, iaModel, stats.RateLimits)
	}
	return response, stats, nil
}

// logRateLimits emits a debug line with the parsed rate-limit fields so
// the user can confirm via Ctrl+L which headers Groq actually returned
// for a given model. A `*Parsed=false` flag is the signal that the
// matching `remaining-*` header was absent from the response.
func logRateLimits(model *Model, modelID string, rl api.RateLimits) {
	if model == nil || model.log == nil {
		return
	}
	model.log.Debug(
		"rate-limit headers",
		"model", modelID,
		"limit_requests", rl.LimitRequests,
		"remaining_requests", rl.RemainingRequests,
		"reset_requests", rl.ResetRequests,
		"requests_parsed", rl.RequestsParsed,
		"limit_tokens", rl.LimitTokens,
		"remaining_tokens", rl.RemainingTokens,
		"reset_tokens", rl.ResetTokens,
		"tokens_parsed", rl.TokensParsed,
	)
}

// persistRateLimits UPSERTs the just-captured rate-limit snapshot for
// modelID so the in-memory cache can be rehydrated on the next launch.
// Best-effort — DB failures are logged but never propagated, the live
// cache already has the data for the current session.
func persistRateLimits(model *Model, modelID string, rl api.RateLimits) {
	if model == nil || model.db == nil || modelID == "" {
		return
	}
	row := storage.ModelRateLimits{
		ModelID:           modelID,
		LimitRequests:     rl.LimitRequests,
		RemainingRequests: rl.RemainingRequests,
		ResetRequestsMs:   int(rl.ResetRequests / time.Millisecond),
		LimitTokens:       rl.LimitTokens,
		RemainingTokens:   rl.RemainingTokens,
		ResetTokensMs:     int(rl.ResetTokens / time.Millisecond),
		CapturedAt:        rl.CapturedAt,
		RequestsParsed:    rl.RequestsParsed,
		TokensParsed:      rl.TokensParsed,
		RequestsToday:     rl.RequestsToday,
		RequestsDay:       rl.RequestsDay,
	}
	if err := model.db.SaveModelRateLimits(row); err != nil {
		model.log.Warn("rate-limit persistence failed", "model", modelID, "error", err)
	}
}

// recordStageStats copies a CallStats into the per-stage record on the
// pipeline model so the card and the persistence layer can read the same
// numbers. Safe with stats == nil (no-op) so error paths can still call it.
func recordStageStats(model *Model, id stageID, stats *api.CallStats) {
	if model == nil || stats == nil {
		return
	}
	if int(id) < 0 || int(id) >= len(model.pipeline.stages) {
		return
	}
	st := &model.pipeline.stages[id]
	st.HasStats = true
	st.PromptTokens = stats.PromptTokens
	st.CompletionTokens = stats.CompletionTokens
	st.TotalTokens = stats.TotalTokens
	st.QueueTime = stats.QueueTime
	st.PromptTime = stats.PromptTime
	st.CompletionTime = stats.CompletionTime
	st.APITotalTime = stats.TotalTime
	st.RequestID = stats.RequestID
	st.StatsModel = stats.Model
	st.TPMLimitAtCall = stats.RateLimits.LimitTokens
}

func iaCallChangeAnalyzer(model *Model) (string, error) {
	promptConfig := model.globalConfig.Prompts

	var gitChanges string
	var err error
	if model.useDbCommmit {
		gitChanges = model.diffCode
	} else {
		gitChanges, err = git.GetStagedDiffSummary(model.globalConfig.Prompts.ChangeAnalyzerMaxDiffSize)
		if err != nil {
			return "", fmt.Errorf("failed to get staged diff: %w", err)
		}
	}
	model.diffCode = gitChanges

	developerPoints := stripMentions(strings.Join(model.keyPoints, "\n"))
	model.log.Debug(
		"Change Analyzer input",
		"developerPoints",
		developerPoints,
		"gitChanges",
		gitChanges,
	)

	result, stats, err := createAndSendIaMessage(
		promptConfig.ChangeAnalyzerPrompt,
		fmt.Sprintf("DEVELOPER_POINTS:\n%s\nGIT_CHANGES:\n%s", developerPoints, gitChanges),
		promptConfig.ChangeAnalyzerPromptModel,
		model,
	)
	if err != nil {
		return "", fmt.Errorf("stage 1 (change analyzer): %w", err)
	}
	recordStageStats(model, stageSummary, stats)
	model.log.Debug("Change Analyzer output", "result", result)
	return result, nil
}

func iaCallCommitBodyGenerator(model *Model, summaryParagraphs string) (string, error) {
	promptConfig := model.globalConfig.Prompts

	result, stats, err := createAndSendIaMessage(
		promptConfig.CommitBodyGeneratorPrompt,
		fmt.Sprintf("TAG:\n%s\nMODULE:\n%s\nSUMMARY_PARAGRAPHS:\n%s",
			model.commitType, model.commitScope, summaryParagraphs),
		promptConfig.CommitBodyGeneratorPromptModel,
		model,
	)
	if err != nil {
		return "", fmt.Errorf("stage 2 (commit body): %w", err)
	}
	recordStageStats(model, stageBody, stats)
	model.log.Debug("Commit Body Generator output", "result", result)
	return result, nil
}

func iaCallCommitTitleGenerator(model *Model, commitBody string) (string, error) {
	promptConfig := model.globalConfig.Prompts

	result, stats, err := createAndSendIaMessage(
		promptConfig.CommitTitleGeneratorPrompt,
		fmt.Sprintf("TAG:\n%s\nMODULE:\n%s\nCOMMIT_BODY:\n%s",
			model.commitType, model.commitScope, commitBody),
		promptConfig.CommitTitleGeneratorPromptModel,
		model,
	)
	if err != nil {
		return "", fmt.Errorf("stage 3 (commit title): %w", err)
	}
	recordStageStats(model, stageTitle, stats)
	model.log.Debug("Commit Title Generator output", "result", result)
	return strings.TrimSpace(result), nil
}

func assembleCommitMessage(titleText, commitBody string) string {
	return fmt.Sprintf("%s\n\n%s", titleText, commitBody)
}

func assembleOutputCommitMessage(model *Model, commit storage.Commit) string {
	formattedCommitType := fmt.Sprintf(model.globalConfig.CommitFormat.TypeFormat, commit.Type)
	return fmt.Sprintf("%s %s: %s", formattedCommitType, commit.Scope, commit.MessageEN)
}

func ia_commit_builder(model *Model) error {
	summaryParagraphs, err := iaCallChangeAnalyzer(model)
	if err != nil {
		return err
	}
	model.iaSummaryOutput = summaryParagraphs

	commitBody, err := iaCallCommitBodyGenerator(model, summaryParagraphs)
	if err != nil {
		return err
	}
	model.iaCommitRawOutput = commitBody

	titleText, err := iaCallCommitTitleGenerator(model, commitBody)
	if err != nil {
		return err
	}

	model.iaTitleRawOutput = titleText
	runChangelogRefiner(model)
	model.commitTranslate = composeFinalCommitMessage(model)
	model.log.Debug("Final commit message", "commitTranslate", model.commitTranslate)
	return nil
}

// changelogRefinerOutput mirrors the JSON contract documented in
// prompts/changelog_refiner.prompt.tmpl. The refiner emits two independent
// pieces of text: the new CHANGELOG block and a single mention line that
// will be appended to the commit body. Stage 2's body is never rewritten —
// the mention line is added on top, the existing bullets stay verbatim.
type changelogRefinerOutput struct {
	ChangelogEntry    string `json:"changelog_entry"`
	CommitMentionLine string `json:"commit_mention_line"`
}

// runChangelogRefiner is the optional 4th step. It detects the project's
// CHANGELOG and asks the AI for two independent pieces of text: the new
// CHANGELOG block and a one-line mention to append to the commit body.
// Stage 2's body is sent purely as input context for the entry — this
// function never mutates iaCommitRawOutput, so the stage 2 viewport keeps
// showing the real model output. The mention line lives in
// iaChangelogMentionLine and is concatenated by composeFinalCommitMessage.
func runChangelogRefiner(model *Model) {
	model.iaChangelogEntry = ""
	model.iaChangelogMentionLine = ""

	cfg := model.globalConfig.Changelog
	// changelogActive is the single source of truth: pipelineStartFullRun
	// already evaluated `Enabled` plus the dirty-file safeguard and decided
	// whether the refiner should run. Re-checking only `cfg.Enabled` here
	// would bypass the safeguard (the file exists and is readable, so the
	// downstream Detect would succeed and the AI call would still fire).
	if !cfg.Enabled || !model.changelogActive {
		return
	}

	info, err := changelog.Detect(model.pwd, cfg.Path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			model.log.Warn("Changelog detect failed, skipping refiner", "error", err)
		}
		return
	}

	suggested := changelog.SuggestNextVersion(info.LatestVersion, cfg.BumpStrategy)
	prompt := cfg.Prompt
	if prompt == "" {
		model.log.Warn("Changelog prompt is empty, skipping refiner")
		return
	}

	bulletHint := pickBulletStyle(model.iaCommitRawOutput)
	if bulletHint == "" {
		bulletHint = "none"
	}

	userInput := fmt.Sprintf(
		"FORMAT_SAMPLE:\n%s\nSUGGESTED_VERSION:\n%s\nDATE:\n%s\nSTAGE2_BODY:\n%s\nSTAGE3_TITLE:\n%s\nBODY_BULLET_STYLE:\n%s",
		info.FormatSample,
		suggested,
		time.Now().Format("2006-01-02"),
		model.iaCommitRawOutput,
		model.iaTitleRawOutput,
		bulletHint,
	)

	response, stats, err := createAndSendIaMessage(prompt, userInput, cfg.PromptModel, model)
	if err != nil {
		model.log.Warn("Changelog refiner call failed", "error", err)
		return
	}
	recordStageStats(model, stageChangelog, stats)

	parsed, perr := parseChangelogRefinerJSON(response)
	if perr != nil {
		model.log.Warn("Changelog refiner JSON parse failed, using fallback", "error", perr)
		model.iaChangelogEntry = fallbackChangelogEntry(
			suggested,
			model.iaTitleRawOutput,
			model.iaCommitRawOutput,
		)
		model.iaChangelogMentionLine = fallbackMentionLine(model.iaCommitRawOutput, suggested)
		model.iaChangelogTargetPath = info.Path
		model.iaChangelogSuggestedVersion = suggested
		return
	}

	mention := strings.TrimSpace(parsed.CommitMentionLine)
	// Safety net: if the model dropped the mention or omitted the
	// CHANGELOG.md token, build a deterministic fallback so the final
	// commit message always documents the update.
	if mention == "" || !strings.Contains(strings.ToLower(mention), "changelog.md") {
		mention = fallbackMentionLine(model.iaCommitRawOutput, suggested)
	}

	model.iaChangelogEntry = strings.TrimSpace(parsed.ChangelogEntry)
	model.iaChangelogMentionLine = mention
	model.iaChangelogTargetPath = info.Path
	model.iaChangelogSuggestedVersion = suggested
	model.log.Debug(
		"Changelog refiner output",
		"entry", model.iaChangelogEntry,
		"mention", model.iaChangelogMentionLine,
		"version", suggested,
	)
}

// composeFinalCommitMessage builds the user-visible commit message: stage 3
// title + stage 2 body verbatim, plus the refiner's mention line appended at
// the end when present. Stage 2's stored output is never modified — the
// appended line only lives in the final commitTranslate string.
func composeFinalCommitMessage(model *Model) string {
	body := model.iaCommitRawOutput
	if model.iaChangelogMentionLine != "" {
		body = strings.TrimRight(body, " \n\t") + "\n\n" + model.iaChangelogMentionLine
	}
	return assembleCommitMessage(model.iaTitleRawOutput, body)
}

// parseChangelogRefinerJSON extracts the refiner's JSON payload, tolerating
// the model wrapping it in prose or a markdown code fence.
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

// fallbackMentionLine builds a deterministic single-line mention used when
// the refiner doesn't return a usable commit_mention_line. The bullet
// character is matched to the existing body so the appended line blends in.
func fallbackMentionLine(body, version string) string {
	bullet := pickBulletStyle(body)
	if bullet == "" {
		return fmt.Sprintf("Updated CHANGELOG.md with %s entry.", version)
	}
	return fmt.Sprintf("%s Updated CHANGELOG.md with %s entry.", bullet, version)
}

// pickBulletStyle scans body for the first existing bullet and returns the
// same prefix character. Returns the empty string when the body has no
// bullets — callers should treat that as "use a plain sentence".
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

// fallbackChangelogEntry builds a minimal entry when the AI response is
// unusable. Conservative: heading + title + first paragraph of the body.
func fallbackChangelogEntry(version, title, body string) string {
	first := strings.SplitN(strings.TrimSpace(body), "\n\n", 2)[0]
	return fmt.Sprintf("## %s — %s\n\n%s\n\n%s",
		version,
		time.Now().Format("2006-01-02"),
		strings.TrimSpace(title),
		first,
	)
}

func iaReleaseBuilder(model *Model) error {
	var input strings.Builder
	delimiter := "--- COMMIT SEPARATOR ---"
	for _, item := range model.selectedCommitList {
		commitContent := fmt.Sprintf(
			"%s\nCommit.Date:%s\nCommit.Title:%s\ncommit.body:%s\n%s\n",
			delimiter,
			item.Date,
			item.Subject,
			item.Body,
			delimiter,
		)
		input.WriteString(commitContent)
	}
	promptConfig := model.globalConfig.Prompts
	model.log.Debug("release ia Input", "input", input)

	iaResponse, _, err := createAndSendIaMessage(
		promptConfig.ReleasePrompt,
		input.String(),
		promptConfig.ReleasePromptModel,
		model,
	)
	if err != nil {
		model.log.Error(
			fmt.Sprintf("An error occurred while trying to generate the release output.\n%s", err),
		)
		return fmt.Errorf(
			"An error occurred while trying to generate the release output.\n%s",
			ExtractJSONError(err.Error()),
		)
	}
	model.commitLivePreview = iaResponse
	model.releaseText = iaResponse
	return nil
}
