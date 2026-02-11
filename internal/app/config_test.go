package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sophialabs/proteusmock/internal/app"
)

func writeTestScenario(t *testing.T, dir string) {
	t.Helper()
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
}

func TestDefaultConfig_HasSensibleValues(t *testing.T) {
	cfg := app.DefaultConfig()

	if cfg.RootDir == "" {
		t.Error("RootDir should not be empty")
	}
	if cfg.Port == 0 {
		t.Error("Port should not be zero")
	}
	if cfg.TraceSize == 0 {
		t.Error("TraceSize should not be zero")
	}
	if cfg.LogLevel == "" {
		t.Error("LogLevel should not be empty")
	}
	if cfg.RateLimiterTTL == 0 {
		t.Error("RateLimiterTTL should not be zero")
	}
	if cfg.WatcherDebounce == 0 {
		t.Error("WatcherDebounce should not be zero")
	}
	if cfg.ReadTimeout == 0 {
		t.Error("ReadTimeout should not be zero")
	}
	if cfg.WriteTimeout == 0 {
		t.Error("WriteTimeout should not be zero")
	}
	if cfg.IdleTimeout == 0 {
		t.Error("IdleTimeout should not be zero")
	}
	if cfg.ShutdownTimeout == 0 {
		t.Error("ShutdownTimeout should not be zero")
	}
}

func TestNew_Success(t *testing.T) {
	dir := t.TempDir()
	writeTestScenario(t, dir)

	cfg := app.DefaultConfig()
	cfg.RootDir = dir

	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil App")
	}
}

func TestNew_InvalidRootDir(t *testing.T) {
	cfg := app.DefaultConfig()
	cfg.RootDir = "/nonexistent/path/that/does/not/exist"

	_, err := app.New(cfg)
	if err == nil {
		t.Error("expected error for invalid root directory")
	}
}

func TestNew_WithDefaultEngine(t *testing.T) {
	dir := t.TempDir()
	writeTestScenario(t, dir)

	cfg := app.DefaultConfig()
	cfg.RootDir = dir
	cfg.DefaultEngine = "expr"

	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if a == nil {
		t.Fatal("expected non-nil App")
	}
}

func TestRun_WithAllLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			dir := t.TempDir()
			writeTestScenario(t, dir)

			cfg := app.DefaultConfig()
			cfg.RootDir = dir
			cfg.LogLevel = level

			a, err := app.New(cfg)
			if err != nil {
				t.Fatalf("New failed for log level %q: %v", level, err)
			}
			if a == nil {
				t.Fatalf("expected non-nil App for log level %q", level)
			}
		})
	}
}
