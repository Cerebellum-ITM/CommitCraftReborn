# Unit 25: dual-api-key-swap

## Goal

Support two Groq API keys — a **user** slot and an **ai** slot — with exactly
one active at a time, swappable via a new `commitcraft ai key` subcommand. Both
the TUI and every headless `commitcraft ai …` command resolve their key from
whichever slot is active. On a `429` rate-limit the error JSON gains a
`rate_limited` code plus a hint naming the active slot and the `ai key swap`
command — no automatic retry, no automatic fallback to the other slot.

Motivation: the free Groq tier rate-limits a single key, which blocked the
agent-driven `ai merge` that was meant to close `feat/agent-cli-improvements`.
A second key the user can swap to unblocks that flow manually.

**This is the closing unit of `feat/agent-cli-improvements`.** Once it lands,
the branch is ready to merge (the dual-key swap is precisely what unblocks the
rate-limited `ai merge` used to perform that merge).

## Design

Decisions locked with the user:

- **Manual swap only.** One active slot at a time; both code paths use the
  active slot's key. No per-context routing.
- **Report-only on 429.** Surface the rate limit with a clear code + hint;
  the user decides when to `ai key swap`. No auto-retry, no silent cross-slot
  fallback.
- **CLI prompts.** Managed through `commitcraft ai key` (no TUI screen yet).

### Storage (`.env`)

Everything lives in `~/.config/CommitCraft/.env`, reusing the existing
`saveEnvVar`-style upsert so unrelated keys (e.g. `GH_TOKEN`) stay intact:

| Var                | Meaning                                  |
| ------------------ | ---------------------------------------- |
| `GROQ_API_KEY`     | the **user** slot (existing var, unchanged name — backward compatible) |
| `GROQ_API_KEY_AI`  | the **ai** slot (new)                    |
| `GROQ_ACTIVE_KEY`  | `user` \| `ai` — the active pointer (new; absent/unknown ⇒ `user`) |

The active pointer is not a secret but lives in `.env` to keep all key state in
one file and avoid a `config.toml` read-modify-write round trip.

### Resolution rule (no silent fallback)

