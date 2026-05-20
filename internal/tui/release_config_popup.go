package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"commit_craft_reborn/internal/tui/styles"
)

// releaseConfigSavedMsg is emitted after the popup persisted the new
// release configuration (TOML + .env). On success the upload pipeline
// can be resumed automatically; on failure the status bar surfaces
// the wrapped error.
type releaseConfigSavedMsg struct {
	repository  string
	branch      string
	version     string
	assetsPath  string
	autoBuild   bool
	buildTool   string
	buildTarget string
	err         error
	// FromAutoOpen reports whether the popup was opened because the
	// upload path detected missing config. When true, the handler in
	// update.go resumes the upload by chaining into the version
	// editor once this msg arrives with err == nil.
	fromAutoOpen bool
}

// closeReleaseConfigPopupMsg dismisses the popup without saving.
type closeReleaseConfigPopupMsg struct{}

const (
	releaseFieldRepository = iota
	releaseFieldBranch
	releaseFieldVersion
	releaseFieldAssets
	releaseFieldAutoBuild
	releaseFieldBuildTool
	releaseFieldBuildTarget
	releaseFieldToken
	releaseFieldCount
)

// releaseConfigPopupModel is a Tab-navigable multi-field form. Mirrors
// the pattern in version_popup.go: one tea.Model satellite that owns
// its inputs, returns a typed msg on save/close, and is hosted by
// model.popup like every other popup.
type releaseConfigPopupModel struct {
	inputs    [releaseFieldCount]textinput.Model
	labels    [releaseFieldCount]string
	hints     [releaseFieldCount]string
	focus     int
	width     int
	height    int
	theme     *styles.Theme
	detected  ReleaseDetect
	autoOpen  bool
	saveError string
	// picker holds the in-popup list-picker state used by the
	// build_tool and build_target fields. When pickerActive is true,
	// the popup intercepts arrow keys / enter / esc for navigation
	// instead of forwarding them to the focused textinput.
	pickerActive  bool
	pickerField   int
	pickerOptions []string
	pickerIndex   int
	// buildToolChoices / buildTargetChoices are cached at popup
	// creation time so Enter doesn't have to re-read disk every time
	// the picker opens.
	buildToolChoices   []string
	buildTargetChoices []string
}

