package clock

import (
	"context"
	"time"

	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
)

var _ ports.Clock = (*RealClock)(nil)

// RealClock implements ports.Clock using the system clock.
type RealClock struct{}

// New creates a new RealClock.
func New() *RealClock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time { return time.Now() }

func (c *RealClock) SleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
