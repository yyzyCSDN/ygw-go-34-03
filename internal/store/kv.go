package store

import (
	"errors"
	"sync"

	"confighub/internal/model"
)

var ErrNoEntry = errors.New("config entry not found")

type Store struct {
	mu         sync.RWMutex
	entries    map[model.Key]model.Entry
	revision   int64
	batchID    string
	namespaces map[string]struct{}
}

func New() *Store {
	return &Store{
		entries:    make(map[model.Key]model.Entry),
		namespaces: make(map[string]struct{}),
	}
}

func (s *Store) Get(key model.Key) (model.Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[key]
	if !ok {
		return model.Entry{}, ErrNoEntry
	}
	return e, nil
}

func (s *Store) Revision() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.revision
}

func (s *Store) BatchID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.batchID
}

func (s *Store) EntryCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

func (s *Store) Delete(keys []model.Key) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteLocked(keys)
	if len(keys) == 0 {
		return s.revision
	}
	s.revision++
	return s.revision
}

func (s *Store) deleteLocked(keys []model.Key) {
	for _, k := range keys {
		delete(s.entries, k)
	}
}
