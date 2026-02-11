package clock_test

import (
	"context"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/clock"
)

func TestRealClock_Now(t *testing.T) {
	clk := clock.New()
	before := time.Now()
	got := clk.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, want between %v and %v", got, before, after)
	}
}

func TestRealClock_SleepContext_Normal(t *testing.T) {
	clk := clock.New()
	ctx := context.Background()

	start := time.Now()
	err := clk.SleepContext(ctx, 50*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("SleepContext returned unexpected error: %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("SleepContext returned too early: %v", elapsed)
	}
}

func TestRealClock_SleepContext_Cancelled(t *testing.T) {
	clk := clock.New()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately.
	cancel()

	err := clk.SleepContext(ctx, 10*time.Second)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
