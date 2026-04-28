package tui

import (
	"fmt"
	"io"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/config"
	"commit_craft_reborn/internal/storage"
	"commit_craft_reborn/internal/tui/styles"
)

// closeModelPickerMsg dismisses the picker without persisting anything.
type closeModelPickerMsg struct{}

// modelPickerResultMsg fires after the user has picked both a model and a
// scope. The parent applies the change to the in-memory config and writes
// the chosen scope's TOML.
type modelPickerResultMsg struct {
	stage   config.ModelStage
	modelID string
	scope   config.ConfigScope
}

// modelPickerRefreshMsg asks the parent to re-fetch the catalogue from the
// Groq API. The parent rebuilds the popup with the fresh list when done.
type modelPickerRefreshMsg struct {
	stage config.ModelStage
}

type modelPickerSubState int

const (
	pickerStateChoosing modelPickerSubState = iota
	pickerStateScope
)

type modelPickerItem struct {
	id       string
	ownedBy  string
	contextW int
	current  bool
}

func (i modelPickerItem) Title() string       { return i.id }
func (i modelPickerItem) Description() string { return i.ownedBy }
func (i modelPickerItem) FilterValue() string { return i.id }

type modelPickerDelegate struct {
	theme *styles.Theme
}

func (d modelPickerDelegate) Height() int                             { return 1 }
func (d modelPickerDelegate) Spacing() int                            { return 0 }
func (d modelPickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d modelPickerDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(modelPickerItem)
	if !ok {
		return
	}
	base := d.theme.AppStyles().Base
	cursor := "  "
	titleColor := d.theme.FgMuted
	if index == m.Index() {
		cursor = base.Foreground(d.theme.Primary).Bold(true).Render("❯ ")
		titleColor = d.theme.FgBase
	}
	tag := ""
	if it.current {
		tag = base.Foreground(d.theme.Secondary).Render(" · current")
	}
	meta := ""
	if it.contextW > 0 {
		meta = base.Foreground(d.theme.Muted).Render(
			fmt.Sprintf("  %dk ctx", it.contextW/1000),
		)
	}
	line := cursor + base.Foreground(titleColor).Render(it.id) + meta + tag
	fmt.Fprint(w, line)
}

type modelPickerPopup struct {
	stage         config.ModelStage
	stageLabel    string
	current       string
	models        []storage.CachedModel
	cachedAt      time.Time
	list          list.Model
	width, height int
	theme         *styles.Theme
	sub           modelPickerSubState
	pickedModel   string
}

func newModelPickerPopup(
	stage config.ModelStage,
	stageLabel string,
	current string,
	models []storage.CachedModel,
	cachedAt time.Time,
	width, height int,
	theme *styles.Theme,
) modelPickerPopup {
	items := make([]list.Item, 0, len(models))
	selected := 0
	for i, m := range models {
		items = append(items, modelPickerItem{
			id:       m.ID,
			ownedBy:  m.OwnedBy,
			contextW: m.ContextWindow,
			current:  m.ID == current,
		})
		if m.ID == current {
			selected = i
		}
	}

	l := list.New(items, modelPickerDelegate{theme: theme}, width, height)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	if len(items) > 0 {
		l.Select(selected)
	}

	return modelPickerPopup{
		stage:      stage,
		stageLabel: stageLabel,
		current:    current,
		models:     models,
		cachedAt:   cachedAt,
		list:       l,
		width:      width,
		height:     height,
		theme:      theme,
		sub:        pickerStateChoosing,
	}
}

func (m modelPickerPopup) Init() tea.Cmd { return nil }

func (m modelPickerPopup) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if m.sub == pickerStateScope {
		switch km.String() {
		case "g":
			return m, func() tea.Msg {
				return modelPickerResultMsg{
					stage: m.stage, modelID: m.pickedModel, scope: config.ScopeGlobal,
				}
			}
		case "l":
			return m, func() tea.Msg {
				return modelPickerResultMsg{
					stage: m.stage, modelID: m.pickedModel, scope: config.ScopeLocal,
				}
			}
		case "esc":
			m.sub = pickerStateChoosing
			m.pickedModel = ""
			return m, nil
		}
		return m, nil
	}

	switch km.String() {
	case "esc":
		if m.list.FilterState() == list.Filtering {
			break
		}
		return m, func() tea.Msg { return closeModelPickerMsg{} }
	case "enter":
		if m.list.FilterState() == list.Filtering {
			break
		}
		it, ok := m.list.SelectedItem().(modelPickerItem)
		if !ok {
			return m, nil
		}
		m.pickedModel = it.id
		m.sub = pickerStateScope
		return m, nil
	case "r":
		if m.list.FilterState() == list.Filtering {
			break
		}
		stage := m.stage
		return m, func() tea.Msg { return modelPickerRefreshMsg{stage: stage} }
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m modelPickerPopup) View() tea.View {
	box := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	innerW := max(20, m.width-box.GetHorizontalFrameSize())
	innerH := max(8, m.height-box.GetVerticalFrameSize()-6)

	base := m.theme.AppStyles().Base
	header := base.Foreground(m.theme.Secondary).Bold(true).
		Render(fmt.Sprintf("Pick model · stage: %s", m.stageLabel))

	cacheLine := ""
	if !m.cachedAt.IsZero() {
		age := time.Since(m.cachedAt).Round(time.Minute)
		cacheLine = base.Foreground(m.theme.Muted).
			Render(fmt.Sprintf("cache: %s ago", age))
	} else {
		cacheLine = base.Foreground(m.theme.Muted).Render("cache: empty")
	}

	if m.sub == pickerStateScope {
		body := lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			base.Foreground(m.theme.FgBase).Render("Picked: "+m.pickedModel),
			"",
			base.Foreground(m.theme.FG).Render("Save to:"),
			"",
			base.Foreground(m.theme.Primary).Bold(true).Render("[g]")+
				base.Foreground(m.theme.FG).Render(" global  ~/.config/CommitCraft/config.toml"),
			base.Foreground(m.theme.Primary).Bold(true).Render("[l]")+
				base.Foreground(m.theme.FG).Render(" local   .commitcraft.toml in CWD"),
			"",
			base.Foreground(m.theme.Muted).Render("[esc] back"),
		)
		return tea.NewView(box.Render(body))
	}

	m.list.SetWidth(innerW)
	m.list.SetHeight(innerH)

	hint := base.Foreground(m.theme.Muted).
		Render("↑↓ navigate · / filter · enter pick · r refresh · esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		cacheLine,
		"",
		m.list.View(),
		"",
		hint,
	)
	return tea.NewView(box.Render(body))
}
