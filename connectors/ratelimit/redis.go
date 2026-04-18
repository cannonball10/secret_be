package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RedisRateLimiter combines a local token-bucket (fast path) with a Redis
// sliding-window counter (distributed coordination). If Redis is unavailable
// it falls back to local-only limiting.
type RedisRateLimiter struct {
	client     *redis.Client
	clientOnce sync.Once

	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

// NewRedisRateLimiter returns a rate limiter backed by the given Redis client.
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:   client,
		limiters: make(map[string]*rate.Limiter),
	}
}

// DefaultRedisRateLimiterConnector creates a RedisRateLimiter from environment variables.
func DefaultRedisRateLimiterConnector(ctx context.Context) (*RedisRateLimiter, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil, errors.New("REDIS_ADDR is not set")
	}
	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")
	var db int
	if dbStr != "" {
		var err error
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
		}
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	return NewRedisRateLimiter(client), nil
}

func (r *RedisRateLimiter) Wait(ctx context.Context, key string, limit int) error {
	// Fast path: local token-bucket to reduce Redis round-trips.
	lim := r.getOrCreateLocal(key, limit)
	if int(lim.Limit()) != limit {
		lim.SetLimit(rate.Limit(limit))
		lim.SetBurst(limit)
	}
	if err := lim.Wait(ctx); err != nil {
		return err
	}

	// Distributed path: Redis sliding-window counter.
	if r.client == nil {
		return nil
	}
	return r.waitRedis(ctx, key, limit)
}

func (r *RedisRateLimiter) waitRedis(ctx context.Context, key string, limit int) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		now := time.Now().Unix()
		redisKey := fmt.Sprintf("rl:%s:%d", key, now)

		count, err := r.client.Incr(ctx, redisKey).Result()
		if err != nil {
			slog.Warn("redis rate limiter INCR failed, falling back to local-only", "key", key, "error", err)
			return nil // graceful degradation
		}

		// Set TTL on first increment so the key auto-expires.
		if count == 1 {
			r.client.Expire(ctx, redisKey, 2*time.Second)
		}

		if int(count) <= limit {
			return nil // allowed
		}

		// Over limit — undo the increment and wait until the next second boundary.
		r.client.Decr(ctx, redisKey)

		sleepDur := time.Until(time.Unix(now+1, 0))
		if sleepDur <= 0 {
			continue
		}
		select {
		case <-time.After(sleepDur):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *RedisRateLimiter) getOrCreateLocal(key string, limit int) *rate.Limiter {
	r.mu.RLock()
	lim, ok := r.limiters[key]
	r.mu.RUnlock()
	if ok {
		return lim
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if lim, ok = r.limiters[key]; ok {
		return lim
	}
	lim = rate.NewLimiter(rate.Limit(limit), limit)
	r.limiters[key] = lim
	return lim
}

// Ping checks Redis connectivity. Implements connectors.ConnectorHealthChecker.
func (r *RedisRateLimiter) Ping(ctx context.Context) error {
	if r.client == nil {
		return nil
	}
	return r.client.Ping(ctx).Err()
}

// Close closes the Redis client. Implements connectors.ConnectorCloser.
func (r *RedisRateLimiter) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
