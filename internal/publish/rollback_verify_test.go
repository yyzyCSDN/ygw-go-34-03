package publish_test

import (
	"context"
	"testing"
	"time"

	"confighub/internal/audit"
	"confighub/internal/metric"
	"confighub/internal/model"
	"confighub/internal/publish"
	"confighub/internal/store"
	"confighub/internal/version"
	"confighub/internal/watch"
)

func TestRollbackKeepsOriginalBatchLineage(t *testing.T) {
	st := store.New()
	versions := version.New()
	registry := watch.NewRegistry()
	hub := watch.NewHub(registry)
	journal := audit.New()
	metrics := metric.New()
	grader := publish.NewGrader()
	p := publish.New(st, versions, hub, journal, metrics, grader, time.Second, 3)
	ctx := context.Background()

	b1 := model.NewBatch("app", "group", map[model.Key]string{"k": "v1"})
	r1 := p.Publish(ctx, &b1)
	if r1.Error != nil {
		t.Fatalf("publish failed: %v", r1.Error)
	}
	b2 := model.NewBatch("app", "group", map[model.Key]string{"k": "v2"})
	r2 := p.Publish(ctx, &b2)
	if r2.Error != nil {
		t.Fatalf("publish failed: %v", r2.Error)
	}
	r3 := p.Rollback(ctx, "app", "group", r1.Revision)
	if r3.Error != nil {
		t.Fatalf("rollback failed: %v", r3.Error)
	}
	if got := st.BatchID(); got != b1.ID {
		t.Fatalf("rollback batch lineage = %q, want original batch %q", got, b1.ID)
	}
}
