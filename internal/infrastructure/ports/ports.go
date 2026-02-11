package ports

import (
	"context"
	"time"
)

// Clock provides the current time (for testing).
type Clock interface {
	Now() time.Time
	// SleepContext blocks for d or until ctx is cancelled. Returns ctx.Err() if cancelled.
	SleepContext(ctx context.Context, d time.Duration) error
}

// Logger provides structured logging.
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

// RateLimiter checks whether a request is allowed under rate limits.
type RateLimiter interface {
	// Allow checks if a request identified by key is within the rate limit.
	// rate is tokens per second, burst is the max burst size.
	Allow(ctx context.Context, key string, rate float64, burst int) bool
}
