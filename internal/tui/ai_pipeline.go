package tui

import (
	"fmt"
	"strings"

	"commit_craft_reborn/internal/api"
	"commit_craft_reborn/internal/git"
	"commit_craft_reborn/internal/storage"
)

func createAndSendIaMessage(
	systemPrompt string,
	userInput string,
	iaModel string,
	model *Model,
) (string, error) {
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
	response, err := api.GetGroqChatCompletion(apiKey, iaModel, messages)
	if err != nil {
		return "", fmt.Errorf(
			"An error occurred while making the following call:\n systemPrompt: %s\n userInput: %s\n Error: %s",
			systemPrompt,
			userInput,
			err,
		)
	}
	return response, nil
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

	developerPoints := strings.Join(model.keyPoints, "\n")
	model.log.Debug(
		"Change Analyzer input",
		"developerPoints",
		developerPoints,
		"gitChanges",
		gitChanges,
	)

	result, err := createAndSendIaMessage(
		promptConfig.ChangeAnalyzerPrompt,
		fmt.Sprintf("DEVELOPER_POINTS:\n%s\nGIT_CHANGES:\n%s", developerPoints, gitChanges),
		promptConfig.ChangeAnalyzerPromptModel,
		model,
	)
	if err != nil {
		return "", fmt.Errorf("change analyzer call failed: %w", err)
	}
	model.log.Debug("Change Analyzer output", "result", result)
	return result, nil
}

func iaCallCommitBodyGenerator(model *Model, summaryParagraphs string) (string, error) {
	promptConfig := model.globalConfig.Prompts

	result, err := createAndSendIaMessage(
		promptConfig.CommitBodyGeneratorPrompt,
		fmt.Sprintf("TAG:\n%s\nMODULE:\n%s\nSUMMARY_PARAGRAPHS:\n%s",
			model.commitType, model.commitScope, summaryParagraphs),
		promptConfig.CommitBodyGeneratorPromptModel,
		model,
	)
	if err != nil {
		return "", fmt.Errorf("commit body generator call failed: %w", err)
	}
	model.log.Debug("Commit Body Generator output", "result", result)
	return result, nil
}

func iaCallCommitTitleGenerator(model *Model, commitBody string) (string, error) {
	promptConfig := model.globalConfig.Prompts

	result, err := createAndSendIaMessage(
		promptConfig.CommitTitleGeneratorPrompt,
		fmt.Sprintf("TAG:\n%s\nMODULE:\n%s\nCOMMIT_BODY:\n%s",
			model.commitType, model.commitScope, commitBody),
		promptConfig.CommitTitleGeneratorPromptModel,
		model,
	)
	if err != nil {
		return "", fmt.Errorf("commit title generator call failed: %w", err)
	}
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
	model.commitTranslate = assembleCommitMessage(titleText, commitBody)
	model.log.Debug("Final commit message", "commitTranslate", model.commitTranslate)
	return nil
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

	iaResponse, err := createAndSendIaMessage(
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

