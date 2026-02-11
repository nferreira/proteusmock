package app_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/app"
)

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

func TestRun_StartsAndShutdownsGracefully(t *testing.T) {
	dir := t.TempDir()
	writeTestScenario(t, dir)

	port := freePort(t)
	cfg := app.DefaultConfig()
	cfg.RootDir = dir
	cfg.Port = port

	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Run(ctx)
	}()

	// Wait for server to be ready.
	addr := fmt.Sprintf("http://localhost:%d/api/health", port)
	waitForServer(t, addr, 3*time.Second)

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestRun_FailsOnInvalidScenarios(t *testing.T) {
	dir := t.TempDir()
	scenarioDir := filepath.Join(dir, "scenarios")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("failed to create scenario dir: %v", err)
	}
	// Write a YAML file with duplicate IDs.
	yaml := `- id: dup
  name: Dup One
  when:
    method: GET
    path: /a
  response:
    status: 200
- id: dup
  name: Dup Two
  when:
    method: GET
    path: /b
  response:
    status: 200
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "dups.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("failed to write scenario file: %v", err)
	}

	cfg := app.DefaultConfig()
	cfg.RootDir = dir

	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = a.Run(ctx)
	if err == nil {
		t.Error("expected error for invalid scenarios")
	}
}

func TestRun_ListensOnPort(t *testing.T) {
	dir := t.TempDir()
	writeTestScenario(t, dir)

	port := freePort(t)
	cfg := app.DefaultConfig()
	cfg.RootDir = dir
	cfg.Port = port

	a, err := app.New(cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Run(ctx)
	}()

	addr := fmt.Sprintf("http://localhost:%d/api/health", port)
	waitForServer(t, addr, 3*time.Second)

	resp, err := http.Get(addr)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func waitForServer(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server not ready at %s after %v", url, timeout)
}
