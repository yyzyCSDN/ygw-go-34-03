package model

import (
	"errors"
	"testing"
)

func TestCopyEntriesDeepCopies(t *testing.T) {
	src := map[Key]string{"a": "1", "b": "2"}
	dst := CopyEntries(src)
	dst["c"] = "3"
	if _, ok := src["c"]; ok {
		t.Fatal("copy mutated the source map")
	}
	if len(dst) != 3 || len(src) != 2 {
		t.Fatalf("unexpected sizes: src=%d dst=%d", len(src), len(dst))
	}
}

func TestCanonicalKeysSorted(t *testing.T) {
	keys := CanonicalKeys(map[Key]string{"b": "1", "a": "2", "c": "3"})
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("keys not sorted: %v", keys)
	}
}

func TestKeyLineAndNamespaceKey(t *testing.T) {
	if got := KeyLine("timeout.ms", "1500"); got != "timeout.ms=1500" {
		t.Fatalf("unexpected key line: %s", got)
	}
	if got := NamespaceKey("Checkout", "Default"); got != "checkout/default" {
		t.Fatalf("unexpected namespace key: %s", got)
	}
}

func TestEventKindString(t *testing.T) {
	if EventPublish.String() != "publish" {
		t.Fatal("publish kind rendered incorrectly")
	}
	if EventRollback.String() != "rollback" {
		t.Fatal("rollback kind rendered incorrectly")
	}
	if EventDelete.String() != "delete" {
		t.Fatal("delete kind rendered incorrectly")
	}
}

func TestNewBatchHasID(t *testing.T) {
	b := NewBatch("app", "group", map[Key]string{"k": "v"})
	if b.ID == "" {
		t.Fatal("batch id is empty")
	}
	if b.Entries["k"] != "v" {
		t.Fatal("batch entries missing")
	}
}

func TestFailedResultCarriesError(t *testing.T) {
	res := Failed(errors.New("boom"))
	if res.Error == nil {
		t.Fatal("failed result lost its error")
	}
}
