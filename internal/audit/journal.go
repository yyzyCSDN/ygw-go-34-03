package audit

import "sync"

type Journal struct {
	mu      sync.Mutex
	records []Record
	seq     int64
}

func New() *Journal {
	return &Journal{}
}

func (j *Journal) Append(rec Record) Record {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.seq++
	rec.Seq = j.seq
	j.records = append(j.records, rec)
	return rec
}

func (j *Journal) Tail(n int) []Record {
	j.mu.Lock()
	defer j.mu.Unlock()
	if n <= 0 || len(j.records) == 0 {
		return nil
	}
	if n > len(j.records) {
		n = len(j.records)
	}
	out := make([]Record, 0, n)
	out = append(out, j.records[len(j.records)-n:]...)
	return out
}

func (j *Journal) Count() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return len(j.records)
}

func (j *Journal) Since(from int64) []Record {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make([]Record, 0)
	for _, rec := range j.records {
		if rec.Seq > from {
			out = append(out, rec)
		}
	}
	return out
}
