package client

import (
	"sync"
	"time"

	"confighub/internal/model"
	"confighub/internal/store"
)

type Cache struct {
	mu     sync.RWMutex
	values map[string]*model.Snapshot
}

func NewCache() *Cache {
	return &Cache{values: make(map[string]*model.Snapshot)}
}

func (c *Cache) Get(app model.AppID, group model.GroupID) (*model.Snapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snap, ok := c.values[model.NamespaceKey(app, group)]
	return snap, ok
}

func (c *Cache) Store(app model.AppID, group model.GroupID, snap *model.Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[model.NamespaceKey(app, group)] = snap
}

func (c *Cache) Revision(app model.AppID, group model.GroupID) int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snap, ok := c.values[model.NamespaceKey(app, group)]
	if !ok {
		return 0
	}
	return snap.Revision
}

func (c *Cache) ApplyEvent(ev model.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := model.NamespaceKey(ev.App, ev.Group)
	cur, ok := c.values[key]
	if ok && eventStale(cur, ev) {
		return nil
	}
	snap := snapshotFromEvent(cur, ev)
	c.values[key] = snap
	return nil
}

func snapshotFromEvent(cur *model.Snapshot, ev model.Event) *model.Snapshot {
	entries := make(map[model.Key]string)
	if cur != nil {
		entries = model.CopyEntries(cur.Entries)
	}
	for _, k := range ev.Keys {
		delete(entries, k)
	}
	for k, v := range ev.Values {
		entries[k] = v
	}
	snap := &model.Snapshot{
		App:        ev.App,
		Group:      ev.Group,
		Revision:   ev.Revision,
		BatchID:    ev.BatchID,
		Entries:    entries,
		CapturedAt: time.Now().UTC(),
	}
	snap.Checksum = store.ChecksumOfSnapshot(snap)
	return snap
}

func eventStale(cur *model.Snapshot, ev model.Event) bool {
	return ev.Revision <= cur.Revision
}
