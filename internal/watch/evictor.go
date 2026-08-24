package watch

import (
	"context"
	"time"

	"confighub/internal/metric"
)

type Evictor struct {
	registry *Registry
	interval time.Duration
	idle     time.Duration
	metrics  *metric.Metrics
}

func NewEvictor(registry *Registry, interval, idle time.Duration, metrics *metric.Metrics) *Evictor {
	return &Evictor{
		registry: registry,
		interval: interval,
		idle:     idle,
		metrics:  metrics,
	}
}

func (e *Evictor) Run(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.sweep()
		case <-ctx.Done():
			return
		}
	}
}

func (e *Evictor) sweep() {
	for _, c := range e.registry.Snapshot() {
		if time.Since(c.LastActive()) > e.idle {
			e.registry.Unregister(c)
			e.metrics.RecordEviction()
		}
	}
}
