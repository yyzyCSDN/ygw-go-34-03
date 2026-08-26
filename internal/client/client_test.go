package client_test

import (
	"testing"

	"confighub/internal/client"
	"confighub/internal/model"
	"confighub/internal/publish"
	"confighub/internal/store"
	"confighub/internal/version"
)

func TestPullReturnsSnapshotAndCache(t *testing.T) {
	st := store.New()
	st.Commit(store.NewTxn("b1", "app/group", map[model.Key]string{"k": "v"}, 1))
	versions := version.New()
	grader := publish.NewGrader()
	cache := client.NewCache()
	cl := client.New(st, versions, grader, cache)

	snap, notModified := cl.Pull("app", "group", "")
	if notModified {
		t.Fatal("first pull should not be not-modified")
	}
	if snap.Entries["k"] != "v" {
		t.Fatalf("entries = %v", snap.Entries)
	}
	_, notModified = cl.Pull("app", "group", snap.Checksum)
	if !notModified {
		t.Fatal("etag match should be not-modified")
	}
}

func TestPullDeltaReturnsChanges(t *testing.T) {
	st := store.New()
	versions := version.New()
	grader := publish.NewGrader()
	cl := client.New(st, versions, grader, client.NewCache())
	for i := 1; i <= 2; i++ {
		st.Commit(store.NewTxn("b", "app/group", map[model.Key]string{"k": "v"}, int64(i)))
		snap := st.Snapshot("app", "group")
		versions.Record(version.RecordFromSnapshot(int64(i), "app/group", snap))
	}
	records := cl.PullDelta("app", "group", 1)
	if len(records) != 1 || records[0].Revision != 2 {
		t.Fatalf("delta = %+v", records)
	}
}

func TestCacheApplyEvent(t *testing.T) {
	cache := client.NewCache()
	cache.Store("app", "group", &model.Snapshot{
		App:      "app",
		Group:    "group",
		Revision: 1,
		Entries:  map[model.Key]string{"a": "1"},
	})
	ev := model.Event{
		Revision: 2,
		Kind:     model.EventPublish,
		BatchID:  "b2",
		App:      "app",
		Group:    "group",
		Values:   map[model.Key]string{"b": "2"},
	}
	if err := cache.ApplyEvent(ev); err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	snap, ok := cache.Get("app", "group")
	if !ok || snap.Revision != 2 || snap.Entries["a"] != "1" || snap.Entries["b"] != "2" {
		t.Fatalf("cache state = %+v", snap)
	}
}
