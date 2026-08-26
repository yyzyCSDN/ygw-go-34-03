package audit_test

import (
	"testing"

	"confighub/internal/audit"
	"confighub/internal/model"
)

func TestJournalAppendTailCount(t *testing.T) {
	j := audit.New()
	j.Append(audit.NewRecord(model.EventPublish, 1, "b1", "app", "group", "c1"))
	j.Append(audit.NewRecord(model.EventRollback, 2, "b1", "app", "group", "c2"))
	if j.Count() != 2 {
		t.Fatalf("count = %d", j.Count())
	}
	tail := j.Tail(1)
	if len(tail) != 1 || tail[0].Revision != 2 {
		t.Fatalf("tail = %+v", tail)
	}
}

func TestJournalSince(t *testing.T) {
	j := audit.New()
	first := j.Append(audit.NewRecord(model.EventPublish, 1, "b1", "app", "group", "c1"))
	j.Append(audit.NewRecord(model.EventPublish, 2, "b2", "app", "group", "c2"))
	rest := j.Since(first.Seq)
	if len(rest) != 1 || rest[0].Revision != 2 {
		t.Fatalf("since = %+v", rest)
	}
}
