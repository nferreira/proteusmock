package trace_test

import (
	"sync"
	"testing"
	"time"

	"github.com/sophialabs/proteusmock/internal/domain/trace"
)

func TestRingBuffer_AddAndLast(t *testing.T) {
	rb := trace.NewRingBuffer(3)

	if rb.Count() != 0 {
		t.Fatalf("expected count 0, got %d", rb.Count())
	}

	rb.Add(trace.Entry{Method: "GET", Path: "/a"})
	rb.Add(trace.Entry{Method: "POST", Path: "/b"})

	entries := rb.Last(5)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != "/a" {
		t.Errorf("expected /a, got %s", entries[0].Path)
	}
	if entries[1].Path != "/b" {
		t.Errorf("expected /b, got %s", entries[1].Path)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := trace.NewRingBuffer(3)

	rb.Add(trace.Entry{Path: "/a"})
	rb.Add(trace.Entry{Path: "/b"})
	rb.Add(trace.Entry{Path: "/c"})
	rb.Add(trace.Entry{Path: "/d"})

	if rb.Count() != 3 {
		t.Fatalf("expected count 3, got %d", rb.Count())
	}

	entries := rb.Last(3)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Path != "/b" {
		t.Errorf("expected /b, got %s", entries[0].Path)
	}
	if entries[1].Path != "/c" {
		t.Errorf("expected /c, got %s", entries[1].Path)
	}
	if entries[2].Path != "/d" {
		t.Errorf("expected /d, got %s", entries[2].Path)
	}
}

func TestRingBuffer_LastZero(t *testing.T) {
	rb := trace.NewRingBuffer(5)
	rb.Add(trace.Entry{Path: "/a"})

	entries := rb.Last(0)
	if entries != nil {
		t.Errorf("expected nil, got %v", entries)
	}
}

func TestRingBuffer_Concurrency(t *testing.T) {
	rb := trace.NewRingBuffer(100)
	var wg sync.WaitGroup
	n := 50

	// Concurrent writers.
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			rb.Add(trace.Entry{
				Timestamp: time.Now(),
				Path:      "/concurrent",
				MatchedID: "",
			})
		}(i)
	}

	// Concurrent readers.
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rb.Last(10)
			_ = rb.Count()
		}()
	}

	wg.Wait()

	if rb.Count() != n {
		t.Errorf("expected count %d, got %d", n, rb.Count())
	}
}
