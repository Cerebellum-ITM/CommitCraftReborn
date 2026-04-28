package api

import "sync"

// rateLimitCache stores the most recent RateLimits observed per model id,
// hydrated as the user makes calls. Lives only in memory — Groq's reset
// windows are minutes-long, so persisting across runs would be misleading.
var (
	rateLimitMu    sync.RWMutex
	rateLimitStore = map[string]RateLimits{}
)

// RecordRateLimits stores the latest RateLimits for modelID, overwriting
// any previous entry. Safe to call from multiple goroutines.
func RecordRateLimits(modelID string, rl RateLimits) {
	if modelID == "" {
		return
	}
	rateLimitMu.Lock()
	rateLimitStore[modelID] = rl
	rateLimitMu.Unlock()
}

// GetRateLimits returns the cached RateLimits for modelID. The bool is
// false when no call has been recorded yet for that model.
func GetRateLimits(modelID string) (RateLimits, bool) {
	rateLimitMu.RLock()
	rl, ok := rateLimitStore[modelID]
	rateLimitMu.RUnlock()
	return rl, ok
}
