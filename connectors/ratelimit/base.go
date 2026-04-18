package ratelimit

import "context"

// RateLimiterConnector abstracts distributed rate limiting.
type RateLimiterConnector interface {
	// Wait blocks until a request is allowed for the given key, or ctx is cancelled.
	// key identifies the rate limit bucket (e.g. "kalshi:read", "kalshi:write").
	// limit is the max requests per second for this bucket.
	Wait(ctx context.Context, key string, limit int) error
}

func DefaultRateLimiterConnector(ctx context.Context) (RateLimiterConnector, error) {
	return DefaultRedisRateLimiterConnector(ctx)
}
