package cache

import (
	"context"
	"time"
)

// CacheConnector is an abstract cache interface that can be extended for concrete implementations like Redis, Memcached, etc.
type CacheConnector interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl *time.Duration) error
	Delete(ctx context.Context, key string) error
}

func DefaultCacheConnector(ctx context.Context) (CacheConnector, error) {
	return DefaultRedisCacheConnector(ctx)
}
