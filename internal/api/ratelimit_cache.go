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

// EffectiveRateLimits returns rl adjusted for time elapsed since capture:
// if the per-resource reset window has already passed, the corresponding
// `Remaining*` value is bumped back up to its `Limit*` so the bar reads
// as a refilled bucket without needing a periodic tick. Resource buckets
// that haven't expired are returned unchanged.
func EffectiveRateLimits(rl RateLimits, now time.Time) RateLimits {
	if rl.CapturedAt.IsZero() {
		return rl
	}
	elapsed := now.Sub(rl.CapturedAt)
	out := rl
	if rl.ResetRequests > 0 && elapsed >= rl.ResetRequests && rl.LimitRequests > 0 {
		out.RemainingRequests = rl.LimitRequests
	}
	if rl.ResetTokens > 0 && elapsed >= rl.ResetTokens && rl.LimitTokens > 0 {
		out.RemainingTokens = rl.LimitTokens
	}
	return out
}
