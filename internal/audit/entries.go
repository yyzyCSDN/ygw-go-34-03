package audit

import (
	"time"

	"confighub/internal/model"
)

type Record struct {
	Seq       int64
	Kind      model.EventKind
	Revision  int64
	BatchID   string
	App       model.AppID
	Group     model.GroupID
	Checksum  string
	SampledAt time.Time
}

func NewRecord(kind model.EventKind, revision int64, batchID string, app model.AppID, group model.GroupID, checksum string) Record {
	return Record{
		Kind:      kind,
		Revision:  revision,
		BatchID:   batchID,
		App:       app,
		Group:     group,
		Checksum:  checksum,
		SampledAt: time.Now().UTC(),
	}
}

func RecordFromSnapshot(kind model.EventKind, batchID string, app model.AppID, group model.GroupID, snap *model.Snapshot) Record {
	return NewRecord(kind, snap.Revision, batchID, app, group, snap.Checksum)
}
