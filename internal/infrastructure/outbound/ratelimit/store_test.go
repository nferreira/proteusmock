package ratelimit_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/ratelimit"
)

func TestTokenBucketStore_AllowWithinBurst(t *testing.T) {
	store := ratelimit.NewTokenBucketStore(time.Minute)
	defer store.Stop()
	ctx := context.Background()

	// Burst of 3, allow 3 requests immediately.
	for i := range 3 {
		if !store.Allow(ctx, "key1", 1, 3) {
			t.Errorf("request %d should be allowed within burst", i+1)
		}
	}
}

func TestTokenBucketStore_DeniedOverBurst(t *testing.T) {
	store := ratelimit.NewTokenBucketStore(time.Minute)
	defer store.Stop()
	ctx := context.Background()

	// Exhaust the burst.
	for range 5 {
		store.Allow(ctx, "key1", 1, 5)
	}

	// Next request should be denied.
	if store.Allow(ctx, "key1", 1, 5) {
		t.Error("request over burst should be denied")
	}
}

func TestTokenBucketStore_PerKeyIsolation(t *testing.T) {
	store := ratelimit.NewTokenBucketStore(time.Minute)
	defer store.Stop()
	ctx := context.Background()

	// Exhaust key1.
	for range 2 {
		store.Allow(ctx, "key1", 1, 2)
	}

	// key2 should still be allowed.
	if !store.Allow(ctx, "key2", 1, 2) {
		t.Error("key2 should be allowed (separate from key1)")
	}
}

func TestTokenBucketStore_Len(t *testing.T) {
	store := ratelimit.NewTokenBucketStore(time.Minute)
	defer store.Stop()
	ctx := context.Background()

	store.Allow(ctx, "a", 1, 1)
	store.Allow(ctx, "b", 1, 1)
	store.Allow(ctx, "a", 1, 1) // Reuse existing key.

	if store.Len() != 2 {
		t.Errorf("expected 2 limiters, got %d", store.Len())
	}
}

func TestTokenBucketStore_Evict(t *testing.T) {
	store := ratelimit.NewTokenBucketStore(1 * time.Millisecond)
	defer store.Stop()
	ctx := context.Background()

	store.Allow(ctx, "old", 1, 1)
	time.Sleep(10 * time.Millisecond)
	store.Evict()

	if store.Len() != 0 {
		t.Errorf("expected 0 after eviction, got %d", store.Len())
	}
}

func TestTokenBucketStore_UpdatedParamsOnHotReload(t *testing.T) {
	store := ratelimit.NewTokenBucketStore(time.Minute)
	defer store.Stop()
	ctx := context.Background()

	// Create a limiter with rate=1, burst=2.
	store.Allow(ctx, "reload-key", 1, 2)
	if store.Len() != 1 {
		t.Fatalf("expected 1 limiter, got %d", store.Len())
	}

	// Call with different params — should reuse the same key (not create new entry).
	store.Allow(ctx, "reload-key", 10, 20)
	if store.Len() != 1 {
		t.Errorf("expected 1 limiter after param update, got %d", store.Len())
	}

	// Verify updated rate takes effect: wait 200ms at rate=10/s → 2 tokens.
	// Exhaust existing tokens first.
	for store.Allow(ctx, "reload-key", 10, 20) {
		// drain
	}
	time.Sleep(200 * time.Millisecond)

	// With old rate (1/s), 200ms → ~0 tokens. With new rate (10/s), 200ms → ~2 tokens.
	if !store.Allow(ctx, "reload-key", 10, 20) {
		t.Error("expected token available after rate increase and sleep")
	}
}

func TestTokenBucketStore_Concurrent(t *testing.T) {
	store := ratelimit.NewTokenBucketStore(time.Minute)
	defer store.Stop()
	ctx := context.Background()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store.Allow(ctx, "concurrent", 100, 100)
		}(i)
	}

	wg.Wait()

	if store.Len() != 1 {
		t.Errorf("expected 1 limiter, got %d", store.Len())
	}
}