At load time the active slot's key becomes `TUI.GroqAPIKey`. If the active slot
is empty, `GroqAPIKey` stays empty and the existing "Groq API key was not
provided" error fires — we do **not** quietly borrow the other slot's key
(consistent with the project's no-silent-fallback ethos). `ai key show` makes
the state obvious so the user knows which slot to fill or swap to.

Existing single-key users are unaffected: no `GROQ_ACTIVE_KEY` ⇒ defaults to
`user` ⇒ reads `GROQ_API_KEY` exactly as today.

## Implementation

### A. Config plumbing — `internal/config/`

`types.go` — extend `TUIConfig` (all `toml:"-"`, derived from `.env`):

```go
GroqAPIKey    string `toml:"-"`
IsAPIKeySet   bool   `toml:"-"`
ActiveKeySlot string `toml:"-"` // "user" | "ai"
UserKeySet    bool   `toml:"-"`
AIKeySet      bool   `toml:"-"`
```

`loader.go` (~lines 247-256) — replace the single-key block with slot
resolution:

```go
userKey := os.Getenv("GROQ_API_KEY")
aiKey := os.Getenv("GROQ_API_KEY_AI")
active := strings.ToLower(strings.TrimSpace(os.Getenv("GROQ_ACTIVE_KEY")))
if active != "ai" {
    active = "user"
}
chosen := userKey
if active == "ai" {
    chosen = aiKey
}
globalCfg.TUI.ActiveKeySlot = active
globalCfg.TUI.UserKeySet = userKey != ""
globalCfg.TUI.AIKeySet = aiKey != ""
globalCfg.TUI.GroqAPIKey = chosen
globalCfg.TUI.IsAPIKeySet = chosen != ""
```

New exported helpers in `internal/config` so the headless `ai key` subcommand
can mutate `.env` without importing `internal/tui` (which would drag Bubble Tea
into the headless path). Reuse / lift the upsert logic that today lives in
`internal/tui/local_config.go:saveEnvVar`:

- `func SaveEnvVar(name, value string) error` — move the canonical upsert here
  (mode `0o600`, order-preserving, empty value deletes). Re-point
  `internal/tui/local_config.go:saveEnvVar` to delegate to
  `config.SaveEnvVar` so there is a single implementation.
- `func ActiveKeySlot() (string, error)` / convenience constants
  `KeySlotUser = "user"`, `KeySlotAI = "ai"`.

> `globalEnvPath()` resolution must match `loader.go`'s `filepath.Join(globalDir, ".env")`.

### B. `commitcraft ai key` subcommand — `internal/cli/ai/key.go`

Register in `ai.go` `Dispatch`: `case "key": return runKey(rest)`, and add a
usage line. Sub-actions (first positional after `key`):

| Invocation                              | Behavior |
| --------------------------------------- | -------- |
| `ai key` / `ai key show`                | Print JSON state (no secrets): `{active_slot, user_key_set, ai_key_set}`. |
| `ai key set [--slot user\|ai] [--value <k>]` | Set a slot's key. If `--slot` missing, prompt for it; if `--value` missing, prompt for the key on stdin. Writes the matching env var via `config.SaveEnvVar`. |
| `ai key swap`                           | Toggle `GROQ_ACTIVE_KEY` user↔ai, persist, print new state. Errors if the target slot has no key set (`empty_slot` code) so the user can't swap into a dead slot. |
| `ai key use --slot user\|ai`            | Set the active slot explicitly (same empty-slot guard as swap). |

Prompts: the slot choice (`user`/`ai`) is read with a plain `bufio.Reader` on
`os.Stdin`. The **key value is read with hidden input** via
`golang.org/x/term` — `term.ReadPassword(int(os.Stdin.Fd()))` so the secret is
never echoed to the screen. When stdin is not a terminal (piped/redirected),
detect it with `term.IsTerminal` and fall back to reading a full line so
scripted/agent use (`echo $KEY | commitcraft ai key set --slot ai`) still
works. Never print a stored key back; only report set/unset booleans.

Output stays JSON-on-stdout / errors-on-stderr like every other `ai`
subcommand, using the existing `printErrorJSON`. Exit codes: `0` success, `2`
usage error, `1` runtime/empty-slot error.

### C. 429 reporting — `internal/api/groq.go` + subcommand error handling

In `GetGroqChatCompletion`, when `resp.StatusCode == http.StatusTooManyRequests`
wrap a sentinel so callers can detect it through `errors.Is`:

```go
var ErrRateLimited = errors.New("groq rate limit (429)")
...
if resp.StatusCode == http.StatusTooManyRequests {
    return "", stats, fmt.Errorf("API returned 429: %s: %w", string(body), ErrRateLimited)
}
if resp.StatusCode != http.StatusOK { /* existing generic branch */ }
```

`aiengine` already wraps stage errors with `%w`, so `errors.Is` survives to the
subcommand. In the shared error path of the generate/merge/release/regenerate
subcommands, branch on it:

```go
if errors.Is(err, api.ErrRateLimited) {
    printErrorJSON("rate_limited",
        fmt.Sprintf("Groq rate-limited the active key (slot=%s). "+
            "Run `commitcraft ai key swap` to switch slots, then retry.",
            bs.cfg.TUI.ActiveKeySlot))
    return 1
}
printErrorJSON("api_error", err.Error())
```

(Factor this into a small `printAIRunError(bs, err)` helper in `ai.go` to avoid
copy-paste across the four subcommands that call `aiengine.Run`/`RunRelease`.)

## Dependencies

- `golang.org/x/term` — hidden (no-echo) reading of the key value at the
  prompt, with `term.IsTerminal` to fall back to a plain line read when stdin
  is piped. `go get golang.org/x/term` (transitively present in the Go
  toolchain ecosystem; pulled into `go.mod`/`go.sum`).
- Otherwise reuses `godotenv` (already loaded in `loader.go`), the existing
  `.env` upsert logic, and stdlib `bufio`/`errors`/`flag`.

## Verify when done

- [ ] `go build ./...` + `go vet ./...` clean.
- [ ] Existing single-key setup (`GROQ_API_KEY` only, no `GROQ_ACTIVE_KEY`)
      still resolves the key and runs `ai generate` unchanged.
- [ ] `commitcraft ai key set --slot ai --value <key2>` writes
      `GROQ_API_KEY_AI` to `.env` and leaves `GROQ_API_KEY` / `GH_TOKEN` intact.
- [ ] `commitcraft ai key show` reports `user_key_set` / `ai_key_set` correctly
      and never prints a key value.
- [ ] `commitcraft ai key swap` toggles `GROQ_ACTIVE_KEY` and the next
      `ai generate` uses the other slot's key; swapping into an empty slot
      errors with `empty_slot`.
- [ ] The key value typed at the `ai key set` prompt is NOT echoed to the
      screen (hidden via `x/term`); piping `echo $KEY | …` still works.
- [ ] A forced 429 surfaces `code: "rate_limited"` with the active-slot hint
      (not the generic `api_error`).
- [ ] `internal/tui/local_config.go:saveEnvVar` now delegates to
      `config.SaveEnvVar` (single implementation; TUI API-key save still works).
- [ ] Bump `cmd/cli/main.go` version to `v0.66.0` and add a `CHANGELOG.md`
      entry with a `### Usage` block for `ai key`.
- [ ] Update `context/progress-tracker.md` to mark unit 25 complete.
