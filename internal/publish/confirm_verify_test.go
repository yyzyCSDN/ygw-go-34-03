package publish_test

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

func TestPublishWaitsForWatchAck(t *testing.T) {
	st := store.New()
	versions := version.New()
	registry := watch.NewRegistry()
	hub := watch.NewHub(registry)
	journal := audit.New()
	metrics := metric.New()
	grader := publish.NewGrader()
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	conn := watch.NewConn("c1", "s1", "app", "group", st, versions, checkpoint.NewCursor(), hub.Acks(), func(model.Event) error {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return nil
	}, 3)
	if err := registry.Register(conn); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	defer registry.Unregister(conn)
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()

	p := publish.New(st, versions, hub, journal, metrics, grader, 2*time.Second, 3)
	b := model.NewBatch("app", "group", map[model.Key]string{"k": "v"})
	done := make(chan struct{})
	go func() {
		p.Publish(context.Background(), &b)
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("delivery did not start")
	}
	select {
	case <-done:
		t.Fatal("publish returned before the watcher acked")
	case <-time.After(300 * time.Millisecond):
	}
	close(release)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("publish did not finish after the ack")
	}
}
