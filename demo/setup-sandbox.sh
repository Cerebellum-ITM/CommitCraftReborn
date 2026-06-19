# Build a throwaway, fully-offline sandbox so the demo tapes can run the REAL
# commitcraft binary without touching the user's config, secrets, or repos.
#
# Sourced by demo/tapes/_setup.tape (NOT executed) so the exported HOME/PATH and
# the final `cd` persist into the VHS shell. Everything lives under /tmp and is
# rebuilt from scratch on every render — no network, no Groq key, no real data.

# Repo root = parent of this script's directory (demo/).
_cc_self="${BASH_SOURCE[0]:-$0}"
CC_REPO_ROOT="$(cd "$(dirname "$_cc_self")/.." && pwd)"

# Build the binary if it isn't there yet.
if [ ! -x "$CC_REPO_ROOT/bin/commitcraft" ]; then
  (cd "$CC_REPO_ROOT" && go build -o bin/commitcraft ./cmd/cli)
fi

CC_SB=/tmp/cc-demo
rm -rf "$CC_SB"
mkdir -p "$CC_SB/home/.config/CommitCraft"

# Sandbox HOME → config dir, the SQLite drafts DB and the .env all resolve under
# /tmp instead of the user's real ~/.config/CommitCraft.
export HOME="$CC_SB/home"
# Freshly built binary first on PATH so the on-screen `commitcraft` is the real one.
export PATH="$CC_REPO_ROOT/bin:$PATH"

# Offline Groq stand-in: point the app at a local mock so the TUI's ^W generate
# flow runs with invented data, no network, no API key, no quota. Relies on the
# COMMITCRAFT_GROQ_BASE_URL override. Kill any stale instance, then start fresh.
pkill -f "demo/mock-groq.py" >/dev/null 2>&1 || true
python3 "$CC_REPO_ROOT/demo/mock-groq.py" >/dev/null 2>&1 &
export COMMITCRAFT_GROQ_BASE_URL="http://127.0.0.1:8899/openai/v1"
# Give the mock a moment to bind its port.
for _ in 1 2 3 4 5 6 7 8 9 10; do
  curl -sf "http://127.0.0.1:8899/openai/v1/models" >/dev/null 2>&1 && break
  sleep 0.2
done

# Invented, non-functional keys: `ai key show` reports both slots set; nothing leaks.
cat > "$HOME/.config/CommitCraft/.env" <<'ENV'
GROQ_API_KEY="gsk_demo_user_0000000000000000000000000000"
GROQ_API_KEY_AI="gsk_demo_ai_00000000000000000000000000000"
GROQ_ACTIVE_KEY="user"
ENV

# Throwaway git repo with one invented, staged change to feed the pipeline.
mkdir -p "$CC_SB/repo/internal/api"
cd "$CC_SB/repo" || return
git init -q
git config user.email demo@example.com
git config user.name "CommitCraft Demo"
cat > internal/api/client.go <<'GO'
package api

// Client talks to the Groq HTTP API.
type Client struct{ key string }
GO
git add -A && git commit -qm "initial" >/dev/null 2>&1
cat >> internal/api/client.go <<'GO'

// doRequest retries transient 429/5xx responses with exponential backoff.
func (c *Client) doRequest() error { return nil }
GO
git add -A

# Seed the Groq models cache so `ai context` can compute usage % / fit offline.
# Context-window sizes are public model metadata, not secrets. Running any ai
# subcommand first creates the SQLite DB (runs migrations); then we insert a row.
commitcraft ai list >/dev/null 2>&1 || true
sqlite3 "$HOME/.config/CommitCraft/commits.db" \
  "INSERT OR REPLACE INTO groq_models_cache (id, owned_by, context_window, fetched_at) VALUES ('meta-llama/llama-4-scout-17b-16e-instruct','Meta',131072,$(date +%s));" 2>/dev/null || true

# A pre-written delegate payload the hero tape submits (stands in for what an AI
# agent would produce from the prompt bundle — see "Agent delegate mode"). Created
# AFTER `git add` so it stays untracked and out of the staged diff.
cat > msg.json <<'JSON'
{
  "kind": "commit",
  "action": "generate",
  "tag": "ADD",
  "scope": ["api"],
  "keypoints": ["retry transient Groq 429s with exponential backoff"],
  "title": "retry transient Groq errors with backoff",
  "body": "- Add doRequest, which retries transient 429 and 5xx responses\n  from the Groq API using an exponential backoff schedule.\n- Keeps the public client surface unchanged; callers see a single\n  request that transparently recovers from rate-limit blips."
}
JSON
