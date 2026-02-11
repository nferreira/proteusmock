package wiring_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/infrastructure/wiring"
	"github.com/sophialabs/proteusmock/internal/testutil"
)

func validParams(t *testing.T) wiring.Params {
	t.Helper()
	dir := t.TempDir()
	scenarioDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}
	yaml := `id: test-health
name: Test Health
priority: 10
when:
  method: GET
  path: /api/health
response:
  status: 200
  body: '{"status":"ok"}'
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "health.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("failed to write scenario file: %v", err)
	}

	return wiring.Params{
		RootDir:        dir,
		TraceSize:      50,
		RateLimiterTTL: 5 * time.Minute,
		Logger:         &testutil.NoopLogger{},
	}
}

func TestNew_Success(t *testing.T) {
	p := validParams(t)
	c, err := wiring.New(p)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer c.Close()

	if c.Logger() == nil {
		t.Error("Logger() returned nil")
	}
	if c.Server() == nil {
		t.Error("Server() returned nil")
	}
	if c.LoadScenariosUseCase() == nil {
		t.Error("LoadScenariosUseCase() returned nil")
	}
	if c.RateLimiterStore() == nil {
		t.Error("RateLimiterStore() returned nil")
	}
	if c.TraceBuf() == nil {
		t.Error("TraceBuf() returned nil")
	}
}

func TestNew_InvalidRootDir(t *testing.T) {
	p := wiring.Params{
		RootDir:        "/nonexistent/path/that/does/not/exist",
		TraceSize:      50,
		RateLimiterTTL: 5 * time.Minute,
		Logger:         &testutil.NoopLogger{},
	}

	c, err := wiring.New(p)
	if err == nil {
		c.Close()
		t.Fatal("expected error for invalid root dir")
	}
	if c != nil {
		t.Error("expected nil container on error")
	}
}

func TestNew_ComponentsAreWiredCorrectly(t *testing.T) {
	p := validParams(t)
	c, err := wiring.New(p)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer c.Close()

	idx, err := c.LoadScenariosUseCase().Execute(context.Background())
	if err != nil {
		t.Fatalf("LoadScenariosUseCase().Execute() failed: %v", err)
	}
	if idx == nil {
		t.Error("expected non-nil index")
	}
}

func TestNew_LoggerIsPassedThrough(t *testing.T) {
	p := validParams(t)
	logger := &testutil.NoopLogger{}
	p.Logger = logger

	c, err := wiring.New(p)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer c.Close()

	if c.Logger() != logger {
		t.Error("Logger() does not return the same logger instance passed in Params")
	}
}

func TestClose_IsIdempotent(t *testing.T) {
	p := validParams(t)
	c, err := wiring.New(p)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Double close must not panic.
	c.Close()
	c.Close()
}
