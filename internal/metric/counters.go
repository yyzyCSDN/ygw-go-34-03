package metric

import (
	"sync/atomic"
	"time"
)

type Metrics struct {
	publishes atomic.Int64
	rollbacks atomic.Int64
	pulls     atomic.Int64
	delivered atomic.Int64
	unacked   atomic.Int64
	errors    atomic.Int64
	evictions atomic.Int64
	latencyNs atomic.Int64
	latencyN  atomic.Int64
}

func New() *Metrics {
	return &Metrics{}
}

func (m *Metrics) RecordPublish(n int, delivered, unacked int) {
	m.publishes.Add(int64(n))
	m.delivered.Add(int64(delivered))
	m.unacked.Add(int64(unacked))
}

func (m *Metrics) RecordRollback(n int, delivered, unacked int) {
	m.rollbacks.Add(int64(n))
	m.delivered.Add(int64(delivered))
	m.unacked.Add(int64(unacked))
}

func (m *Metrics) RecordPull() {
	m.pulls.Add(1)
}

func (m *Metrics) RecordError() {
	m.errors.Add(1)
}

func (m *Metrics) RecordEviction() {
	m.evictions.Add(1)
}

func (m *Metrics) RecordLatency(d time.Duration) {
	m.latencyNs.Add(d.Nanoseconds())
	m.latencyN.Add(1)
}

func (m *Metrics) AverageLatencyNs() int64 {
	n := m.latencyN.Load()
	if n == 0 {
		return 0
	}
	return m.latencyNs.Load() / n
}
