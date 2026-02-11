package trace

import "sync"

// RingBuffer is a concurrent-safe fixed-size ring buffer for trace entries.
type RingBuffer struct {
	mu      sync.RWMutex
	entries []Entry
	size    int
	head    int
	count   int
}

// NewRingBuffer creates a ring buffer that holds up to size entries.
func NewRingBuffer(size int) *RingBuffer {
	if size <= 0 {
		size = 100
	}
	return &RingBuffer{
		entries: make([]Entry, size),
		size:    size,
	}
}

// Add appends an entry to the ring buffer, overwriting the oldest if full.
func (rb *RingBuffer) Add(e Entry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.entries[rb.head] = e
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// Last returns the last n entries in chronological order.
func (rb *RingBuffer) Last(n int) []Entry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	if n > rb.count {
		n = rb.count
	}
	if n <= 0 {
		return nil
	}

	result := make([]Entry, n)
	start := (rb.head - n + rb.size) % rb.size
	for i := range n {
		result[i] = rb.entries[(start+i)%rb.size]
	}
	return result
}

// Count returns the number of entries currently stored.
func (rb *RingBuffer) Count() int {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.count
}
