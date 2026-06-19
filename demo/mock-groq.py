#!/usr/bin/env python3
"""Tiny offline stand-in for the Groq API, used only to record the demo GIFs.

CommitCraft's TUI talks to whatever `COMMITCRAFT_GROQ_BASE_URL` points at. The
demo sandbox points it here, so the `^W` generate flow runs end to end with
**invented** content and **no network, no API key, no quota** — the GIFs stay
reproducible and faithful to the real UI.

It answers the two endpoints the app uses:
  GET  /openai/v1/models             → a one-model catalogue
  POST /openai/v1/chat/completions   → a canned completion, branched by the
                                       pipeline stage detected in the prompt.
"""

import json
import time
from http.server import BaseHTTPRequestHandler, HTTPServer

PORT = 8899
MODEL = "meta-llama/llama-4-scout-17b-16e-instruct"

# Canned, fully-invented outputs for the sandbox change (an ADD on `api`:
# doRequest retries transient Groq 429/5xx responses with exponential backoff).
SUMMARY = (
    "- Adds retry handling to the Groq HTTP client so transient failures no\n"
    "  longer surface to the caller.\n"
    "  * New doRequest method wraps the request with an exponential backoff\n"
    "    loop covering 429 and 5xx responses.\n"
    "  * The public client surface is unchanged; callers see one request."
)
BODY = (
    "- Add doRequest, which retries transient 429 and 5xx responses from the\n"
    "  Groq API using an exponential backoff schedule.\n"
    "- Keeps the public client surface unchanged; callers see a single request\n"
    "  that transparently recovers from rate-limit blips."
)
TITLE = "retry transient Groq errors with backoff"


def pick_content(payload: str) -> str:
    text = payload.lower()
    if "expert code change analyzer" in text:
        return SUMMARY
    if "writing commit message bodies" in text:
        return BODY
    if "expert commit message editor" in text:
        return TITLE
    if "changelog" in text:
        return SUMMARY
    return BODY


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):  # silence access logs
        pass

    def _json(self, code, obj):
        data = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("x-request-id", "demo-mock-0001")
        self.send_header("x-ratelimit-limit-requests", "1000")
        self.send_header("x-ratelimit-remaining-requests", "999")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path.endswith("/models"):
            self._json(200, {
                "object": "list",
                "data": [{
                    "id": MODEL, "object": "model",
                    "owned_by": "Meta", "context_window": 131072,
                }],
            })
        else:
            self._json(404, {"error": "not found"})

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        payload = self.rfile.read(length).decode("utf-8", "replace")
        # A touch of latency so the multi-stage pipeline animation is visible.
        time.sleep(0.6)
        content = pick_content(payload)
        self._json(200, {
            "model": MODEL,
            "choices": [{"index": 0, "message": {"role": "assistant", "content": content}}],
            "usage": {
                "prompt_tokens": 420, "completion_tokens": 88, "total_tokens": 508,
                "queue_time": 0.012, "prompt_time": 0.031,
                "completion_time": 0.144, "total_time": 0.175,
            },
        })


if __name__ == "__main__":
    HTTPServer(("127.0.0.1", PORT), Handler).serve_forever()
