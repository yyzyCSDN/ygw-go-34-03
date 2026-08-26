package watch_test

import (
	"context"
	"testing"
	"time"

	"confighub/internal/checkpoint"
	"confighub/internal/metric"
	"confighub/internal/model"
	"confighub/internal/store"
	"confighub/internal/version"
	"confighub/internal/watch"
)

func TestDeliverAppliesAndAdvancesCursor(t *testing.T) {
	st := store.New()
	st.Commit(store.NewTxn("b1", "app/group", map[model.Key]string{"k": "v"}, 1))
	cursors := checkpoint.NewCursor()
	acks := make(chan model.Ack, 8)
	conn := watch.NewConn("c1", "s1", "app", "group", st, version.New(), cursors, acks, func(model.Event) error {
		return nil
	}, 3)
	defer conn.Close()
	ev := model.Event{Revision: 1, Kind: model.EventPublish, App: "app", Group: "group", Values: map[model.Key]string{"k": "v"}}
	if err := conn.Deliver(ev); err != nil {
		t.Fatalf("deliver failed: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for conn.AppliedCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if conn.AppliedCount() != 1 {
		t.Fatalf("applied = %d", conn.AppliedCount())
	}
	if conn.Cursor() != 1 {
		t.Fatalf("cursor = %d", conn.Cursor())
	}
}

func TestRetryCountsAttempts(t *testing.T) {
	st := store.New()
	cursors := checkpoint.NewCursor()
	acks := make(chan model.Ack, 8)
	attempts := 0
	conn := watch.NewConn("c2", "s2", "app", "group", st, version.New(), cursors, acks, func(model.Event) error {
		attempts++
		return nil
	}, 3)
	defer conn.Close()
	ev := model.Event{Revision: 2, Kind: model.EventPublish, App: "app", Group: "group"}
	if err := conn.Deliver(ev); err != nil {
		t.Fatalf("deliver failed: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for conn.Attempts() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if conn.Attempts() == 0 {
		t.Fatal("no apply attempt recorded")
	}
}

func TestResumeCatchesUp(t *testing.T) {
	st := store.New()
	st.Commit(store.NewTxn("b1", "app/group", map[model.Key]string{"k": "v"}, 1))
	cursors := checkpoint.NewCursor()
	acks := make(chan model.Ack, 8)
	seen := make([]int64, 0)
	conn := watch.NewConn("c3", "s3", "app", "group", st, version.New(), cursors, acks, func(ev model.Event) error {
		seen = append(seen, ev.Revision)
		return nil
	}, 3)
	defer conn.Close()
	if err := conn.Resume(context.Background()); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if conn.Cursor() != 1 || len(seen) != 1 {
		t.Fatalf("resume cursor=%d seen=%v", conn.Cursor(), seen)
	}
}

func TestRegistryRegisterUnregister(t *testing.T) {
	st := store.New()
	cursors := checkpoint.NewCursor()
	acks := make(chan model.Ack, 8)
	registry := watch.NewRegistry()
	conn := watch.NewConn("c4", "s4", "app", "group", st, version.New(), cursors, acks, func(model.Event) error { return nil }, 3)
	if err := registry.Register(conn); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if registry.Count() != 1 {
		t.Fatalf("count = %d", registry.Count())
	}
	registry.Unregister(conn)
	if registry.Count() != 0 {
		t.Fatal("unregister failed")
	}
}

func TestEvictorSweepsIdleConnections(t *testing.T) {
	st := store.New()
	cursors := checkpoint.NewCursor()
	acks := make(chan model.Ack, 8)
	metrics := metric.New()
	registry := watch.NewRegistry()
	conn := watch.NewConn("c5", "s5", "app", "group", st, version.New(), cursors, acks, func(model.Event) error { return nil }, 3)
	_ = registry.Register(conn)
	evictor := watch.NewEvictor(registry, 10*time.Millisecond, time.Millisecond, metrics)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go evictor.Run(ctx)
	deadline := time.Now().Add(2 * time.Second)
	for registry.Count() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if registry.Count() != 0 {
		t.Fatal("idle connection was not evicted")
	}
}
