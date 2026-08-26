package model

import "time"

type EventKind int

const (
	EventPublish EventKind = iota + 1
	EventRollback
	EventDelete
)

func (k EventKind) String() string {
	switch k {
	case EventPublish:
		return "publish"
	case EventRollback:
		return "rollback"
	case EventDelete:
		return "delete"
	default:
		return "unknown"
	}
}

type Event struct {
	Seq       int64
	Revision  int64
	Kind      EventKind
	BatchID   string
	App       AppID
	Group     GroupID
	Keys      []Key
	Values    map[Key]string
	CreatedAt time.Time
}

type Delivery struct {
	Event  Event
	SentAt time.Time
}

type Ack struct {
	ConnID   string
	Revision int64
	Err      error
}
