package store

import (
	"time"

	"confighub/internal/model"
)

func (s *Store) Snapshot(app model.AppID, group model.GroupID) *model.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshotLocked(app, group)
}

func (s *Store) snapshotLocked(app model.AppID, group model.GroupID) *model.Snapshot {
	entries := make(map[model.Key]string, len(s.entries))
	for k, e := range s.entries {
		entries[k] = e.Value
	}
	snap := &model.Snapshot{
		App:        app,
		Group:      group,
		Revision:   s.revision,
		BatchID:    s.batchID,
		Entries:    entries,
		CapturedAt: time.Now().UTC(),
	}
	snap.Checksum = checksumOf(s.revision, s.batchID, entries)
	return snap
}
