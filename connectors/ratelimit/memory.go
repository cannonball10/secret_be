package ratelimit

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// MemoryRateLimiter is an in-memory rate limiter for single-process use or tests.
type MemoryRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

// NewMemoryRateLimiter returns a new in-memory rate limiter.
func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{limiters: make(map[string]*rate.Limiter)}
}

func (m *MemoryRateLimiter) Wait(ctx context.Context, key string, limit int) error {
	lim := m.getOrCreate(key, limit)

	// Update limit if it changed.
	if int(lim.Limit()) != limit {
		lim.SetLimit(rate.Limit(limit))
		lim.SetBurst(limit)
	}

	return lim.Wait(ctx)
}

func (m *MemoryRateLimiter) getOrCreate(key string, limit int) *rate.Limiter {
	m.mu.RLock()
	lim, ok := m.limiters[key]
	m.mu.RUnlock()
	if ok {
		return lim
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock.
	if lim, ok = m.limiters[key]; ok {
		return lim
	}
	lim = rate.NewLimiter(rate.Limit(limit), limit)
	m.limiters[key] = lim
	return lim
}
