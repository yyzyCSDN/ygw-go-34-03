package watch

import (
	"errors"
	"sync"
)

var ErrDuplicateConn = errors.New("connection already registered")

type Registry struct {
	mu    sync.RWMutex
	conns map[string]*Conn
}

func NewRegistry() *Registry {
	return &Registry{conns: make(map[string]*Conn)}
}

func (r *Registry) Register(c *Conn) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.conns[c.id]; ok {
		return ErrDuplicateConn
	}
	r.conns[c.id] = c
	return nil
}

func (r *Registry) Unregister(c *Conn) {
	if !r.hasConn(c.id) {
		return
	}
	r.removeConn(c)
	CloseSession(c)
}

func (r *Registry) removeConn(c *Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.conns, c.id)
}

func (r *Registry) hasConn(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.conns[id]
	return ok
}

func (r *Registry) Snapshot() []*Conn {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Conn, 0, len(r.conns))
	for _, c := range r.conns {
		out = append(out, c)
	}
	return out
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.conns)
}
