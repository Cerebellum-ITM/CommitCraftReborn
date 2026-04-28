package config

// FreeTierChatModels is the curated allowlist of Groq model IDs that are
// (a) available on the free tier and (b) chat-capable, i.e. usable as a
// commit-message generator. It deliberately excludes whisper/tts/guard/
// orpheus families and any model that requires the paid developer tier.
//
// The Groq /openai/v1/models response does not distinguish free vs paid,
// so we intersect it with this list at runtime. Update the slice when
// Groq promotes or retires a model on the free tier.
var FreeTierChatModels = []string{
	"llama-3.1-8b-instant",
	"llama-3.3-70b-versatile",
	"meta-llama/llama-4-scout-17b-16e-instruct",
	"openai/gpt-oss-20b",
	"openai/gpt-oss-120b",
	"qwen/qwen3-32b",
	"groq/compound",
	"groq/compound-mini",
	"allam-2-7b",
}

// IsFreeTierChatModel reports whether id is in the curated free-tier set.
func IsFreeTierChatModel(id string) bool {
	for _, m := range FreeTierChatModels {
		if m == id {
			return true
		}
	}
	return false
}
