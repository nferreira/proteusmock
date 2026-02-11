package testutil

import (
	"context"
	"time"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
)

var _ ports.Logger = (*NoopLogger)(nil)

// NoopLogger discards all log output.
type NoopLogger struct{}

func (l *NoopLogger) Info(string, ...any)  {}
func (l *NoopLogger) Warn(string, ...any)  {}
func (l *NoopLogger) Error(string, ...any) {}
func (l *NoopLogger) Debug(string, ...any) {}

var _ ports.Clock = (*FixedClock)(nil)

// FixedClock returns a fixed time and never sleeps.
type FixedClock struct {
	T time.Time
}

func (c *FixedClock) Now() time.Time { return c.T }
func (c *FixedClock) SleepContext(context.Context, time.Duration) error {
	return nil
}

var _ ports.RateLimiter = (*StubRateLimiter)(nil)

// StubRateLimiter returns a configurable Allow result.
type StubRateLimiter struct {
	AllowAll bool
}

func (r *StubRateLimiter) Allow(context.Context, string, float64, int) bool {
	return r.AllowAll
}

var _ match.BodyRenderer = (*StubBodyRenderer)(nil)

// StubBodyRenderer returns a configurable render result.
type StubBodyRenderer struct {
	Result []byte
	Err    error
}

func (r *StubBodyRenderer) Render(match.RenderContext) ([]byte, error) {
	return r.Result, r.Err
}
