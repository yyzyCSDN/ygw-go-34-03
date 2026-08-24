package model

import (
	"time"

	"github.com/google/uuid"
)

type GrayPlan struct {
	App              AppID
	Group            GroupID
	Percent          int
	ExcludedGroups   []GroupID
	CapturedRevision int64
	CreatedAt        time.Time
}

type Batch struct {
	ID         string
	App        AppID
	Group      GroupID
	Entries    map[Key]string
	DeleteKeys []Key
	CreatedAt  time.Time
}

func NewBatch(app AppID, group GroupID, entries map[Key]string) Batch {
	return Batch{
		ID:        uuid.NewString(),
		App:       app,
		Group:     group,
		Entries:   CopyEntries(entries),
		CreatedAt: time.Now().UTC(),
	}
}

type Result struct {
	BatchID   string
	Revision  int64
	Checksum  string
	Delivered int
	Unacked   int
	Error     error
}

func Failed(err error) Result {
	return Result{Error: err}
}
