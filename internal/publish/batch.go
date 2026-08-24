package publish

import (
	"errors"

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
