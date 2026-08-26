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

func TestDeleteInvalidatesPullCache(t *testing.T) {
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

	p.Publish(ctx, &model.Batch{ID: "b1", App: "app", Group: "group", Entries: map[model.Key]string{"k": "v"}})
	first, notModified := cl.Pull("app", "group", "")
	if notModified {
		t.Fatal("first pull should be fresh")
	}
	p.Publish(ctx, &model.Batch{ID: "b2", App: "app", Group: "group", DeleteKeys: []model.Key{"k"}})
	second, notModified := cl.Pull("app", "group", first.Checksum)
	if notModified {
		t.Fatal("delete must invalidate the pull cache")
	}
	if _, ok := second.Entries["k"]; ok {
		t.Fatal("deleted key is still served from cache")
	}
}
