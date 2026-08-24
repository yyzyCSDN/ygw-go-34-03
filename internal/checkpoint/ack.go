package checkpoint

import "sync"

type Tracker struct {
	mu       sync.Mutex
	expected int
	received int
	failed   int
}

func NewTracker(expected int) *Tracker {
	return &Tracker{expected: expected}
}

func (t *Tracker) Ack(_ int64, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.received++
	if err != nil {
		t.failed++
	}
}

func (t *Tracker) Done() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.received >= t.expected
}

func (t *Tracker) Failed() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.failed
}

func (t *Tracker) Unacked() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.received >= t.expected {
		return 0
	}
	return t.expected - t.received
}
