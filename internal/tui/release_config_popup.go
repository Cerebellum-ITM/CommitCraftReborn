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
	repository string
	branch     string
	version    string
	assetsPath string
	err        error
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
) releaseConfigPopupModel {
	m := releaseConfigPopupModel{
		width:    width,
		height:   height,
		theme:    theme,
		detected: detected,
		autoOpen: autoOpen,
	}
	m.labels[releaseFieldRepository] = "Repository (owner/name)"
	m.labels[releaseFieldBranch] = "Branch"
	m.labels[releaseFieldVersion] = "Version"
	m.labels[releaseFieldAssets] = "Binary assets path"
	m.labels[releaseFieldToken] = "GH_TOKEN"

	m.hints[releaseFieldRepository] = formatHint("Detected", detected.Repository)
	m.hints[releaseFieldBranch] = formatHint("Current branch", detected.Branch)
	m.hints[releaseFieldVersion] = formatVersionHint(detected.LastTag, detected.SuggestedVersion)
	m.hints[releaseFieldAssets] = formatHint("Detected", detected.AssetsPath)
	if detected.GhTokenSet {
		m.hints[releaseFieldToken] = "stored in ~/.config/CommitCraft/.env — leave blank to keep current"
	} else {
		m.hints[releaseFieldToken] = "not configured — required to upload to GitHub"
	}

	pre := [releaseFieldCount]string{
		firstNonEmpty(current.Repository, detected.Repository),
		firstNonEmpty(current.Branch, detected.Branch),
		firstNonEmpty(current.Version, detected.SuggestedVersion),
		firstNonEmpty(current.AssetsPath, detected.AssetsPath),
		"", // token always empty: never echo it back
	}

	for i := 0; i < releaseFieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = "  "
		ti.SetValue(pre[i])
		ti.SetCursor(len(pre[i]))
		if i == releaseFieldToken {
			ti.EchoMode = textinput.EchoPassword
			ti.Placeholder = "ghp_..."
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
	Repository string
	Branch     string
	Version    string
	AssetsPath string
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
				m.inputs[m.focus].SetValue(bumpDigitAtCursor(cur, m.inputs[m.focus].Position(), +1))
				return m, nil
			}
		case "ctrl+x":
			if m.focus == releaseFieldVersion {
				cur := m.inputs[m.focus].Value()
				m.inputs[m.focus].SetValue(bumpDigitAtCursor(cur, m.inputs[m.focus].Position(), -1))
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return m, cmd
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
	token := m.inputs[releaseFieldToken].Value()
	autoOpen := m.autoOpen

	return func() tea.Msg {
		if err := UpdateLocalConfigRelease(repo, branch, version, assets); err != nil {
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
			fromAutoOpen: autoOpen,
		}
	}
}

func (m releaseConfigPopupModel) View() tea.View {
	base := m.theme.AppStyles().Base
	title := base.Foreground(m.theme.Secondary).Bold(true).Render("Configure release")

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
		rows = append(rows, head, m.inputs[i].View(), hint, "")
	}

	footer := muted.Render(
		"tab/↓ next · shift+tab/↑ prev · enter save (last field) · ctrl+s save · esc cancel",
	)
	rows = append(rows, footer)
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