// newReleaseConfigPopup builds the popup with detected defaults merged
// on top of any values already present in the live ReleaseConfig.
// `autoOpen` flags that the popup was triggered by the upload path
// detecting missing config — used by the saved msg so the caller can
// resume the upload after the user finishes.
func newReleaseConfigPopup(
	width, height int,
	theme *styles.Theme,
	current ReleaseConfigSnapshot,
	detected ReleaseDetect,
	autoOpen bool,
	buildToolChoices []string,
	buildTargetChoices []string,
) releaseConfigPopupModel {
	m := releaseConfigPopupModel{
		width:              width,
		height:             height,
		theme:              theme,
		detected:           detected,
		autoOpen:           autoOpen,
		buildToolChoices:   buildToolChoices,
		buildTargetChoices: buildTargetChoices,
	}
	sym := theme.AppSymbols()
	m.labels[releaseFieldRepository] = sym.BranchIcon + "  Repository (owner/name)"
	m.labels[releaseFieldBranch] = sym.BranchIcon + "  Branch"
	m.labels[releaseFieldVersion] = sym.Tag + "  Version"
	m.labels[releaseFieldAssets] = sym.NewDbRecord + "  Binary assets path"
	m.labels[releaseFieldAutoBuild] = sym.BuildTool + "  Auto build (true/false)"
	m.labels[releaseFieldBuildTool] = sym.BuildTool + "  Build tool"
	m.labels[releaseFieldBuildTarget] = sym.BuildTool + "  Build target"
	m.labels[releaseFieldToken] = sym.TokenIcon + "  GH_TOKEN"

	m.hints[releaseFieldRepository] = formatHint("Detected", detected.Repository)
	m.hints[releaseFieldBranch] = formatHint("Current branch", detected.Branch)
	m.hints[releaseFieldVersion] = formatVersionHint(detected.LastTag, detected.SuggestedVersion)
	m.hints[releaseFieldAssets] = formatHint("Detected", detected.AssetsPath)
	m.hints[releaseFieldAutoBuild] = "space to toggle · runs the configured build target before upload"
	m.hints[releaseFieldBuildTool] = formatHint("Detected", detected.BuildTool)
	m.hints[releaseFieldBuildTarget] = formatHint("Detected", detected.BuildTarget)
	if detected.GhTokenSet {
		m.hints[releaseFieldToken] = "stored in ~/.config/CommitCraft/.env — leave blank to keep current"
	} else {
		m.hints[releaseFieldToken] = "not configured — required to upload to GitHub"
	}

	autoBuildPre := "false"
	if current.AutoBuild {
		autoBuildPre = "true"
	}
	pre := [releaseFieldCount]string{
		firstNonEmpty(current.Repository, detected.Repository),
		firstNonEmpty(current.Branch, detected.Branch),
		firstNonEmpty(current.Version, detected.SuggestedVersion),
		firstNonEmpty(current.AssetsPath, detected.AssetsPath),
		autoBuildPre,
		firstNonEmpty(current.BuildTool, detected.BuildTool),
		firstNonEmpty(current.BuildTarget, detected.BuildTarget),
		"", // token always empty: never echo it back
	}

	for i := 0; i < releaseFieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = "  "
		ti.SetValue(pre[i])
		ti.SetCursor(len(pre[i]))
		if i == releaseFieldToken {
			ti.EchoMode = textinput.EchoPassword
			// Default is already '*' but set it explicitly so a future
			// bubbles upgrade that drops the default never regresses
			// to plain text.
			ti.EchoCharacter = '*'
			// NO placeholder. bubbles renders the placeholder by copying
			// up to `Width()+1` runes from it; without an explicit
			// Width(), only the first rune is painted, so "ghp_..."
			// shows as a lone `g` and looks identical to a typed char
			// that failed to mask. The hint label below the input
			// already explains what to enter.
		}
		m.inputs[i] = ti
	}
	m.focus = releaseFieldRepository
	m.inputs[m.focus].Focus()
	return m
}

// ReleaseConfigSnapshot is the minimum surface the popup needs from
// the live `config.ReleaseConfig`. Kept here so the popup file never
// imports the `config` package and the tests can pass a literal.
type ReleaseConfigSnapshot struct {
	Repository  string
	Branch      string
	Version     string
	AssetsPath  string
	AutoBuild   bool
	BuildTool   string
	BuildTarget string
}

// hasReleaseEssentials reports whether the live ReleaseConfig has the
// minimum needed for an upload — a repository and a token. Version is
// optional here because the version editor pops up next and already
// nudges the user to confirm/bump it. BinaryAssetsPath and Branch are
// both optional (notes-only releases on the current branch are valid).
func hasReleaseEssentials(repository string, tokenSet bool) bool {
	return strings.TrimSpace(repository) != "" && tokenSet
}

func (m releaseConfigPopupModel) Init() tea.Cmd { return nil }

