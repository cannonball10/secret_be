//go:build integration

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/cannonball10/foundation/testutil"
	"github.com/redis/go-redis/v9"
)

func setupRedisTest(t *testing.T) (*RedisCache, string) {
	t.Helper()
	testutil.SkipIfServiceUnavailable(t, testutil.RedisAddr, "Redis")

	client := redis.NewClient(&redis.Options{
		Addr: testutil.RedisAddr,
	})

	cache := NewRedisCache(client)
	prefix := testutil.TestPrefix(t)

	t.Cleanup(func() {
		// Clean up test keys
		ctx := context.Background()
		keys, _ := client.Keys(ctx, prefix+"*").Result()
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		cache.Close()
	})

	return cache, prefix
}

func TestRedisCache_SetGet(t *testing.T) {
	cache, prefix := setupRedisTest(t)
	ctx := context.Background()

	key := prefix + "testkey"
	value := "testvalue"

	// Set value
	err := cache.Set(ctx, key, value, nil)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get value
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != value {
		t.Errorf("Get returned %q, want %q", got, value)
	}
}

func TestRedisCache_SetGetWithTTL(t *testing.T) {
	cache, prefix := setupRedisTest(t)
	ctx := context.Background()

	key := prefix + "ttlkey"
	value := "ttlvalue"
	ttl := 100 * time.Millisecond

	// Set value with TTL
	err := cache.Set(ctx, key, value, &ttl)
	if err != nil {
		t.Fatalf("Set with TTL failed: %v", err)
	}

	// Verify value exists
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed immediately after Set: %v", err)
	}
	if got != value {
		t.Errorf("Get returned %q, want %q", got, value)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Verify value expired
	_, err = cache.Get(ctx, key)
	if err == nil {
		t.Error("Expected error for expired key, got nil")
	}
}

func TestRedisCache_Delete(t *testing.T) {
	cache, prefix := setupRedisTest(t)
	ctx := context.Background()

	key := prefix + "deletekey"
	value := "deletevalue"

	// Set value
	err := cache.Set(ctx, key, value, nil)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify value exists
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != value {
		t.Errorf("Get returned %q, want %q", got, value)
	}

	// Delete value
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify value is gone
	_, err = cache.Get(ctx, key)
	if err == nil {
		t.Error("Expected error for deleted key, got nil")
	}
}

func TestRedisCache_GetNonExistent(t *testing.T) {
	cache, prefix := setupRedisTest(t)
	ctx := context.Background()

	key := prefix + "nonexistent"

	_, err := cache.Get(ctx, key)
	if err == nil {
		t.Error("Expected error for non-existent key, got nil")
	}
}

func TestRedisCache_OverwriteExisting(t *testing.T) {
	cache, prefix := setupRedisTest(t)
	ctx := context.Background()

	key := prefix + "overwritekey"
	value1 := "value1"
	value2 := "value2"

	// Set initial value
	err := cache.Set(ctx, key, value1, nil)
	if err != nil {
		t.Fatalf("Set initial value failed: %v", err)
	}

	// Verify initial value
	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get initial value failed: %v", err)
	}
	if got != value1 {
		t.Errorf("Get returned %q, want %q", got, value1)
	}

	// Overwrite with new value
	err = cache.Set(ctx, key, value2, nil)
	if err != nil {
		t.Fatalf("Set overwrite value failed: %v", err)
	}

	// Verify new value
	got, err = cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get overwrite value failed: %v", err)
	}
	if got != value2 {
		t.Errorf("Get returned %q, want %q", got, value2)
	}
}
