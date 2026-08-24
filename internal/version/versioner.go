package version

import (
	"sync"
	"time"

	"confighub/internal/model"
)

type Table struct {
	mu      sync.RWMutex
	records map[int64]Record
	latest  int64
}

func New() *Table {
	return &Table{records: make(map[int64]Record)}
}

func (t *Table) Record(rec Record) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[rec.Revision] = rec
	if rec.Revision > t.latest {
		t.latest = rec.Revision
	}
}

func (t *Table) Lookup(revision int64) (Record, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	rec, ok := t.records[revision]
	return rec, ok
}

func (t *Table) Latest() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.latest
}

func (t *Table) Since(from int64) []Record {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]Record, 0)
	for rev, rec := range t.records {
		if rev > from {
			out = append(out, rec)
		}
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].Revision > out[j].Revision; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func RecordFromSnapshot(rev int64, namespace string, snap *model.Snapshot) Record {
	return Record{
		Revision:  rev,
		BatchID:   snap.BatchID,
		Namespace: namespace,
		Entries:   model.CopyEntries(snap.Entries),
		Checksum:  snap.Checksum,
		AppliedAt: time.Now().UTC(),
	}
}
