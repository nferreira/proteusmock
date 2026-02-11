//go:build integration

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
