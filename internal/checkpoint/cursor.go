package checkpoint

import "sync"

type Cursor struct {
	mu     sync.Mutex
	values map[string]int64
}

func NewCursor() *Cursor {
	return &Cursor{values: make(map[string]int64)}
}

func (c *Cursor) Get(session string) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.values[session]
}

func (c *Cursor) Set(session string, rev int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[session] = rev
}

func (c *Cursor) Release(session string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.values, session)
}

func (c *Cursor) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.values)
}

func (c *Cursor) Snapshot() map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int64, len(c.values))
	for k, v := range c.values {
		out[k] = v
	}
	return out
}
