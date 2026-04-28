package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/api"
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

type modelPickerPopup struct {
	stage         config.ModelStage
	stageLabel    string
	current       string
	models        []storage.CachedModel
	cachedAt      time.Time
	table         table.Model
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
	innerW := max(40, width-6)
	// Each cell carries Padding(0,1) from table.DefaultStyles, so the
	// rendered width per column is colWidth + 2. Subtract that overhead
	// from the available inner width before splitting it across columns
	// — otherwise the header's bottom-border line wraps to a second row
	// because the joined cells are 2*numCols chars wider than the
	// viewport.
	const cellPadding = 2
	const numCols = 4
	currentW := 14
	ctxW := 10
	ownerW := 18
	idW := max(20, innerW-currentW-ctxW-ownerW-cellPadding*numCols)

	cols := []table.Column{
		{Title: "Model", Width: idW},
		{Title: "Owner", Width: ownerW},
		{Title: "Ctx", Width: ctxW},
		{Title: "Status", Width: currentW},
	}

	rows := make([]table.Row, 0, len(models))
	selected := 0
	for i, m := range models {
		ctx := "—"
		if m.ContextWindow > 0 {
			ctx = fmt.Sprintf("%dk", m.ContextWindow/1000)
		}
		status := ""
		if m.ID == current {
			status = "current"
			selected = i
		}
		rows = append(rows, table.Row{m.ID, m.OwnedBy, ctx, status})
	}

	tableH := max(6, height-12)

	st := table.DefaultStyles()
	st.Header = st.Header.
		Foreground(theme.Secondary).
		Bold(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(theme.Subtle)
	st.Cell = st.Cell.Foreground(theme.FgMuted)
	st.Selected = st.Selected.
		Foreground(theme.BG).
		Background(theme.Primary).
		Bold(true)

	tbl := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithWidth(innerW),
		table.WithHeight(tableH),
		table.WithStyles(st),
	)
	if len(rows) > 0 {
		tbl.SetCursor(selected)
	}

	return modelPickerPopup{
		stage:      stage,
		stageLabel: stageLabel,
		current:    current,
		models:     models,
		cachedAt:   cachedAt,
		table:      tbl,
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
		m.table, cmd = m.table.Update(msg)
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
		return m, func() tea.Msg { return closeModelPickerMsg{} }
	case "enter":
		row := m.table.SelectedRow()
		if len(row) == 0 {
			return m, nil
		}
		m.pickedModel = row[0]
		m.sub = pickerStateScope
		return m, nil
	case "r":
		stage := m.stage
		return m, func() tea.Msg { return modelPickerRefreshMsg{stage: stage} }
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m modelPickerPopup) View() tea.View {
	box := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	base := m.theme.AppStyles().Base
	header := base.Foreground(m.theme.Secondary).Bold(true).
		Render(fmt.Sprintf("Pick model · stage: %s", m.stageLabel))

	cacheLine := base.Foreground(m.theme.Muted).Render("cache: empty")
	if !m.cachedAt.IsZero() {
		age := time.Since(m.cachedAt).Round(time.Minute)
		cacheLine = base.Foreground(m.theme.Muted).
			Render(fmt.Sprintf("cache: %s ago · %d models", age, len(m.models)))
	}

	if m.sub == pickerStateScope {
		scopeHint := renderPopupHelpLine(m.theme, []helpEntry{
			{"g", "global  ~/.config/CommitCraft/config.toml"},
			{"l", "local   .commitcraft.toml"},
			{"esc", "back"},
		})
		body := lipgloss.JoinVertical(lipgloss.Left,
			header,
			"",
			base.Foreground(m.theme.FgBase).Render("Picked: "+m.pickedModel),
			"",
			base.Foreground(m.theme.FG).Render("Save to:"),
			"",
			scopeHint,
		)
		return tea.NewView(box.Render(body))
	}

	hint := renderPopupHelpLine(m.theme, []helpEntry{
		{"↑↓/jk", "navigate"},
		{"↵", "pick"},
		{"r", "refresh"},
		{"esc", "cancel"},
	})

	footer := renderModelQuotaFooter(m.theme, m.focusedModelID(), m.width-6)

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		cacheLine,
		"",
		m.table.View(),
		"",
		footer,
		"",
		hint,
	)
	return tea.NewView(box.Render(body))
}

// focusedModelID returns the model id under the table cursor, or the
// empty string when the table has no rows yet.
func (m modelPickerPopup) focusedModelID() string {
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return ""
	}
	return row[0]
}

// renderModelQuotaFooter draws a 4-row panel with the latest known
// RPM/RPD/TPM/TPD figures for modelID, fed by the in-memory rate-limit
// cache. Falls back to a "no telemetry yet" placeholder when no call has
// been recorded for the model in this session.
func renderModelQuotaFooter(theme *styles.Theme, modelID string, width int) string {
	base := theme.AppStyles().Base
	if modelID == "" {
		return base.Foreground(theme.Muted).Italic(true).
			Render("Move the cursor to inspect a model's quotas.")
	}

	header := base.Foreground(theme.Secondary).Bold(true).Render(modelID)

	raw, ok := api.GetRateLimits(modelID)
	if !ok {
		hint := base.Foreground(theme.Muted).Italic(true).Render(
			"No telemetry yet — make a call with this model to populate the quotas.",
		)
		return lipgloss.JoinVertical(lipgloss.Left, header, hint)
	}
	rl := api.EffectiveRateLimits(raw, time.Now())

	if width < 24 {
		width = 24
	}
	// REQ + TOK both derive from the captured `remaining-*` headers;
	// the Parsed flags guard against models whose response omitted a
	// header (would otherwise show a misleading 100% bar).
	var reqs, toks string
	if rl.RequestsParsed {
		reqs = renderThinQuotaBar(theme, "REQ",
			rl.LimitRequests-rl.RemainingRequests, rl.LimitRequests, width)
	} else {
		reqs = renderThinQuotaBar(theme, "REQ", 0, 0, width)
	}
	if rl.TokensParsed {
		toks = renderThinQuotaBar(theme, "TOK",
			rl.LimitTokens-rl.RemainingTokens, rl.LimitTokens, width)
	} else {
		toks = renderThinQuotaBar(theme, "TOK", 0, 0, width)
	}

	resetLine := base.Foreground(theme.Muted).Render(fmt.Sprintf(
		"resets in: requests %s · tokens %s",
		formatResetDuration(rl.ResetRequests),
		formatResetDuration(rl.ResetTokens),
	))
	captured := base.Foreground(theme.Subtle).Render(fmt.Sprintf(
		"captured %s ago",
		formatResetDuration(time.Since(rl.CapturedAt).Round(time.Second)),
	))

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		reqs,
		toks,
		resetLine,
		captured,
	)
}

// formatResetDuration prints a Groq-style reset window in a compact form.
func formatResetDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

// renderPopupHelpLine mirrors renderHelpEntries (in compose_status.go) so
// in-popup hints share the same themed key/desc styling as the global
// help bar — key = Primary, description = Muted, separated by `·`.
func renderPopupHelpLine(theme *styles.Theme, entries []helpEntry) string {
	base := theme.AppStyles().Base
	keyStyle := base.Foreground(theme.Primary)
	descStyle := base.Foreground(theme.Muted)
	parts := make([]string, 0, len(entries)*4)
	for i, e := range entries {
		parts = append(parts,
			keyStyle.Render(e.key),
			" ",
			descStyle.Render(e.desc),
		)
		if i < len(entries)-1 {
			parts = append(parts, "  ", descStyle.Render("·"), "  ")
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
