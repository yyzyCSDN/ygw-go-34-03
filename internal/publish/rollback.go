package publish

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"confighub/internal/audit"
	"confighub/internal/model"
	"confighub/internal/store"
	"confighub/internal/version"
)

var ErrVersionNotFound = errors.New("rollback revision not found")

func (p *Publisher) Rollback(ctx context.Context, app model.AppID, group model.GroupID, to int64) model.Result {
	p.lock()
	defer p.unlock()

	ns := model.NamespaceKey(app, group)
	rec, ok := p.versions.Lookup(to)
	if !ok {
		return model.Failed(ErrVersionNotFound)
	}
	rev := p.st.Revision()
	batchID := uuid.NewString()
	txn := store.NewTxn(batchID, ns, rec.Entries, rev+1)
	_ = p.st.Commit(txn)
	snap := p.st.Snapshot(app, group)
	p.versions.Record(version.RecordFromSnapshot(snap.Revision, ns, snap))

	p.journal.Append(audit.NewRecord(model.EventRollback, snap.Revision, batchID, app, group, snap.Checksum))

	ev := model.Event{
		Revision:  snap.Revision,
		Kind:      model.EventRollback,
		BatchID:   batchID,
		App:       app,
		Group:     group,
		Keys:      model.CanonicalKeys(rec.Entries),
		Values:    model.CopyEntries(rec.Entries),
		CreatedAt: time.Now().UTC(),
	}
	delivered, unacked, err := p.hub.BroadcastAndWait(ctx, ev, p.ackTimeout)
	p.metrics.RecordRollback(1, delivered, unacked)
	return model.Result{
		BatchID:   batchID,
		Revision:  snap.Revision,
		Checksum:  snap.Checksum,
		Delivered: delivered,
		Unacked:   unacked,
		Error:     err,
	}
}
