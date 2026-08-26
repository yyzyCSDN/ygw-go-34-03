package version

import (
	"time"

	"confighub/internal/model"
)

type Record struct {
	Revision  int64
	BatchID   string
	Namespace string
	Entries   map[model.Key]string
	Checksum  string
	AppliedAt time.Time
}
