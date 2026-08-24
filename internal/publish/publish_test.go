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

func TestValidateRejectsEmptyBatch(t *testing.T) {
	if err := publish.Validate(&model.Batch{}); err == nil {
		t.Fatal("empty batch accepted")
	}
	b := model.NewBatch("app", "group", map[model.Key]string{"k": "v"})
	if err := publish.Validate(&b); err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
}

func TestValidatePercent(t *testing.T) {
	if err := publish.ValidatePercent(50); err != nil {
		t.Fatal("valid percent rejected")
	}
	if err := publish.ValidatePercent(101); err == nil {
		t.Fatal("invalid percent accepted")
	}
}

func TestPublishAndRollbackRoundTrip(t *testing.T) {
	st := store.New()
	versions := version.New()
	registry := watch.NewRegistry()
	hub := watch.NewHub(registry)
	journal := audit.New()
	metrics := metric.New()
	grader := publish.NewGrader()
	p := publish.New(st, versions, hub, journal, metrics, grader, time.Second, 3)

	b1 := model.NewBatch("app", "group", map[model.Key]string{"k": "v1"})
	res1 := p.Publish(context.Background(), &b1)
	if res1.Error != nil {
		t.Fatalf("publish failed: %v", res1.Error)
	}
	b2 := model.NewBatch("app", "group", map[model.Key]string{"k": "v2"})
	res2 := p.Publish(context.Background(), &b2)
	if res2.Error != nil || res2.Revision <= res1.Revision {
		t.Fatalf("second publish failed: %+v", res2)
	}
	res3 := p.Rollback(context.Background(), "app", "group", res1.Revision)
	if res3.Error != nil {
		t.Fatalf("rollback failed: %v", res3.Error)
	}
	snap := st.Snapshot("app", "group")
	if snap.Entries["k"] != "v1" {
		t.Fatalf("rollback did not restore content: %v", snap.Entries)
	}
	if journal.Count() != 3 {
		t.Fatalf("audit count = %d", journal.Count())
	}
}

func TestGrayEvaluateBasics(t *testing.T) {
	grader := publish.NewGrader()
	plan := &model.GrayPlan{
		App:              "app",
		Group:            "group",
		Percent:          100,
		CapturedRevision: 5,
	}
	if !grader.Evaluate(plan) {
		t.Fatal("full rollout should be eligible")
	}
	plan.Percent = 0
	if grader.Evaluate(plan) {
		t.Fatal("zero percent rollout should be ineligible")
	}
	plan.Percent = 100
	plan.ExcludedGroups = []model.GroupID{"group"}
	if grader.Evaluate(plan) {
		t.Fatal("excluded group should be ineligible")
	}
}

// A namespace with no prior history (cursor 0) must be judged by the same
// rule as one that has already advanced past the captured baseline. The old
// implementation compared the client cursor against CapturedRevision and
// stranded brand-new clients on the old revision forever.
func TestGrayEvaluateIgnoresClientHistory(t *testing.T) {
	grader := publish.NewGrader()
	plan := &model.GrayPlan{
		App:              "app",
		Group:            "group",
		Percent:          100,
		CapturedRevision: 5,
	}
	if !grader.Evaluate(plan) {
		t.Fatal("new client with no history must still be eligible under full rollout")
	}
}

func TestGraderListAndRemove(t *testing.T) {
	grader := publish.NewGrader()
	grader.Set(&model.GrayPlan{App: "app", Group: "group", Percent: 50})
	if len(grader.List()) != 1 {
		t.Fatalf("plans = %d", len(grader.List()))
	}
	grader.Remove("app", "group")
	if len(grader.List()) != 0 {
		t.Fatal("plan not removed")
	}
}
