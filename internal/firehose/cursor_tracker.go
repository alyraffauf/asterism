package firehose

import "sync"

// cursorTracker tracks completion of firehose event sequence numbers that
// may finish out of order (different repos are verified/indexed
// concurrently) and reports the highest contiguous prefix that has fully
// completed, so the persisted cursor never advances past an event that
// hasn't actually finished.
type cursorTracker struct {
	mu          sync.Mutex
	pending     map[int64]struct{}
	persisted   int64
	initialized bool
}

func newCursorTracker() *cursorTracker {
	return &cursorTracker{pending: make(map[int64]struct{})}
}

// markDone records that seq has finished processing. If this allows the
// contiguous watermark to advance, it returns the new watermark and true.
func (t *cursorTracker) markDone(seq int64) (int64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.initialized {
		t.persisted = seq - 1
		t.initialized = true
	}

	if seq <= t.persisted {
		return 0, false
	}

	t.pending[seq] = struct{}{}

	advanced := false
	for {
		next := t.persisted + 1
		if _, done := t.pending[next]; !done {
			break
		}
		delete(t.pending, next)
		t.persisted = next
		advanced = true
	}

	if advanced {
		return t.persisted, true
	}
	return 0, false
}
