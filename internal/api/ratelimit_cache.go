package api

import (
	"sync"
	"time"
)

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

// EffectiveRateLimits is reserved for any future post-processing of a
// captured RateLimits at render time. The previous implementation tried
// to "decay" Remaining* back up to Limit* when `elapsed >= reset`, but
// Groq's `reset-*` headers describe how long until the *next slot*
// becomes available (token-bucket refill rate), not when the entire
// bucket refills — so that decay produced false zeros. The captured
// snapshot is now returned unchanged; bars only refresh on the next
// real call, which matches what the user actually sees against Groq.
func EffectiveRateLimits(rl RateLimits, _ time.Time) RateLimits {
	return rl
}
