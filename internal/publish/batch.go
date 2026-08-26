package publish

import (
	"errors"
	"time"

	"confighub/internal/model"
)

var ErrEmptyBatch = errors.New("batch has no entries and no deletions")

var ErrBadKey = errors.New("batch contains an empty key")

func Validate(b *model.Batch) error {
	if b == nil {
		return ErrEmptyBatch
	}
	if len(b.Entries) == 0 && len(b.DeleteKeys) == 0 {
		return ErrEmptyBatch
	}
	for k := range b.Entries {
		if k == "" {
			return ErrBadKey
		}
	}
	return nil
}

func RestoreBatch(originalID string, app model.AppID, group model.GroupID, entries map[model.Key]string) model.Batch {
	return model.Batch{
		ID:        originalID,
		App:       app,
		Group:     group,
		Entries:   model.CopyEntries(entries),
		CreatedAt: time.Now().UTC(),
	}
}
