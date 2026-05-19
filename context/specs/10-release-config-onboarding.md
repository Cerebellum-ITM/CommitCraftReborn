# Unit 10: release-config-onboarding

## Goal

Replace the manual `release_config = { ... }` block in `.commitcraft.toml` with a guided in-TUI onboarding popup that:

1. Migrates `GH_TOKEN` out of the per-repo TOML into the global `~/.config/CommitCraft/.env` (where `GROQ_API_KEY` already lives), and removes it from the TOML on first read.
2. Auto-detects sensible defaults for the remaining fields (`repository` from `git remote`, `branch` from current HEAD, `version` from the last `git tag`, `binary_assets_path` from a `bin/` / `build/` / `dist/` scan).
3. Presents a multi-field popup so the user can review/edit those defaults and save them.
4. Auto-opens the popup before a release upload when any required field is missing.
5. Adds a command-palette entry "Configure release" so the popup is reachable on demand.
6. Refreshes the legacy `stateSettingAPIKey` view to reuse the same styled component as the new popup, so the two configuration surfaces look consistent.

## Context

### Current state

- `internal/config/types.go` — `ReleaseConfig` struct holds `Version`, `GhToken` (toml:`"GH_TOKEN"`), `Repository`, `BinaryAssetsPath`, `Branch`, `AutoBuild`, `BuildTool`, `BuildTarget`, plus the version-prefix/changelog knobs.
- `internal/config/loader.go:355-382` — `ResolveReleaseConfig` merges local TOML on top of global.
- `internal/config/loader.go:247-256` — `.env` is already loaded via `godotenv.Load`; `GROQ_API_KEY` lands in `globalCfg.TUI.GroqAPIKey`.
- `internal/tui/release_upload.go:60` — reads `config.ReleaseConfig.GhToken` direct from the struct.
- `internal/tui/local_config.go:81-95` — `saveAPIKeyToEnv` writes `GROQ_API_KEY=...\n` to `~/.config/CommitCraft/.env` (0o600). Only one key right now; we need to append/upsert.
- `internal/tui/update_apikey.go` — minimal handler. View renders inside `view.go:139-172` (`stateSettingAPIKey`) using a hand-rolled `boxStyle.Render`.
- `internal/tui/version_popup.go` — closest existing precedent for a styled form popup: `tea.Model` with `textinput.Model`, `Init/Update/View`, registered via `model.popup`, dismissed with `closeVersionPopupMsg`.

### Where `GH_TOKEN` is referenced

`grep -rn "GhToken\|GH_TOKEN" internal/`:

- `config/types.go:51` — struct tag.
- `config/types.go:210` — dummy default value `"ghp_123456789dummytoken"`.
- `tui/release_upload.go:60,83` — read at upload time, injected into `cmd.Env`.

That's it. Migration is small: change `ReleaseConfig.GhToken` from a TOML field to a runtime-only field populated by the env loader, and write a one-shot migrator that strips `GH_TOKEN` from any TOML it finds.

## Design

### Component A — env helpers (extend `local_config.go`)

`saveAPIKeyToEnv` becomes `saveEnvVar(name, value string) error` that:

- Reads `~/.config/CommitCraft/.env` if it exists, parses key=value lines into a map.
- Upserts the requested key (preserving the order of existing keys, appending the new key at the end).
- Writes back with mode 0o600.
- A second helper `readEnvVar(name string) string` (optional — `os.Getenv` already works because `godotenv.Load` runs at startup).

Then `saveAPIKeyToEnv` keeps the same exported signature but delegates to `saveEnvVar("GROQ_API_KEY", key)`. A new `saveGhTokenToEnv(token string)` mirrors it.

### Component B — config loader migration

`config/loader.go:LoadConfigs`:

- After `godotenv.Load(envPath)`, read `GH_TOKEN` from the environment. If non-empty, set `globalCfg.ReleaseConfig.GhToken` and mark `IsGhTokenSet = true` (new bool on `ReleaseConfig`, `toml:"-"`).
- If the TOML still has a non-empty `GhToken` (legacy), and the env var was empty: write the TOML value to `.env` via `saveGhTokenToEnv`, then clear `ReleaseConfig.GhToken` in memory **and on disk** (re-encode the global TOML without the field). Emit a startup log entry: "migrated GH_TOKEN from config.toml → .env".
- Strip the `GH_TOKEN` toml tag from the struct field and replace it with `toml:"-"` so the field is no longer serialized. This guarantees it never lands back in the file when the user hits "save" through the popup.

