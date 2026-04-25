package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (model *Model) iaHeaderView(state string) string {
	title := "Final response of AI models"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) iaFooterView(state string) string {
	info := fmt.Sprintf("%3.f%%", model.iaViewport.ScrollPercent()*100)
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
}

func (model *Model) userInputHeaderView(state string) string {
	title := "Enter the text with your summary of the changes"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) userInputFooterView(state string) string {
	textColor, lineColor := model.setColorVariables(state)

	scrollInfo := fmt.Sprintf("%3.f%%", model.commitsKeysViewport.ScrollPercent()*100)
	charInfo := fmt.Sprintf(
		"Number of characters %d",
		lipgloss.Width(model.commitsKeysInput.Value()),
	)

	leftContent := FooterStyle.Foreground(textColor).Render(scrollInfo)
	rightContent := FooterStyle.Foreground(textColor).Render(charInfo)

	lineW := max(0, model.width/2-lipgloss.Width(leftContent)-lipgloss.Width(rightContent))
	line := LineStyle.Foreground(lineColor).Render(strings.Repeat("─", lineW))

	return lipgloss.JoinHorizontal(lipgloss.Left, leftContent, line, rightContent)
}

func (model *Model) msgEditHeaderView(state string) string {
	title := "Write the modifications"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) msgEditFooterView(state string) string {
	info := fmt.Sprintf("Number of characters %d", lipgloss.Width(model.msgEdit.Value()))
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
}

func (model *Model) releaseHeaderView(state string) string {
	title := "Commit list | Select at least one commit"
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) releaseFooterView(state string) string {
	commitSymbol := model.Theme.AppSymbols().Commit
	info := fmt.Sprintf("%s %d %s",
		commitSymbol,
		len(model.selectedCommitList),
		"Selected Commits",
	)
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
}

func (model *Model) releaseLivePreviewHeaderView(state string) string {
	var title string
	switch model.state {
	case stateReleaseChoosingCommits:
		title = "Commit content"
		if model.releaseViewState.releaseCreated {
			title = title + " - Use the shortcut keys to switch between the response and the commit preview."
		}
	case stateReleaseBuildingText:
		title = "AI model response"
	}
	return model.buildStyledBorder(
		state,
		title,
		HeaderStyle,
		model.width/2,
		AlignHeader,
	)
}

func (model *Model) releaseLivePreviewFooterView(state string) string {
	info := fmt.Sprintf("%3.f%%", model.releaseViewport.ScrollPercent()*100)
	return model.buildStyledBorder(
		state,
		info,
		FooterStyle,
		model.width/2,
		AlignFooter,
	)
}
