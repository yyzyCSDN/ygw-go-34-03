package metric

func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"publishes":      m.publishes.Load(),
		"rollbacks":      m.rollbacks.Load(),
		"pulls":          m.pulls.Load(),
		"delivered":      m.delivered.Load(),
		"unacked":        m.unacked.Load(),
		"errors":         m.errors.Load(),
		"evictions":      m.evictions.Load(),
		"avg_latency_ns": m.AverageLatencyNs(),
	}
}
