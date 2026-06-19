# demo — README GIFs (VHS)

The animated GIFs in the project README are recorded with
[charmbracelet/vhs](https://github.com/charmbracelet/vhs) from the `.tape`
scripts in this folder, so anyone can regenerate them.

These are **real recordings of the actual binary** — not simulations. To stay
reproducible and leak nothing, every tape sources [`setup-sandbox.sh`](setup-sandbox.sh),
which:

- overrides `HOME` to a throwaway `/tmp/cc-demo` so the config dir, the SQLite
  drafts DB and the `.env` all live under `/tmp` (never your real
  `~/.config/CommitCraft`);
- writes **invented, non-functional** Groq keys (so `ai key show` reports both
  slots set without exposing anything);
- creates a throwaway git repo with one staged change to feed the pipeline;
- seeds the public model-context metadata so `ai context` computes a real fit.

Because of this, the GIFs make **no Groq calls** — the hero uses delegate mode
(`ai generate --agent` → `ai submit`), so the whole flow runs offline.

## Requirements

```sh
brew install vhs ttyd ffmpeg     # vhs needs ttyd + ffmpeg
```

A [Nerd Font](https://www.nerdfonts.com/) must be installed (the tapes use
`JetBrainsMono Nerd Font Mono`) and `jq` + `sqlite3` on `PATH`.

## Regenerate

Run from the **repo root** (tape paths are repo-relative):

```sh
vhs demo/tapes/hero.tape              # one
for t in demo/tapes/*.tape; do        # all (skip the shared _setup.tape)
  case "$(basename "$t")" in _*) continue;; esac
  vhs "$t"
done
```

Output lands in `demo/gifs/`. Shared settings + the sandbox prep live in
[`tapes/_setup.tape`](tapes/_setup.tape); each per-command tape `Source`s it.
To add a command: create `demo/tapes/<cmd>.tape`, render, and embed the new GIF
in the root `README.md`.

## Tapes

| Tape           | GIF            | Shows                                                       |
| -------------- | -------------- | ----------------------------------------------------------- |
| `hero.tape`    | `hero.gif`     | Delegate-mode flow: prompt bundle → `ai submit` → commit    |
| `context.tape` | `context.gif`  | `ai context --strict` — offline context-window pre-flight   |
| `tags.tape`    | `tags.gif`     | `ai list-tags` — the commit-type tags `generate` accepts    |
| `keys.tape`    | `keys.gif`     | `ai key show` / `ai key swap` — the two Groq key slots      |
