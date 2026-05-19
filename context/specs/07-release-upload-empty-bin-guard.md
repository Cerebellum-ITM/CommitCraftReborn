# Unit 07: release-upload-empty-bin-guard (slim)

## Goal

Tell the user, via a `LevelInfo` status-bar message, when a GitHub release upload completes with zero asset files attached. The release shipping notes-only is a legitimate case — the user just shouldn't be left guessing whether their `BinaryAssetsPath` was misconfigured.

## Context — what the original audit asked for

The original Unit 07 spec listed two parts:

- (a) Pre-flight guard — if no expected binaries are present, abort with a status-bar message.
- (b) Pass release notes via stdin or temp file to `gh release create`, not as inline `-n <body>`.

**Both root causes (the `ARG_MAX` crash and the silent walk over the repo when `BinaryAssetsPath` is empty) were fixed by `v0.51.2` (`bd41cf7`)**:

- `internal/tui/release_upload.go:36-49` — `os.Stat` + `IsDir()` guard skips `filepath.Walk` when the path is empty or absent.
- `internal/tui/release_upload.go:72-78` — notes written to a `tmpFile` and passed via `--notes-file`, never as inline argv.
- `internal/tui/release_upload.go:82-83` — `exec.Command("gh", args...)` direct (no `sh -c`); `GH_TOKEN` injected via `cmd.Env` instead of argv.

The aborting behavior in (a) was *not* implemented and intentionally not: a release without assets is a valid use case (notes-only release). The remaining UX gap is just the lack of feedback that the upload happened with zero files.

## Design

When `UploadReleaseToGithub` finishes and `files` was empty, the TUI surfaces an extra `LevelInfo` status-bar line ("Release uploaded to GitHub · no asset files attached"). The existing success message ("The release was successfully uploaded to Github") stays as-is.

No new state, no behavior change to the upload itself.

## Implementation

### 1. Carry the no-assets signal back to Update

File: `internal/tui/release_upload.go`

`releaseUpdloadResultMsg` currently only carries `Err`. Add a `NoAssets bool` field so the result handler can branch on it:

```go
type releaseUpdloadResultMsg struct {
    Err      error
    NoAssets bool
}
```

In `UploadReleaseToGithub`, before returning nil, track whether `files` ended up empty and surface it through the wrapper. Cleanest path: change the function signature to return `(noAssets bool, err error)` and have `execUploadRelease` forward both into the message.

### 2. Show the info message in the handler

File: `internal/tui/update.go` — the existing `case releaseUpdloadResultMsg` block. After the current success path, if `msg.NoAssets`, emit a follow-up `LevelInfo` message via `WritingStatusBar.ShowMessageForDuration`. Keep the existing 2s success message, then queue the info one for ~3s after.

If chaining two timed messages is awkward, prefer the single combined message: `"Release uploaded to GitHub · no asset files attached"` (LevelSuccess), and drop the original separate line.

### 3. Documentation

- Bump `cmd/cli/main.go` from `v0.51.4` to `v0.52.0` (minor — user-visible status-bar copy change is part of the broader Unit 10 onboarding milestone; ships together).
- CHANGELOG entry lives in the combined v0.53.0 line described in Unit 10's spec (don't double-document this trivial slice).

## Out of scope

- The full pre-flight abort behavior in the original spec — that path conflicts with the legitimate notes-only release use case.
- Anything related to GH_TOKEN, repository, version, or assets configuration — that's Unit 10.

## Verify when done

- [ ] `go build ./...` passes.
- [ ] With `BinaryAssetsPath` unset (or pointing at an empty/missing directory) and a valid GH_TOKEN, "Create release in repository" completes, the release appears on GitHub with notes only, and the status bar shows "Release uploaded to GitHub · no asset files attached".
- [ ] With `BinaryAssetsPath` set to a directory containing files, the upload behaves as before and the status bar shows the original success message (no "no asset files" annotation).
- [ ] No regression in the error path: if `gh` fails, the existing error message is still surfaced.
