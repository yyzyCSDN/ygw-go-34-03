package store

import "confighub/internal/model"

type Txn struct {
	BatchID   string
	Namespace string
	Writes    map[model.Key]model.Entry
}

func NewTxn(batchID, namespace string, writes map[model.Key]string, version int64) *Txn {
	t := &Txn{
		BatchID:   batchID,
		Namespace: namespace,
		Writes:    make(map[model.Key]model.Entry, len(writes)),
	}
	for k, v := range writes {
		t.Writes[k] = model.Entry{Key: k, Value: v, Version: version}
	}
	return t
}

func (s *Store) Commit(t *Txn) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k := range t.Writes {
		s.entries[k] = t.Writes[k]
	}
	s.revision++
	s.batchID = t.BatchID
	s.namespaces[t.Namespace] = struct{}{}
	return s.revision
}
