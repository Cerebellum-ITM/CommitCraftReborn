package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// changelogConfigSavedMsg flows out when the user confirms the changelog
// configuration popup. The handler in update.go mirrors the values into
// the live config so the rest of the session sees the new state without
// a TUI restart.
type changelogConfigSavedMsg struct {
	enabled      bool
	path         string
	bumpStrategy string
	promptFile   string
	promptModel  string
	err          error
}

// closeChangelogConfigPopupMsg dismisses the popup without saving.
type closeChangelogConfigPopupMsg struct{}

const (
	changelogFieldEnabled = iota
	changelogFieldPath
	changelogFieldBumpStrategy
	changelogFieldPromptFile
	changelogFieldPromptModel
	changelogFieldCount
)

// ChangelogDetect carries the auto-detected defaults for the changelog
// configuration popup. Detection is best-effort — every field defaults
// to empty on failure so the popup keeps working on workspaces that
// have no CHANGELOG yet.
type ChangelogDetect struct {
	PathDetected          string // first CHANGELOG-ish file that exists
	LastVersion           string // most recent `## vX.Y.Z` heading
	SuggestedBumpStrategy string // "patch" by default
	Style                 string // "keep-a-changelog" | "headings" | "unknown"
}

// DetectChangelog runs the read-only probes against `pwd` and returns
// the changelog config defaults. Never errors.
func DetectChangelog(pwd string) ChangelogDetect {
	d := ChangelogDetect{SuggestedBumpStrategy: "patch", Style: "unknown"}
	candidates := []string{
		"CHANGELOG.md", "CHANGELOG", "HISTORY.md", "RELEASE_NOTES.md",
	}
	var path string
	for _, c := range candidates {
		p := filepath.Join(pwd, c)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			d.PathDetected = c
			path = p
			break
		}
	}
	if path == "" {
		return d
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return d
	}
	d.LastVersion = firstVersionHeading(string(raw))
	if strings.Contains(string(raw), "### Added") ||
		strings.Contains(string(raw), "### Changed") ||
		strings.Contains(string(raw), "### Fixed") {
		d.Style = "keep-a-changelog"
	} else {
		d.Style = "headings"
	}
	return d
}

var versionHeadingRegex = regexp.MustCompile(`(?m)^##\s+(v?\d+\.\d+\.\d+\S*)`)

func firstVersionHeading(content string) string {
	m := versionHeadingRegex.FindStringSubmatch(content)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// ChangelogConfigSnapshot is the surface the popup needs from the live
// config.ChangelogConfig. Mirrors the ReleaseConfigSnapshot pattern.
type ChangelogConfigSnapshot struct {
	Enabled      bool
	Path         string
	BumpStrategy string
	PromptFile   string
	PromptModel  string
}

type changelogConfigPopupModel struct {
	inputs   [changelogFieldCount]textinput.Model
	labels   [changelogFieldCount]string
	hints    [changelogFieldCount]string
	focus    int
	width    int
	height   int
	theme    *styles.Theme
	detected ChangelogDetect
}

func newChangelogConfigPopup(
	width, height int,
	theme *styles.Theme,
	current ChangelogConfigSnapshot,
	detected ChangelogDetect,
) changelogConfigPopupModel {
	m := changelogConfigPopupModel{
		width:    width,
		height:   height,
		theme:    theme,
		detected: detected,
	}
	sym := theme.AppSymbols()
	m.labels[changelogFieldEnabled] = sym.ConfigureChangelog + "  Enabled (true/false)"
	m.labels[changelogFieldPath] = sym.ConfigureChangelog + "  Path"
	m.labels[changelogFieldBumpStrategy] = sym.Tag + "  Bump strategy (patch/minor/major)"
	m.labels[changelogFieldPromptFile] = sym.CommitCraft + "  Prompt file"
	m.labels[changelogFieldPromptModel] = sym.CommitCraft + "  Prompt model"

	m.hints[changelogFieldEnabled] = "space to toggle · runs after the commit pipeline when on"
	m.hints[changelogFieldPath] = formatHint("Detected", detected.PathDetected)
	m.hints[changelogFieldBumpStrategy] = formatHint(
		fmt.Sprintf("Last version: %s · suggesting", emptyOr(detected.LastVersion, "none")),
		detected.SuggestedBumpStrategy,
	)
	m.hints[changelogFieldPromptFile] = "optional · file inside ~/.config/CommitCraft/prompts/ to override the built-in prompt"
	m.hints[changelogFieldPromptModel] = "optional · leave blank to inherit the active pipeline model"

	enabledPre := "false"
	if current.Enabled {
		enabledPre = "true"
	}
	pre := [changelogFieldCount]string{
		enabledPre,
		firstNonEmpty(current.Path, detected.PathDetected, "CHANGELOG.md"),
		firstNonEmpty(current.BumpStrategy, detected.SuggestedBumpStrategy),
		current.PromptFile,
		current.PromptModel,
	}
	for i := 0; i < changelogFieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = "  "
		ti.SetValue(pre[i])
		ti.SetCursor(len(pre[i]))
		m.inputs[i] = ti
	}
	m.focus = changelogFieldEnabled
	m.inputs[m.focus].Focus()
	return m
}

