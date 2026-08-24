package checkpoint_test

import (
	"testing"

	"confighub/internal/checkpoint"
)

func TestCursorSetGetRelease(t *testing.T) {
	c := checkpoint.NewCursor()
	c.Set("s1", 3)
	if got := c.Get("s1"); got != 3 {
		t.Fatalf("cursor = %d", got)
	}
	c.Release("s1")
	if got := c.Get("s1"); got != 0 {
		t.Fatalf("cursor after release = %d", got)
	}
}

func TestCursorCount(t *testing.T) {
	c := checkpoint.NewCursor()
	c.Set("s1", 1)
	c.Set("s2", 2)
	if c.Count() != 2 {
		t.Fatalf("count = %d", c.Count())
	}
}

func TestTrackerCountsAcks(t *testing.T) {
	tr := checkpoint.NewTracker(2)
	tr.Ack(1, nil)
	if tr.Done() {
		t.Fatal("tracker done too early")
	}
	tr.Ack(1, nil)
	if !tr.Done() {
		t.Fatal("tracker not done")
	}
	if tr.Unacked() != 0 || tr.Failed() != 0 {
		t.Fatalf("unexpected tracker state: unacked=%d failed=%d", tr.Unacked(), tr.Failed())
	}
}
