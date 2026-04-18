package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrCacheMiss = errors.New("cache miss")

type memoryEntry struct {
	value     interface{}
	expiresAt *time.Time
}

// MemoryCache is a simple in-memory cache with optional TTL support.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]memoryEntry
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: map[string]memoryEntry{},
	}
}

func (c *MemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, ErrCacheMiss
	}

	if entry.expiresAt != nil && time.Now().After(*entry.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, ErrCacheMiss
	}

	return entry.value, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl *time.Duration) error {
	var expiresAt *time.Time
	if ttl != nil {
		t := time.Now().Add(*ttl)
		expiresAt = &t
	}

	c.mu.Lock()
	c.items[key] = memoryEntry{
		value:     value,
		expiresAt: expiresAt,
	}
	c.mu.Unlock()

	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
	return nil
}