func (m changelogConfigPopupModel) Init() tea.Cmd { return nil }

func (m changelogConfigPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			return m, func() tea.Msg { return closeChangelogConfigPopupMsg{} }
		case "tab", "down":
			m = m.cycleFocus(+1)
			return m, nil
		case "shift+tab", "up":
			m = m.cycleFocus(-1)
			return m, nil
		case "enter":
			if m.focus < changelogFieldCount-1 {
				m = m.cycleFocus(+1)
				return m, nil
			}
			return m, m.save()
		case "ctrl+s":
			return m, m.save()
		case " ", "space":
			if m.focus == changelogFieldEnabled {
				cur := strings.ToLower(strings.TrimSpace(m.inputs[m.focus].Value()))
				next := "true"
				if cur == "true" {
					next = "false"
				}
				m.inputs[m.focus].SetValue(next)
				m.inputs[m.focus].SetCursor(len(next))
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
}

func (m changelogConfigPopupModel) cycleFocus(delta int) changelogConfigPopupModel {
	m.inputs[m.focus].Blur()
	m.focus = (m.focus + delta + changelogFieldCount) % changelogFieldCount
	m.inputs[m.focus].Focus()
	return m
}

func (m changelogConfigPopupModel) save() tea.Cmd {
	enabledRaw := strings.ToLower(
		strings.TrimSpace(m.inputs[changelogFieldEnabled].Value()),
	)
	enabled := enabledRaw == "true" || enabledRaw == "yes" || enabledRaw == "1"
	path := strings.TrimSpace(m.inputs[changelogFieldPath].Value())
	bump := strings.ToLower(
		strings.TrimSpace(m.inputs[changelogFieldBumpStrategy].Value()),
	)
	promptFile := strings.TrimSpace(m.inputs[changelogFieldPromptFile].Value())
	promptModel := strings.TrimSpace(m.inputs[changelogFieldPromptModel].Value())

	return func() tea.Msg {
		if bump != "" && bump != "patch" && bump != "minor" && bump != "major" {
			return changelogConfigSavedMsg{
				err: fmt.Errorf("bump_strategy must be one of patch/minor/major (got %q)", bump),
			}
		}
		if err := UpdateLocalConfigChangelog(
			enabled, path, bump, promptFile, promptModel,
		); err != nil {
			return changelogConfigSavedMsg{err: err}
		}
		return changelogConfigSavedMsg{
			enabled:      enabled,
			path:         path,
			bumpStrategy: bump,
			promptFile:   promptFile,
			promptModel:  promptModel,
		}
	}
}

func (m changelogConfigPopupModel) renderHelpFooter() string {
	help := m.theme.AppStyles().Help
	sep := help.ShortSeparator.Render(" · ")
	type entry struct{ k, d string }
	rows := []entry{
		{"tab/↓", "next"},
		{"shift+tab/↑", "prev"},
	}
	if m.focus == changelogFieldEnabled {
		rows = append(rows, entry{"space", "toggle"})
	}
	rows = append(rows,
		entry{"enter", "save (last field)"},
		entry{"ctrl+s", "save"},
		entry{"esc", "cancel"},
		entry{"ctrl+x", "quit"},
	)
	var parts []string
	for i, e := range rows {
		if i > 0 {
			parts = append(parts, sep)
		}
		parts = append(parts, help.ShortKey.Render(e.k), " ", help.ShortDesc.Render(e.d))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func (m changelogConfigPopupModel) View() tea.View {
	base := m.theme.AppStyles().Base
	sym := m.theme.AppSymbols()
	title := base.Foreground(m.theme.Secondary).Bold(true).
		Render(sym.ConfigureChangelog + "  Configure changelog")
	muted := base.Foreground(m.theme.FgMuted)
	label := base.Foreground(m.theme.FgBase).Bold(true)

	var rows []string
	rows = append(rows, title, "")
	for i := 0; i < changelogFieldCount; i++ {
		head := label.Render(m.labels[i])
		if i == m.focus {
			head = base.Foreground(m.theme.Accent).Bold(true).Render("▸ " + m.labels[i])
		}
		hint := ""
		if m.hints[i] != "" {
			hint = muted.Italic(true).Render(m.hints[i])
		}
		rows = append(rows, head, m.inputs[i].View(), hint, "")
	}
	rows = append(rows, m.renderHelpFooter())

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	box := lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocus).
		Render(body)
	return tea.NewView(box)
}

func emptyOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
