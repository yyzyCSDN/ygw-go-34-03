package metric_test

import (
	"testing"
	"time"

	"confighub/internal/metric"
)

func TestCountersAndSnapshot(t *testing.T) {
	m := metric.New()
	m.RecordPublish(1, 2, 0)
	m.RecordRollback(1, 1, 0)
	m.RecordPull()
	m.RecordError()
	m.RecordEviction()
	m.RecordLatency(50 * time.Millisecond)
	snap := m.Snapshot()
	if snap["publishes"] != 1 || snap["rollbacks"] != 1 || snap["pulls"] != 1 {
		t.Fatalf("unexpected snapshot: %v", snap)
	}
	if snap["errors"] != 1 || snap["evictions"] != 1 {
		t.Fatalf("unexpected snapshot: %v", snap)
	}
	if snap["avg_latency_ns"] == 0 {
		t.Fatalf("latency not recorded: %v", snap)
	}
}