> If the user has `GH_TOKEN` in a *local* `.commitcraft.toml` (per-repo), the same migration runs against that file's path. Both writes are best-effort: log on failure, don't block startup.

### Component C — auto-detection helpers

New file: `internal/tui/release_config_detect.go`.

```go
type ReleaseDetect struct {
    Repository       string // owner/repo
    Branch           string
    LastTag          string
    SuggestedVersion string // BumpVersionPatch(LastTag) or "v0.1.0"
    AssetsPath       string // first of "bin", "build", "dist" that exists, else ""
    GhTokenSet       bool   // does .env already have GH_TOKEN?
}

func DetectRelease(pwd string) ReleaseDetect
```

Implementation:

- `Repository`: run `git -C pwd remote get-url origin`, parse `git@github.com:owner/repo.git` and `https://github.com/owner/repo.git` patterns into `owner/repo` (strip `.git`).
- `Branch`: `git -C pwd symbolic-ref --short HEAD`.
- `LastTag`: `git -C pwd describe --tags --abbrev=0` (already exists as `git.GetLastTag` somewhere — reuse if present).
- `SuggestedVersion`: `BumpVersionPatch(LastTag)` (already exists in `version_popup.go` helpers).
- `AssetsPath`: iterate `[]string{"bin", "build", "dist"}`, return first whose `filepath.Join(pwd, p)` `os.Stat` succeeds and `IsDir()`. Else "".
- `GhTokenSet`: `os.Getenv("GH_TOKEN") != ""`.

All git calls are best-effort: empty string on failure, no error returned. Detection is *advisory*; the popup shows the field empty if detection fails and the user fills it in.

### Component D — release config popup

New file: `internal/tui/release_config_popup.go`. Mirrors `version_popup.go`:

```go
type releaseConfigSavedMsg struct {
    cfg config.ReleaseConfig
    err error
}
type closeReleaseConfigPopupMsg struct{}

type releaseConfigPopupModel struct {
    inputs    []textinput.Model // ordered: repo, branch, version, assets, token
    labels    []string
    helps     []string          // per-field placeholder/help text
    focus     int               // which input is active
    masked    map[int]bool      // token field gets EchoPassword
    width, height int
    theme     *styles.Theme
    detected  ReleaseDetect     // for "Detected: ..." hints
}
```

Behavior:

- `Init` → focus first input, prefill from existing `config.ReleaseConfig` + `ReleaseDetect` (existing values take precedence; detection fills the blanks).
- `Update`:
  - `tab` / `shift+tab` → cycle `focus`, blur/focus the right `textinput`.
  - `enter` (on any field that isn't the last) → advance focus (don't save).
  - `enter` (on last field, OR `ctrl+enter` anywhere) → save: write the config back to `.commitcraft.toml` via a new `UpdateLocalConfigRelease(rc)`, write the GH_TOKEN via `saveGhTokenToEnv`, return `releaseConfigSavedMsg{cfg, err}`.
  - `esc` → `closeReleaseConfigPopupMsg`.
  - `ctrl+a` / `ctrl+x` on the version field → reuse `bumpDigitAtCursor`.
- `View` → vertical layout: title, a "detected" annotation panel under each field showing what `DetectRelease` proposed (or "no value detected"), the inputs themselves, hint footer.

Styling: shared with the refreshed API-key popup (see Component F).

### Component E — `UpdateLocalConfigRelease` helper

New function in `internal/tui/local_config.go` (or `config/saver.go`, whichever fits). Pattern matches `UpdateLocalConfigVersion`:

- Ensure `.commitcraft.toml` exists.
- Decode it, overwrite `cfg.ReleaseConfig.Repository / Branch / Version / BinaryAssetsPath`.
- **Do not write `GhToken`** — the struct tag is already `toml:"-"`.
- Re-encode and write back at mode 0o644.

