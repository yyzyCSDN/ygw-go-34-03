package version_test

import (
	"testing"

	"confighub/internal/model"
	"confighub/internal/store"
	"confighub/internal/version"
)

func TestRecordLookupLatest(t *testing.T) {
	table := version.New()
	s := store.New()
	s.Commit(store.NewTxn("b1", "app/group", map[model.Key]string{"k": "v"}, 1))
	snap := s.Snapshot("app", "group")
	table.Record(version.RecordFromSnapshot(1, "app/group", snap))
	rec, ok := table.Lookup(1)
	if !ok {
		t.Fatal("record missing")
	}
	if rec.BatchID != "b1" {
		t.Fatalf("batch id = %s", rec.BatchID)
	}
	if table.Latest() != 1 {
		t.Fatalf("latest = %d", table.Latest())
	}
}

func TestSinceReturnsOrdered(t *testing.T) {
	table := version.New()
	s := store.New()
	for i := 1; i <= 3; i++ {
		s.Commit(store.NewTxn("b", "app/group", map[model.Key]string{"k": "v"}, int64(i)))
		snap := s.Snapshot("app", "group")
		table.Record(version.RecordFromSnapshot(int64(i), "app/group", snap))
	}
	records := table.Since(1)
	if len(records) != 2 || records[0].Revision != 2 || records[1].Revision != 3 {
		t.Fatalf("since returned %+v", records)
	}
}
