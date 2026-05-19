# Unit 09: release-upload-loading-panel-stale

## Goal

After the user confirms "Create release in repository" and the GitHub upload succeeds, the loading panel ("Loading releases / resolving commit subjects…") must disappear and the copy shown during the in-flight upload must describe the upload — not the release-history sync that isn't even running.

## Root cause

`update.go:594` (`case "Create release in Github"`) calls `createRelease(model)` and **discards the returned `(tea.Model, tea.Cmd)`**. That returned `cmd` is the only thing that dispatches `startReleaseHistorySync` (via `enterReleaseHistoryLoading`). The model mutations from `createRelease` persist (because `model` is a pointer): `state = stateReleaseMainMenu` and `releaseLoading = true`. But because the cmd was thrown away, **no sync ever runs**, so no `releaseHistorySyncMsg` ever arrives, so `update.go:248-265` never flips `releaseLoading` back to false. After the version popup closes and the upload finishes, the panel stays on screen forever.

Secondary problem: even if the sync did run, the panel's title is `"Loading releases"` and subtitle `"resolving commit subjects…"` (`release_loading.go:35-36`). That's correct copy for the history-sync phase, but during the in-flight upload it's misleading — the user sees it while the system is actually pushing to GitHub.

## Design

No new visual elements. The existing loading panel rendered by `renderReleaseLoading` is reused but gets two presentations driven by two distinct flags:

- `releaseLoading` (existing) — async release-history sync in flight. Copy stays "Loading releases / resolving commit subjects…".
- `releaseUploading` (new) — release build and/or GitHub upload in flight. Copy switches to "Uploading release to GitHub / pushing assets…".

When both are true, the upload copy wins. When neither is true, the panel is not shown.

## Implementation

### 1. Preserve the loadCmd from `createRelease`

File: `internal/tui/update.go`, around line 594.

```go
case "Create release in Github":
    ...
    createRelease(model)
    release, err := model.db.GetLatestRelease(model.pwd)
    ...
```

becomes:

```go
case "Create release in Github":
    ...
    _, loadCmd := createRelease(model)
    release, err := model.db.GetLatestRelease(model.pwd)
    ...
    model.popup = openVersionEditor(model)
    return model, loadCmd
```

This fires the release-history sync that `createRelease` already prepared, so `releaseLoading` clears the moment the sync resolves — the original design intent.

### 2. Add `releaseUploading` flag

File: `internal/tui/model.go`, near the existing `releaseLoading` field (around line 203):

```go
// releaseUploading is set while the GitHub build/upload pipeline is in
// flight (between versionUpdatedMsg dispatching execReleaseBuild /
// execUploadRelease and the matching result message arriving).
// renderReleaseLoading reads it to swap the panel copy onto the upload
// phase; view.go gates the panel on releaseLoading || releaseUploading.
releaseUploading bool
```

No default initialization needed (`bool` zero value is `false`).

### 3. Drive the new flag from the upload pipeline

File: `internal/tui/update.go`.

- In `case versionUpdatedMsg:` (around line 413), when the version-editor confirms and the upload chain starts, set `model.releaseUploading = true` before dispatching `execReleaseBuild` or `execUploadRelease`.
- In `case releaseBuildResultMsg:` (around line 863), clear `model.releaseUploading = false` on the error branch (the chain stops there). Leave it `true` on the success branch (the upload step continues).
- In `case releaseUpdloadResultMsg:` (around line 890), clear `model.releaseUploading = false` on **both** error and success branches.
- In `case closeVersionPopupMsg:` (around line 388), when the user cancels and `pendingReleaseUpload` is non-nil, clear `model.releaseUploading = false` for symmetry (currently false anyway, but explicit).

### 4. Gate the panel on either flag

File: `internal/tui/view.go`, around line 196.

```go
if model.releaseLoading {
    mainContent = model.renderReleaseLoading(histW, histH)
    break
}
```

becomes:

```go
if model.releaseLoading || model.releaseUploading {
    mainContent = model.renderReleaseLoading(histW, histH)
    break
}
```

### 5. Switch the panel copy

File: `internal/tui/release_loading.go`.

`renderReleaseLoading` reads `model.releaseUploading` and chooses the strings:

```go
title := "Loading releases"
subtitle := "resolving commit subjects…"
if model.releaseUploading {
    title = "Uploading release to GitHub"
    subtitle = "building & pushing assets…"
}
```

The rest of the panel (spinner glyph, workspace hint, box chrome) stays untouched.

### 6. Documentation

- Bump `cmd/cli/main.go` version from `v0.51.3` to `v0.51.4`.
- Add a `## v0.51.4 — 2026-05-13` entry to `CHANGELOG.md` describing both the panel persistence fix and the copy disambiguation. No `### Usage` subsection — internal bug fix.

## Out of scope

- Refactoring the release-creation flow to not transition into `stateReleaseMainMenu` until the upload finishes (would require deeper state-machine surgery; Unit 06 territory).
- Adding a "release just uploaded" success view in place of the dual panel (could surface as Unit 06's `converge-report-screen` work).
- Touching the `closeVersionPopupMsg` cancel-path UX beyond clearing `releaseUploading` (the `"GitHub release cancelled"` toast already exists).

## Verify when done

- [ ] `go build ./...` passes.
- [ ] From the release flow, complete the pipeline → confirm "Create release in repository" → confirm version. While the upload is in flight, the panel reads "Uploading release to GitHub / building & pushing assets…", not "Loading releases".
- [ ] When `releaseUpdloadResultMsg` arrives (success), the panel disappears and the underlying release main view shows. Status bar reads "The release was successfully uploaded to Github".
- [ ] If the upload errors, the panel also disappears and the status bar surfaces the error.
- [ ] Booting straight into Release mode (e.g. `commitcraft -r`) still shows the original "Loading releases / resolving commit subjects…" copy during the initial history sync.
- [ ] `cmd/cli/main.go` is bumped to `v0.51.4` and `CHANGELOG.md` has the new entry on top.
