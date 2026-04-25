package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func updateSettingApiKey(msg tea.Msg, model *Model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var nextState appState

	model.apiKeyInput, cmd = model.apiKeyInput.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, model.keys.Enter):
			apiKey := model.apiKeyInput.Value()
			if apiKey != "" {
				err := saveAPIKeyToEnv(apiKey)
				if err != nil {
					model.err = err
					return model, nil
				}
				model.globalConfig.TUI.GroqAPIKey = apiKey
				model.globalConfig.TUI.IsAPIKeySet = true

				switch model.AppMode {
				case ReleaseMode:
					nextState = stateReleaseMainMenu
				case CommitMode:
					nextState = stateChoosingCommit
				}
				return model.cancelProcess(nextState)
			}
		case key.Matches(msg, model.keys.GlobalQuit):
			return model, tea.Quit
		}
	}
	return model, cmd
}

