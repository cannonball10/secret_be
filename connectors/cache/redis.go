package cache

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client        *redis.Client
	clientOnce    sync.Once
	clientFactory func() *redis.Client
}

// NewRedisCache returns a cache wrapper with an eagerly provided client.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

// NewRedisCacheLazy returns a cache wrapper that initializes the client on first use.
func NewRedisCacheLazy(factory func() *redis.Client) *RedisCache {
	return &RedisCache{clientFactory: factory}
}

func DefaultRedisCacheConnector(ctx context.Context) (*RedisCache, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil, errors.New("redis address is not set")
	}
	password := os.Getenv("REDIS_PASSWORD")
	dbstring := os.Getenv("REDIS_DB")
	var db int
	if dbstring == "" {
		db = 0
	} else {
		var err error
		db, err = strconv.Atoi(dbstring)
		if err != nil {
			return nil, err
		}
	}
	return NewRedisCache(redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})), nil
}

func (c *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}
	val, err := client.Get(ctx, key).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		slog.ErrorContext(ctx, "redis Get failed", "key", key, "error", err)
	}
	return val, err
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl *time.Duration) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}
	var setErr error
	if ttl != nil {
		setErr = client.Set(ctx, key, value, *ttl).Err()
	} else {
		setErr = client.Set(ctx, key, value, 0).Err()
	}
	if setErr != nil {
		slog.ErrorContext(ctx, "redis Set failed", "key", key, "error", setErr)
	}
	return setErr
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}
	if err := client.Del(ctx, key).Err(); err != nil {
		slog.ErrorContext(ctx, "redis Delete failed", "key", key, "error", err)
		return err
	}
	return nil
}

func (c *RedisCache) getClient() (*redis.Client, error) {
	c.clientOnce.Do(func() {
		if c.client == nil && c.clientFactory != nil {
			c.client = c.clientFactory()
		}
	})
	if c.client == nil {
		return nil, errors.New("redis client is not initialized")
	}
	return c.client, nil
}

// Ping checks Redis connectivity. Implements connectors.ConnectorHealthChecker.
func (c *RedisCache) Ping(ctx context.Context) error {
	client, err := c.getClient()
	if err != nil {
		return err
	}
	if err := client.Ping(ctx).Err(); err != nil {
		slog.ErrorContext(ctx, "redis Ping failed", "error", err)
		return err
	}
	return nil
}

// Close closes the Redis client connection.
func (c *RedisCache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
