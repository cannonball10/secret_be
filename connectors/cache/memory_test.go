package cache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	if err := c.Set(ctx, "k1", "v1", nil); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "v1" {
		t.Errorf("got %v, want %q", got, "v1")
	}
}

func TestMemoryCache_GetMiss(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	_, err := c.Get(ctx, "missing")
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	_ = c.Set(ctx, "k1", "v1", nil)
	_ = c.Delete(ctx, "k1")

	_, err := c.Get(ctx, "k1")
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestMemoryCache_Overwrite(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	_ = c.Set(ctx, "k1", "v1", nil)
	_ = c.Set(ctx, "k1", "v2", nil)

	got, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "v2" {
		t.Errorf("got %v, want %q", got, "v2")
	}
}

func TestMemoryCache_TTLExpiry(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()
	ttl := time.Millisecond

	_ = c.Set(ctx, "k1", "v1", &ttl)

	// Should be available immediately
	_, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("expected value before expiry, got %v", err)
	}

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	_, err = c.Get(ctx, "k1")
	if !errors.Is(err, ErrCacheMiss) {
		t.Errorf("expected ErrCacheMiss after TTL, got %v", err)
	}
}

func TestMemoryCache_NoTTL(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	_ = c.Set(ctx, "k1", "v1", nil)

	// Should still be available (no expiry)
	got, err := c.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "v1" {
		t.Errorf("got %v, want %q", got, "v1")
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := "k"
			_ = c.Set(ctx, key, n, nil)
			_, _ = c.Get(ctx, key)
			_ = c.Delete(ctx, key)
		}(i)
	}

	wg.Wait()
}
