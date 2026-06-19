# demo — README GIF (VHS)

The animated GIF in the project README is recorded with
[charmbracelet/vhs](https://github.com/charmbracelet/vhs) from the `.tape`
scripts in this folder, so anyone can regenerate it.

It's a **real recording of the actual TUI** — not a mock-up. To stay
reproducible and leak nothing, the tape sources
[`setup-sandbox.sh`](setup-sandbox.sh), which:

- overrides `HOME` to a throwaway `/tmp/cc-demo` so the config dir, the SQLite
  drafts DB and the `.env` all live under `/tmp` (never your real
  `~/.config/CommitCraft`);
- writes **invented, non-functional** Groq keys;
- creates a throwaway git repo with one staged change to feed the pipeline;
- seeds the public model-context metadata so the pipeline view has data;
- starts [`mock-groq.py`](mock-groq.py), a tiny local stand-in for the Groq
  API, and points the app at it via the `COMMITCRAFT_GROQ_BASE_URL` override.

Because of the mock, the `^W` **generate** flow runs end to end with invented
content and **no network, no API key, no quota** — while the UI, colors,
glyphs, and multi-stage pipeline are exactly the real thing.

## Requirements

```sh
brew install vhs ttyd ffmpeg     # vhs needs ttyd + ffmpeg
```

A [Nerd Font](https://www.nerdfonts.com/) must be installed (the tape uses
`JetBrainsMono Nerd Font Mono`), plus `python3`, `sqlite3`, `curl` and `jq` on
`PATH`.

## Regenerate

Run from the **repo root** (tape paths are repo-relative):

```sh
vhs demo/tapes/hero.tape
```

Output lands in `demo/gifs/`. Shared settings + the sandbox prep live in
[`tapes/_setup.tape`](tapes/_setup.tape); each per-command tape `Source`s it.

## Files

| File                  | Purpose                                                       |
| --------------------- | ------------------------------------------------------------- |
| `tapes/_setup.tape`   | Shared VHS settings + hidden sandbox prep (`Source`d by tapes)|
| `tapes/hero.tape`     | The hero recording: new commit → type → scope → describe → ^W |
| `setup-sandbox.sh`    | Builds the offline sandbox (HOME, repo, keys, mock)           |
| `mock-groq.py`        | Local offline stand-in for the Groq API                       |
| `gifs/hero.gif`       | The rendered hero GIF embedded in the root README             |
