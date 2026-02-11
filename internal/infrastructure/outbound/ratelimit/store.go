package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
)

var _ ports.RateLimiter = (*TokenBucketStore)(nil)

type limiterEntry struct {
	limiter  *rate.Limiter
	rate     float64
	burst    int
	lastUsed time.Time
}

// TokenBucketStore provides per-key rate limiters using token bucket algorithm.
type TokenBucketStore struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	ttl      time.Duration
	stop     chan struct{}
}

// NewTokenBucketStore creates a new store with the given TTL for inactive limiters.
// It starts a background goroutine that evicts stale entries every TTL interval.
// Call Stop to terminate the eviction goroutine.
func NewTokenBucketStore(ttl time.Duration) *TokenBucketStore {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	s := &TokenBucketStore{
		limiters: make(map[string]*limiterEntry),
		ttl:      ttl,
		stop:     make(chan struct{}),
	}
	go s.evictLoop()
	return s
}

// Stop terminates the background eviction goroutine.
func (s *TokenBucketStore) Stop() {
	close(s.stop)
}

func (s *TokenBucketStore) evictLoop() {
	ticker := time.NewTicker(s.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.Evict()
		case <-s.stop:
			return
		}
	}
}

// Allow checks if a request for the given key is within limits.
func (s *TokenBucketStore) Allow(_ context.Context, key string, r float64, burst int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.limiters[key]
	if !ok {
		entry = &limiterEntry{
			limiter: rate.NewLimiter(rate.Limit(r), burst),
			rate:    r,
			burst:   burst,
		}
		s.limiters[key] = entry
	} else if entry.rate != r || entry.burst != burst {
		// Rate/burst params changed (e.g. after hot-reload), update the limiter.
		entry.limiter.SetLimit(rate.Limit(r))
		entry.limiter.SetBurst(burst)
		entry.rate = r
		entry.burst = burst
	}

	entry.lastUsed = time.Now()
	return entry.limiter.Allow()
}

// Evict removes inactive entries older than the TTL.
func (s *TokenBucketStore) Evict() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.ttl)
	for key, entry := range s.limiters {
		if entry.lastUsed.Before(cutoff) {
			delete(s.limiters, key)
		}
	}
}

// Len returns the number of active limiters.
func (s *TokenBucketStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.limiters)
}
