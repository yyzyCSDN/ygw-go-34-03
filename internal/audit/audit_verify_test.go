package audit_test

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

func TestAuditMatchesAppliedRevision(t *testing.T) {
	st := store.New()
	versions := version.New()
	registry := watch.NewRegistry()
	hub := watch.NewHub(registry)
	journal := audit.New()
	metrics := metric.New()
	grader := publish.NewGrader()
	p := publish.New(st, versions, hub, journal, metrics, grader, time.Second, 3)
	ctx := context.Background()

	p.Publish(ctx, &model.Batch{ID: "b1", App: "app", Group: "group", Entries: map[model.Key]string{"k": "v1"}})
	p.Publish(ctx, &model.Batch{ID: "b2", App: "app", Group: "group", Entries: map[model.Key]string{"k": "v2"}})

	tail := journal.Tail(1)
	if len(tail) != 1 {
		t.Fatalf("audit tail empty")
	}
	snap := st.Snapshot("app", "group")
	rec := tail[0]
	if rec.Revision != snap.Revision {
		t.Fatalf("audit revision %d does not match applied revision %d", rec.Revision, snap.Revision)
	}
	if rec.Checksum != snap.Checksum {
		t.Fatalf("audit checksum %q does not match applied snapshot %q", rec.Checksum, snap.Checksum)
	}
	if rec.BatchID != "b2" {
		t.Fatalf("audit batch %q, want b2", rec.BatchID)
	}
}
