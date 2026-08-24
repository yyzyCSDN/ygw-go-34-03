package watch_test

import (
	"errors"
	"testing"
	"time"

	"confighub/internal/checkpoint"
	"confighub/internal/model"
	"confighub/internal/store"
	"confighub/internal/version"
	"confighub/internal/watch"
)

func TestFailedDeliveryKeepsCursorBehind(t *testing.T) {
	st := store.New()
	cursors := checkpoint.NewCursor()
	acks := make(chan model.Ack, 16)
	conn := watch.NewConn("c1", "s1", "app", "group", st, version.New(), cursors, acks, func(model.Event) error {
		return errors.New("apply failed")
	}, 3)
	defer conn.Close()

	ev := model.Event{
		Revision: 2,
		Kind:     model.EventPublish,
		BatchID:  "b2",
		App:      "app",
		Group:    "group",
		Values:   map[model.Key]string{"k": "v"},
	}
	if err := conn.Deliver(ev); err != nil {
		t.Fatalf("deliver failed: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for conn.Attempts() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if conn.Attempts() < 2 {
		t.Fatal("delivery was not retried")
	}
	if got := conn.Cursor(); got != 0 {
		t.Fatalf("cursor advanced to %d although the ack failed", got)
	}
}
