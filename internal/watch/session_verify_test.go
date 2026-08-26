package watch_test

import (
	"context"
	"testing"
	"time"

	"confighub/internal/audit"
	"confighub/internal/checkpoint"
	"confighub/internal/metric"
	"confighub/internal/model"
	"confighub/internal/publish"
	"confighub/internal/store"
	"confighub/internal/version"
	"confighub/internal/watch"
)

func TestSessionCloseKeepsCatchupPoint(t *testing.T) {
	st := store.New()
	versions := version.New()
	cursors := checkpoint.NewCursor()
	registry := watch.NewRegistry()
	hub := watch.NewHub(registry)
	journal := audit.New()
	metrics := metric.New()
	grader := publish.NewGrader()
	p := publish.New(st, versions, hub, journal, metrics, grader, time.Second, 3)
	ctx := context.Background()
	acks := make(chan model.Ack, 16)

	p.Publish(ctx, &model.Batch{ID: "b1", App: "app", Group: "group", Entries: map[model.Key]string{"k": "v1"}})
	conn1 := watch.NewConn("c1", "s1", "app", "group", st, versions, cursors, acks, func(model.Event) error { return nil }, 3)
	if err := conn1.ApplyEvent(model.Event{
		Revision: 1,
		Kind:     model.EventPublish,
		BatchID:  "b1",
		App:      "app",
		Group:    "group",
		Values:   map[model.Key]string{"k": "v1"},
	}); err != nil {
		t.Fatalf("apply failed: %v", err)
	}

	p.Publish(ctx, &model.Batch{ID: "b2", App: "app", Group: "group", Entries: map[model.Key]string{"k": "v2"}})
	watch.CloseSession(conn1)
	p.Publish(ctx, &model.Batch{ID: "b3", App: "app", Group: "group", Entries: map[model.Key]string{"k": "v3"}})

	seen := make([]int64, 0, 4)
	conn2 := watch.NewConn("c2", "s1", "app", "group", st, versions, cursors, acks, func(ev model.Event) error {
		seen = append(seen, ev.Revision)
		return nil
	}, 3)
	defer conn2.Close()
	if err := conn2.Resume(ctx); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	for _, rev := range seen {
		if rev == 2 {
			return
		}
	}
	t.Fatalf("intermediate revision 2 missing after reconnect, seen=%v", seen)
}