### Component F — refresh `stateSettingAPIKey` to use the new component

The hand-rolled `stateSettingAPIKey` view in `view.go:139-172` becomes a popup model (`api_key_popup.go`) that follows the same shape as `releaseConfigPopupModel` (single field, the same rounded box, the same title/hint conventions). The state machine stays — `stateSettingAPIKey` still exists — but `View()` for that state now renders `model.apiKeyPopup.View()` (or calls a shared `renderConfigBox` helper). This is the smallest possible refactor: extract `renderConfigBox(theme, title, fields, hint, width, height)` into a helper used by both popups, and have `view.go` call it for `stateSettingAPIKey`.

### Component G — wire auto-open and command palette

- **Auto-open**: in `update.go`'s "Create release in repository" path, before opening `openVersionEditor`, call `config.HasReleaseEssentials(globalCfg)` (new helper that checks repo + version + token). If false, open the release config popup first; once saved, fall through to the version editor.
- **Command palette**: extend `cmdGenerateLocalConfig`'s neighborhood in `update.go:211` with a new `cmdConfigureRelease` constant + `commandRunMsg` case that opens the popup.

### Where the popup is hosted

`model.popup` already drives every other popup. Add a `releaseConfigPopupModel` branch to the type-switch in `view.go:276-326`. Make `Update` route `releaseConfigPopupModel` messages to `model.popup.Update(msg)` like the others.

## Implementation order

1. **Loader migration + env helpers + ReleaseConfig struct cleanup** (smallest blast radius — just config + local_config files).
2. **Auto-detection helpers** (`release_config_detect.go`, pure functions; testable on /tmp repo).
3. **`UpdateLocalConfigRelease`** (mirror `UpdateLocalConfigVersion`).
4. **Release config popup** (`release_config_popup.go` + wire into `model.popup` switch + view dispatcher).
5. **API key popup refresh** (extract `renderConfigBox`, swap `view.go` over).
6. **Auto-open wiring + command palette entry**.
7. **Manual smoke test on `/tmp/cc-test/repo`**.

## Documentation

- Bump `cmd/cli/main.go` from `v0.51.4` to `v0.53.0` (minor — new user-facing onboarding surface + token migration is a user-visible behavior change).
- CHANGELOG entry under `## v0.53.0 — 2026-05-19`. Combine Unit 07's status-bar tweak and Unit 10 into one entry — both ship together on this branch. Include a `### Usage` block covering: (a) GH_TOKEN now lives in `.env`, (b) how to open the config popup (auto / palette), (c) the status-bar info on notes-only uploads.

## Out of scope

- Refactoring the whole config-popup family (`config_popup.go`) — only the API key view is rebuilt to share the new look.
- Encrypting `.env` or moving to OS keychain — discussed and deferred to a future unit.
- Multi-repository GH_TOKEN scoping (one token per remote, etc.). Single global token for now.
- Validating that the GH_TOKEN actually authenticates against GitHub before saving. Out-of-band — `gh release create` will fail loud if the token is bad.

## Verify when done

- [ ] `go build ./...` passes, `go vet ./...` clean.
- [ ] On a repo whose `.commitcraft.toml` contains a `GH_TOKEN=...` line: starting `commitcraft` migrates the token to `~/.config/CommitCraft/.env`, removes the field from the TOML, and logs the migration. Subsequent starts find the token via env and don't re-migrate.
- [ ] Command palette → "Configure release" opens the popup. Detected defaults are pre-filled (or blank with a "no value detected" hint). Tab cycles fields. Enter on the last field (or Ctrl+Enter anywhere) saves. Esc closes without saving.
- [ ] Attempting to upload a release with missing repo/version auto-opens the config popup, and after saving, the upload proceeds.
- [ ] The `stateSettingAPIKey` screen visually matches the new release config popup (same border, padding, title placement, hint style).
- [ ] `.commitcraft.toml` after saving from the popup contains the repo/branch/version/assets-path values but **no `GH_TOKEN`**.
- [ ] `~/.config/CommitCraft/.env` permissions are `0o600`.
- [ ] `CHANGELOG.md` has the `v0.53.0` entry and `cmd/cli/main.go` is bumped.
