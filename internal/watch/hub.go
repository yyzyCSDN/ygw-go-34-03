package watch

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"confighub/internal/checkpoint"
	"confighub/internal/model"
)

var ErrAckTimeout = errors.New("watch ack timeout")

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

func (h *Hub) BroadcastAndWait(ctx context.Context, ev model.Event, timeout time.Duration) (int, int, error) {
	conns := h.registry.Snapshot()
	tracker := checkpoint.NewTracker(len(conns))
	if len(conns) == 0 {
		return 0, 0, nil
	}
	accepted := h.Broadcast(ev)
	for i := accepted; i < len(conns); i++ {
		tracker.Ack(ev.Revision, ErrAckTimeout)
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for !tracker.Done() {
		select {
		case ack := <-h.acks:
			tracker.Ack(ack.Revision, ack.Err)
		case <-timer.C:
			return tracker.Failed(), tracker.Unacked(), ErrAckTimeout
		case <-ctx.Done():
			return tracker.Failed(), tracker.Unacked(), ctx.Err()
		}
	}
	return len(conns) - tracker.Failed(), tracker.Unacked(), nil
}
