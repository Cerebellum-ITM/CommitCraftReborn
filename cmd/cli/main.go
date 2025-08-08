package cli

import (
	"fmt"
	"os"
	"time"

	"commit_craft_reborn/pkg/db"
	"commit_craft_reborn/pkg/model"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	appStyle      = lipgloss.NewStyle().Margin(1, 2)
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	formBoxStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1)
	dbClient      *db.DB
)

type (
	ui struct {
		state model.State
	}
	commitActionMsg struct{ message string }
	clearStatusMsg  struct{}
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

	pwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	initialState := model.NewState(pwd)
	m := ui{state: initialState}
	m.loadCommits()

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	if fm, ok := finalModel.(ui); ok && fm.state.SelectedCommitMessage != "" {
		fmt.Print(fm.state.SelectedCommitMessage)
	}
}

func (m ui) Init() tea.Cmd { return nil }

func (m ui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case commitActionMsg:
		m.state.StatusMessage = msg.message
		m.loadCommits()
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})
	case clearStatusMsg:
		m.state.StatusMessage = ""
		return m, nil
	}

	var cmd tea.Cmd
	if m.state.List.Title == "New Commit" {
		m, cmd = m.updateNewCommitView(msg)
	} else {
		m, cmd = m.updateListView(msg)
	}
	return m, cmd
}

func (m ui) updateListView(msg tea.Msg) (ui, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.state.List.SetSize(msg.Width-h, msg.Height-v)
	case tea.KeyMsg:
		if m.state.List.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("n"))):
			m.state.List.Title = "New Commit"
			return m, nil
		case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
			return m, m.deleteCommit
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if i, ok := m.state.List.SelectedItem().(model.Commit); ok {
				m.state.SelectedCommitMessage = fmt.Sprintf("[%s] %s: %s", i.Type, i.Scope, i.MessageEN)
				return m, tea.Quit
			}
		}
	}
	m.state.List, cmd = m.state.List.Update(msg)
	return m, cmd
}

func (m ui) updateNewCommitView(msg tea.Msg) (ui, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.state.List.Title = "Commit History"
			return m, nil
		case tea.KeyEnter:
			if m.state.FocusIndex == 0 {
				m.state.FocusIndex = 1
				m.state.Inputs[0].Blur()
				m.state.Inputs[1].Focus()
				return m, nil
			}
			return m, m.createCommit
		}
	}
	m.state.Inputs[m.state.FocusIndex], cmd = m.state.Inputs[m.state.FocusIndex].Update(msg)
	return m, cmd
}

func (m ui) View() string {
	if m.state.Err != nil {
		return fmt.Sprintf("Error: %v", m.state.Err)
	}
	if m.state.List.Title == "New Commit" {
		return appStyle.Render(m.newCommitView())
	}
	workspaceHeader := titleStyle.Render("Workspace: " + m.state.CurrentWorkspace)
	status := statusStyle.Render(m.state.StatusMessage)
	return appStyle.Render(workspaceHeader + "\n" + m.state.List.View() + "\n" + status)
}

func (m ui) newCommitView() string {
	return formBoxStyle.Render(fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s",
		titleStyle.Render("Create a new commit"),
		m.state.Inputs[0].View(),
		m.state.Inputs[1].View(),
		"(press enter to move/save, esc to cancel)",
	))
}

func (m *ui) createCommit() tea.Msg {
	newCommit := model.Commit{
		Type:      "ADD",
		Scope:     m.state.Inputs[0].Value(),
		MessageES: m.state.Inputs[1].Value(),
		Workspace: m.state.CurrentWorkspace,
	}
	if err := dbClient.CreateCommit(newCommit); err != nil {
		return err
	}
	m.state.List.Title = "Commit History"
	return commitActionMsg{message: "Commit Created!"}
}

func (m *ui) deleteCommit() tea.Msg {
	if i, ok := m.state.List.SelectedItem().(model.Commit); ok {
		if err := dbClient.DeleteCommit(i.ID); err != nil {
			return err
		}
		return commitActionMsg{message: "Commit Deleted!"}
	}
	return nil
}

func (m *ui) loadCommits() {
	commits, err := dbClient.GetCommits()
	if err != nil {
		m.state.Err = err
		return
	}
	items := make([]list.Item, len(commits))
	for i, c := range commits {
		items[i] = c
	}
	m.state.List.SetItems(items)
}
