package client_test

import (
	"context"
	"testing"
	"time"

	"confighub/internal/audit"
	"confighub/internal/client"
	"confighub/internal/metric"
	"confighub/internal/model"
	"confighub/internal/publish"
	"confighub/internal/store"
	"confighub/internal/version"
	"confighub/internal/watch"
)

func TestGrayRolloutServesNewClients(t *testing.T) {
	st := store.New()
	versions := version.New()
	registry := watch.NewRegistry()
	hub := watch.NewHub(registry)
	journal := audit.New()
	metrics := metric.New()
	grader := publish.NewGrader()
	p := publish.New(st, versions, hub, journal, metrics, grader, time.Second, 3)
	cl := client.New(st, versions, grader, client.NewCache())
	ctx := context.Background()

	b1 := model.NewBatch("app", "group", map[model.Key]string{"k": "old"})
	r1 := p.Publish(ctx, &b1)
	if r1.Error != nil {
		t.Fatalf("seed publish failed: %v", r1.Error)
	}
	grader.Set(&model.GrayPlan{
		App:              "app",
		Group:            "group",
		Percent:          100,
		CapturedRevision: r1.Revision,
		CreatedAt:        time.Now().UTC(),
	})
	b2 := model.NewBatch("app", "group", map[model.Key]string{"k": "new"})
	if res := p.Publish(ctx, &b2); res.Error != nil {
		t.Fatalf("rollout publish failed: %v", res.Error)
	}
	snap, _ := cl.Pull("app", "group", "")
	if got := snap.Entries["k"]; got != "new" {
		t.Fatalf("fresh client served %q, want rollout content %q", got, "new")
	}
}
