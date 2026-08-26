package store_test

import (
	"testing"

	"confighub/internal/model"
	"confighub/internal/store"
)

func TestCommitAppliesEntries(t *testing.T) {
	s := store.New()
	txn := store.NewTxn("b1", "app/group", map[model.Key]string{"k": "v"}, 1)
	rev := s.Commit(txn)
	if rev != 1 {
		t.Fatalf("revision = %d, want 1", rev)
	}
	snap := s.Snapshot("app", "group")
	if snap.Entries["k"] != "v" {
		t.Fatalf("entry not applied: %v", snap.Entries)
	}
	if s.BatchID() != "b1" {
		t.Fatalf("batch id = %s", s.BatchID())
	}
}

func TestDeleteRemovesEntry(t *testing.T) {
	s := store.New()
	s.Commit(store.NewTxn("b1", "app/group", map[model.Key]string{"a": "1", "b": "2"}, 1))
	s.Delete([]model.Key{"a"})
	snap := s.Snapshot("app", "group")
	if _, ok := snap.Entries["a"]; ok {
		t.Fatal("deleted key still present")
	}
	if snap.Entries["b"] != "2" {
		t.Fatal("unrelated key changed")
	}
}

func TestSnapshotChecksumStable(t *testing.T) {
	s := store.New()
	s.Commit(store.NewTxn("b1", "app/group", map[model.Key]string{"a": "1", "b": "2"}, 1))
	first := s.Snapshot("app", "group")
	second := s.Snapshot("app", "group")
	if first.Checksum == "" || first.Checksum != second.Checksum {
		t.Fatalf("checksum unstable: %q vs %q", first.Checksum, second.Checksum)
	}
}

func TestGetReturnsEntry(t *testing.T) {
	s := store.New()
	s.Commit(store.NewTxn("b1", "app/group", map[model.Key]string{"k": "v"}, 1))
	e, err := s.Get("k")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if e.Value != "v" {
		t.Fatalf("value = %q", e.Value)
	}
	if _, err := s.Get("missing"); err == nil {
		t.Fatal("missing key returned no error")
	}
}

func TestExportContainsNamespace(t *testing.T) {
	s := store.New()
	s.Commit(store.NewTxn("b1", "app/group", map[model.Key]string{"k": "v"}, 1))
	view := s.Export("app", "group")
	if view.Entries["k"] != "v" || view.Revision == 0 {
		t.Fatalf("export incomplete: %+v", view)
	}
}

func TestChecksumForIsDeterministic(t *testing.T) {
	a := store.ChecksumFor(3, "b1", map[model.Key]string{"x": "1"})
	b := store.ChecksumFor(3, "b1", map[model.Key]string{"x": "1"})
	if a != b || a == "" {
		t.Fatalf("checksum mismatch: %q %q", a, b)
	}
}
