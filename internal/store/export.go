package store

import "confighub/internal/model"

type ExportView struct {
	Revision int64                `json:"revision"`
	BatchID  string               `json:"batch_id"`
	Entries  map[model.Key]string `json:"entries"`
}

func (s *Store) Export(app model.AppID, group model.GroupID) ExportView {
	snap := s.Snapshot(app, group)
	return ExportView{
		Revision: snap.Revision,
		BatchID:  snap.BatchID,
		Entries:  model.CopyEntries(snap.Entries),
	}
}