func (m releaseConfigPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		// Picker mode: intercept navigation keys before the textinput
		// gets a chance to swallow them. Esc cancels without committing
		// the highlighted option, Enter commits.
		if m.pickerActive {
			switch km.String() {
			case "esc":
				m.pickerActive = false
				return m, nil
			case "enter":
				if len(m.pickerOptions) > 0 {
					choice := m.pickerOptions[m.pickerIndex]
					m.inputs[m.pickerField].SetValue(choice)
					m.inputs[m.pickerField].SetCursor(len(choice))
				}
				m.pickerActive = false
				return m, nil
			case "up", "k":
				if len(m.pickerOptions) > 0 {
					m.pickerIndex = (m.pickerIndex - 1 + len(m.pickerOptions)) %
						len(m.pickerOptions)
				}
				return m, nil
			case "down", "j":
				if len(m.pickerOptions) > 0 {
					m.pickerIndex = (m.pickerIndex + 1) % len(m.pickerOptions)
				}
				return m, nil
			}
			// Any other key while the picker is open is ignored —
			// textinput edits don't apply mid-pick.
			return m, nil
		}
		switch km.String() {
		case "esc":
			return m, func() tea.Msg { return closeReleaseConfigPopupMsg{} }
		case "tab", "down":
			m = m.cycleFocus(+1)
			return m, nil
		case "shift+tab", "up":
			m = m.cycleFocus(-1)
			return m, nil
		case "enter":
			// Build tool / target fields open the in-popup list
			// picker on Enter instead of advancing focus. Saves the
			// user from typing exact target names.
			if m.focus == releaseFieldBuildTool {
				m = m.openPicker(releaseFieldBuildTool, m.buildToolChoices)
				return m, nil
			}
			if m.focus == releaseFieldBuildTarget {
				m = m.openPicker(releaseFieldBuildTarget, m.buildTargetChoices)
				return m, nil
			}
			if m.focus < releaseFieldCount-1 {
				m = m.cycleFocus(+1)
				return m, nil
			}
			return m, m.save()
		case "ctrl+s":
			return m, m.save()
		case "ctrl+a":
			if m.focus == releaseFieldVersion {
				cur := m.inputs[m.focus].Value()
				m.inputs[m.focus].SetValue(
					bumpDigitAtCursor(cur, m.inputs[m.focus].Position(), +1),
				)
				return m, nil
			}
		case " ", "space":
			// Space toggles the Auto build field. For every other
			// field we want the space character to land in the
			// textinput normally, so fall through.
			if m.focus == releaseFieldAutoBuild {
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

// openPicker arms the in-popup list picker for `field`. The picker
// starts highlighted on whichever option matches the current field
// value (so the user sees the existing pick as the cursor); otherwise
// it lands on index 0. Empty option lists are tolerated — the picker
// just renders "no options" and Enter is a no-op until the user Escs.
func (m releaseConfigPopupModel) openPicker(
	field int, options []string,
) releaseConfigPopupModel {
	m.pickerActive = true
	m.pickerField = field
	m.pickerOptions = options
	m.pickerIndex = 0
	cur := strings.TrimSpace(m.inputs[field].Value())
	for i, opt := range options {
		if opt == cur {
			m.pickerIndex = i
			break
		}
	}
	return m
}

func (m releaseConfigPopupModel) cycleFocus(delta int) releaseConfigPopupModel {
	m.inputs[m.focus].Blur()
	m.focus = (m.focus + delta + releaseFieldCount) % releaseFieldCount
	m.inputs[m.focus].Focus()
	return m
}

func (m releaseConfigPopupModel) save() tea.Cmd {
	repo := strings.TrimSpace(m.inputs[releaseFieldRepository].Value())
	branch := strings.TrimSpace(m.inputs[releaseFieldBranch].Value())
	version := strings.TrimSpace(m.inputs[releaseFieldVersion].Value())
	assets := strings.TrimSpace(m.inputs[releaseFieldAssets].Value())
	autoBuildRaw := strings.ToLower(
		strings.TrimSpace(m.inputs[releaseFieldAutoBuild].Value()),
	)
	autoBuild := autoBuildRaw == "true" || autoBuildRaw == "yes" || autoBuildRaw == "1"
	buildTool := strings.TrimSpace(m.inputs[releaseFieldBuildTool].Value())
	buildTarget := strings.TrimSpace(m.inputs[releaseFieldBuildTarget].Value())
	token := m.inputs[releaseFieldToken].Value()
	autoOpen := m.autoOpen

	return func() tea.Msg {
		if err := UpdateLocalConfigRelease(
			repo, branch, version, assets, autoBuild, buildTool, buildTarget,
		); err != nil {
			return releaseConfigSavedMsg{err: err, fromAutoOpen: autoOpen}
		}
		if strings.TrimSpace(token) != "" {
			if err := SaveGhTokenToEnv(strings.TrimSpace(token)); err != nil {
				return releaseConfigSavedMsg{err: err, fromAutoOpen: autoOpen}
			}
		}
		return releaseConfigSavedMsg{
			repository:   repo,
			branch:       branch,
			version:      version,
			assetsPath:   assets,
			autoBuild:    autoBuild,
			buildTool:    buildTool,
			buildTarget:  buildTarget,
			fromAutoOpen: autoOpen,
		}
	}
}

// renderHelpFooter renders the popup's bottom hint line through the
// project-wide help styles (ShortKey / ShortDesc / ShortSeparator) so
// the popup matches the look of every other on-screen help row. The
// advertised keys change depending on which field is focused and
// whether the in-popup picker is active.
func (m releaseConfigPopupModel) renderHelpFooter() string {
	help := m.theme.AppStyles().Help
	sep := help.ShortSeparator.Render(" · ")
	type entry struct{ k, d string }
	var rows []entry
	switch {
	case m.pickerActive:
		rows = []entry{
			{"↑/k", "up"},
			{"↓/j", "down"},
			{"enter", "pick"},
			{"esc", "cancel pick"},
			{"ctrl+x", "quit"},
		}
	default:
		rows = []entry{
			{"tab/↓", "next"},
			{"shift+tab/↑", "prev"},
		}
		switch m.focus {
		case releaseFieldAutoBuild:
			rows = append(rows, entry{"space", "toggle"})
		case releaseFieldBuildTool, releaseFieldBuildTarget:
			rows = append(rows, entry{"enter", "pick from list"})
		default:
			rows = append(rows, entry{"enter", "save (last field)"})
		}
		rows = append(rows,
			entry{"ctrl+s", "save"},
			entry{"esc", "cancel"},
			entry{"ctrl+x", "quit"},
		)
	}
	var parts []string
	for i, e := range rows {
		if i > 0 {
			parts = append(parts, sep)
		}
		parts = append(parts, help.ShortKey.Render(e.k), " ", help.ShortDesc.Render(e.d))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// renderFieldPicker renders the inline list-picker overlay used by the
// build_tool and build_target fields. Replaces the textinput row so the
// user keeps spatial context: the picker sits exactly where the value
// would be.
func (m releaseConfigPopupModel) renderFieldPicker() string {
	base := m.theme.AppStyles().Base
	if len(m.pickerOptions) == 0 {
		return base.Foreground(m.theme.FgMuted).
			Italic(true).
			Render("  (no options detected · esc to dismiss)")
	}
	idle := base.Foreground(m.theme.FgBase)
	active := base.Foreground(m.theme.Accent).Bold(true)
	var rows []string
	for i, opt := range m.pickerOptions {
		if i == m.pickerIndex {
			rows = append(rows, active.Render("  ▸ "+opt))
		} else {
			rows = append(rows, idle.Render("    "+opt))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m releaseConfigPopupModel) View() tea.View {
	base := m.theme.AppStyles().Base
	sym := m.theme.AppSymbols()
	title := base.Foreground(m.theme.Secondary).Bold(true).
		Render(sym.ConfigureRelease + "  Configure release")

	muted := base.Foreground(m.theme.FgMuted)
	label := base.Foreground(m.theme.FgBase).Bold(true)

	var rows []string
	rows = append(rows, title, "")
	for i := 0; i < releaseFieldCount; i++ {
		head := label.Render(m.labels[i])
		if i == m.focus {
			head = base.Foreground(m.theme.Accent).Bold(true).Render("▸ " + m.labels[i])
		}
		hint := ""
		if m.hints[i] != "" {
			hint = muted.Italic(true).Render(m.hints[i])
		}
		body := m.inputs[i].View()
		if m.pickerActive && i == m.pickerField {
			body = m.renderFieldPicker()
		}
		rows = append(rows, head, body, hint, "")
	}

	rows = append(rows, m.renderHelpFooter())
	if m.saveError != "" {
		errLine := base.Foreground(m.theme.Error).Render("error: " + m.saveError)
		rows = append(rows, "", errLine)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	box := lipgloss.NewStyle().
		Width(m.width).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderFocus).
		Render(body)
	return tea.NewView(box)
}

func formatHint(prefix, value string) string {
	if value == "" {
		return "no value detected"
	}
	return fmt.Sprintf("%s: %s", prefix, value)
}

func formatVersionHint(lastTag, suggestion string) string {
	if lastTag == "" {
		return fmt.Sprintf("no tags found · suggesting %s", suggestion)
	}
	return fmt.Sprintf("Last tag: %s · suggesting %s", lastTag, suggestion)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
