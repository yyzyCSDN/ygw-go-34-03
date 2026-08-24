package watch

import (
	"context"
	"time"

	"confighub/internal/model"
)

func (c *Conn) Catchup(ctx context.Context, from int64) error {
	ns := model.NamespaceKey(c.app, c.group)
	lastRev := from
	for _, rec := range c.versions.Since(from) {
		if rec.Namespace != ns {
			continue
		}
		ev := model.Event{
			Revision:  rec.Revision,
			Kind:      model.EventPublish,
			BatchID:   rec.BatchID,
			App:       c.app,
			Group:     c.group,
			Keys:      model.CanonicalKeys(rec.Entries),
			Values:    model.CopyEntries(rec.Entries),
			CreatedAt: rec.AppliedAt,
		}
		if err := c.ApplyEvent(ev); err != nil {
			return err
		}
		lastRev = rec.Revision
	}
	snap := c.store.Snapshot(c.app, c.group)
	if snap.Revision <= lastRev {
		return nil
	}
	ev := model.Event{
		Revision:  snap.Revision,
		Kind:      model.EventPublish,
		BatchID:   snap.BatchID,
		App:       c.app,
		Group:     c.group,
		Keys:      model.CanonicalKeys(snap.Entries),
		Values:    model.CopyEntries(snap.Entries),
		CreatedAt: time.Now().UTC(),
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return c.ApplyEvent(ev)
}

func (c *Conn) Resume(ctx context.Context) error {
	from := c.cursors.Get(c.session)
	return c.Catchup(ctx, from)
}
