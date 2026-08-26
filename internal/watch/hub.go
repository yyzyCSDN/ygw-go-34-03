package watch

import (
	"sync/atomic"
	"time"

	"confighub/internal/model"
)

type Hub struct {
	registry *Registry
	acks     chan model.Ack
	seq      atomic.Int64
}

func NewHub(registry *Registry) *Hub {
	return &Hub{
		registry: registry,
		acks:     make(chan model.Ack, 64),
	}
}

func (h *Hub) Acks() chan model.Ack {
	return h.acks
}

func (h *Hub) Broadcast(ev model.Event) int {
	seq := h.seq.Add(1)
	ev.Seq = seq
	accepted := 0
	for _, c := range h.registry.Snapshot() {
		if err := c.Deliver(ev); err == nil {
			accepted++
		}
	}
	return accepted
}

func (h *Hub) BroadcastAndWait(_ interface{ Done() <-chan struct{} }, ev model.Event, _ time.Duration) (int, int, error) {
	delivered := h.Broadcast(ev)
	return delivered, 0, nil
}
