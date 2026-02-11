package filesystem_test

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/infrastructure/outbound/filesystem"
)

type testLogger struct{}

func (l *testLogger) Info(string, ...any)  {}
func (l *testLogger) Warn(string, ...any)  {}
func (l *testLogger) Error(string, ...any) {}
func (l *testLogger) Debug(string, ...any) {}

func TestWatcher_DetectsFileCreate(t *testing.T) {
	tmpDir := t.TempDir()

	var reloadCount atomic.Int32
	w, err := filesystem.NewWatcher(tmpDir, 100*time.Millisecond, &testLogger{}, func() {
		reloadCount.Add(1)
	})
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Stop()
	w.Start()

	// Create a YAML file.
	if err := os.WriteFile(filepath.Join(tmpDir, "test.yaml"), []byte("id: test"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Wait for debounce + processing.
	time.Sleep(500 * time.Millisecond)

	if reloadCount.Load() < 1 {
		t.Error("expected at least one reload")
	}
}

func TestWatcher_DetectsFileModify(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create a file.
	f := filepath.Join(tmpDir, "existing.yaml")
	os.WriteFile(f, []byte("id: v1"), 0644)

	var reloadCount atomic.Int32
	w, err := filesystem.NewWatcher(tmpDir, 100*time.Millisecond, &testLogger{}, func() {
		reloadCount.Add(1)
	})
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Stop()
	w.Start()

	// Modify the file.
	os.WriteFile(f, []byte("id: v2"), 0644)

	time.Sleep(500 * time.Millisecond)

	if reloadCount.Load() < 1 {
		t.Error("expected at least one reload on modify")
	}
}

func TestWatcher_IgnoresNonYAML(t *testing.T) {
	tmpDir := t.TempDir()

	var reloadCount atomic.Int32
	w, err := filesystem.NewWatcher(tmpDir, 100*time.Millisecond, &testLogger{}, func() {
		reloadCount.Add(1)
	})
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Stop()
	w.Start()

	// Create a non-YAML file.
	os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hello"), 0644)

	time.Sleep(500 * time.Millisecond)

	if reloadCount.Load() != 0 {
		t.Error("expected no reload for non-YAML file")
	}
}

func TestWatcher_Debounce(t *testing.T) {
	tmpDir := t.TempDir()

	var reloadCount atomic.Int32
	w, err := filesystem.NewWatcher(tmpDir, 200*time.Millisecond, &testLogger{}, func() {
		reloadCount.Add(1)
	})
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Stop()
	w.Start()

	// Rapid-fire changes — should debounce into one reload.
	for i := range 5 {
		os.WriteFile(filepath.Join(tmpDir, "test.yaml"), []byte("id: "+string(rune('a'+i))), 0644)
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(500 * time.Millisecond)

	count := reloadCount.Load()
	if count > 2 {
		t.Errorf("expected 1-2 reloads (debounced), got %d", count)
	}
}
