package firehose

import "testing"

func TestCursorTrackerOutOfOrder(t *testing.T) {
	tracker := newCursorTracker()

	// Anchor the watermark at 99 (as if resuming from a saved cursor),
	// so seq 100-102 below are genuinely out of order relative to it.
	if _, ok := tracker.markDone(99); !ok {
		t.Fatalf("expected initial seq to advance")
	}

	if _, ok := tracker.markDone(102); ok {
		t.Fatalf("expected no advance when seq 100 not yet done")
	}

	if _, ok := tracker.markDone(101); ok {
		t.Fatalf("expected no advance when seq 100 still not done")
	}

	advanced, ok := tracker.markDone(100)
	if !ok {
		t.Fatalf("expected advance once contiguous run 100-102 completes")
	}
	if advanced != 102 {
		t.Fatalf("expected watermark 102, got %d", advanced)
	}
}

func TestCursorTrackerLazyInit(t *testing.T) {
	tracker := newCursorTracker()

	advanced, ok := tracker.markDone(500)
	if !ok {
		t.Fatalf("expected first observed seq to immediately advance the watermark")
	}
	if advanced != 500 {
		t.Fatalf("expected watermark 500, got %d", advanced)
	}
}

func TestCursorTrackerIgnoresReplays(t *testing.T) {
	tracker := newCursorTracker()

	if _, ok := tracker.markDone(10); !ok {
		t.Fatalf("expected initial seq to advance")
	}

	if _, ok := tracker.markDone(10); ok {
		t.Fatalf("expected replayed seq to not advance watermark again")
	}

	if _, ok := tracker.markDone(5); ok {
		t.Fatalf("expected stale seq below watermark to be ignored")
	}
}

func TestCursorTrackerSequentialAdvance(t *testing.T) {
	tracker := newCursorTracker()

	for seq := int64(1); seq <= 5; seq++ {
		advanced, ok := tracker.markDone(seq)
		if !ok {
			t.Fatalf("expected advance at seq %d", seq)
		}
		if advanced != seq {
			t.Fatalf("expected watermark %d, got %d", seq, advanced)
		}
	}
}
