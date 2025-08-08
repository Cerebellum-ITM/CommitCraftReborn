package cli

import (
	"fmt"
	"os"

	"commit_craft_reborn/pkg/db"
	"commit_craft_reborn/pkg/model"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle     = lipgloss.NewStyle().Margin(1, 2)
	formBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1)
	dbClient     *db.DB
)

type (
	ui struct {
		model model.Model
	}
	commitsLoadedMsg struct{ commits []model.Commit }
	commitCreatedMsg struct{}
)

// Run is the entry point for the CLI application.
func Run() {
	var err error
	dbClient, err = db.New()
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	initialModel := ui{model: model.NewModel()}
	p := tea.NewProgram(initialModel, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(ui); ok && m.model.SelectedMessage != "" {
		fmt.Print(m.model.SelectedMessage)
	}
}

func (m ui) Init() tea.Cmd {
	return loadCommits
}

func (m ui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.model.CommitList.SetSize(msg.Width, msg.Height)
		m.model.CommitTypeList.SetSize(msg.Width, msg.Height)
	case commitCreatedMsg:
		m.model.Mode = model.ListView
		return m, loadCommits
	case commitsLoadedMsg:
		items := make([]list.Item, len(msg.commits))
		for i, c := range msg.commits {
			items[i] = c
		}
		m.model.CommitList.SetItems(items)
		return m, nil
	case error:
		m.model.Err = msg
		return m, tea.Quit
	}

	switch m.model.Mode {
	case model.SelectCommitTypeView:
		return m.updateSelectTypeView(msg)
	case model.EnterScopeView:
		return m.updateEnterScopeView(msg)
	case model.EnterMessageView:
		return m.updateEnterMessageView(msg)
	default: // ListView
		return m.updateListView(msg)
	}
}

func (m ui) updateListView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "n":
			m.model.Mode = model.SelectCommitTypeView
			return m, nil
		case "enter":
			if i, ok := m.model.CommitList.SelectedItem().(model.Commit); ok {
				m.model.SelectedMessage = fmt.Sprintf("[%s] %s: %s", i.Type, i.Scope, i.MessageEN)
				return m, tea.Quit
			}
		}
	}
	m.model.CommitList, cmd = m.model.CommitList.Update(msg)
	return m, cmd
}

func (m ui) updateSelectTypeView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.model.CommitTypeList.SelectedItem().(model.CommitType); ok {
				m.model.NewCommit.Type = i.Tag
				m.model.Mode = model.EnterScopeView
				m.model.Inputs[0].Focus()
			}
		case "esc":
			m.model.Mode = model.ListView
		}
	}
	m.model.CommitTypeList, cmd = m.model.CommitTypeList.Update(msg)
	return m, cmd
}

func (m ui) updateEnterScopeView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.model.NewCommit.Scope = m.model.Inputs[0].Value()
			m.model.Mode = model.EnterMessageView
			m.model.Inputs[0].Blur()
			m.model.Inputs[1].Focus()
		case "esc":
			m.model.Mode = model.SelectCommitTypeView
		}
	}
	m.model.Inputs[0], cmd = m.model.Inputs[0].Update(msg)
	return m, cmd
}

func (m ui) updateEnterMessageView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.model.NewCommit.MessageES = m.model.Inputs[1].Value()
			return m, createCommit(m.model.NewCommit)
		case "esc":
			m.model.Mode = model.EnterScopeView
		}
	}
	m.model.Inputs[1], cmd = m.model.Inputs[1].Update(msg)
	return m, cmd
}

func (m ui) View() string {
	if m.model.Err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.model.Err)
	}

	switch m.model.Mode {
	case model.SelectCommitTypeView:
		return appStyle.Render(m.model.CommitTypeList.View())
	case model.EnterScopeView:
		return m.formView("Enter Scope")
	case model.EnterMessageView:
		return m.formView("Enter Message (Spanish)")
	default: // ListView
		return appStyle.Render(m.model.CommitList.View())
	}
}

func (m ui) formView(title string) string {
	input := m.model.Inputs[0]
	if m.model.Mode == model.EnterMessageView {
		input = m.model.Inputs[1]
	}
	return formBoxStyle.Render(fmt.Sprintf(
		"%s\n\n%s",
		title,
		input.View(),
	))
}

func loadCommits() tea.Msg {
	commits, err := dbClient.GetCommits()
	if err != nil {
		return err
	}
	return commitsLoadedMsg{commits}
}

func createCommit(commit model.Commit) tea.Cmd {
	return func() tea.Msg {
		commit.Workspace = "default" // Placeholder
		if err := dbClient.CreateCommit(commit); err != nil {
			return err
		}
		return commitCreatedMsg{}
	}
}
