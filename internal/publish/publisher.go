package publish

import (
	"context"
	"time"

	"confighub/internal/audit"
	"confighub/internal/metric"
	"confighub/internal/model"
	"confighub/internal/store"
	"confighub/internal/version"
	"confighub/internal/watch"
)

type Publisher struct {
	mu         chan struct{}
	st         *store.Store
	versions   *version.Table
	hub        *watch.Hub
	journal    *audit.Journal
	metrics    *metric.Metrics
	grader     *Grader
	ackTimeout time.Duration
	maxRetries int
}

func New(
	st *store.Store,
	versions *version.Table,
	hub *watch.Hub,
	journal *audit.Journal,
	metrics *metric.Metrics,
	grader *Grader,
	ackTimeout time.Duration,
	maxRetries int,
) *Publisher {
	return &Publisher{
		mu:         make(chan struct{}, 1),
		st:         st,
		versions:   versions,
		hub:        hub,
		journal:    journal,
		metrics:    metrics,
		grader:     grader,
		ackTimeout: ackTimeout,
		maxRetries: maxRetries,
	}
}

func (p *Publisher) lock() {
	p.mu <- struct{}{}
}

func (p *Publisher) unlock() {
	<-p.mu
}

func (p *Publisher) Publish(ctx context.Context, b *model.Batch) model.Result {
	if err := Validate(b); err != nil {
		return model.Failed(err)
	}
	p.lock()
	defer p.unlock()
	snap := p.apply(b)
	p.recordAudit(model.EventPublish, b, snap)
	return p.notify(ctx, b, snap)
}

func (p *Publisher) apply(b *model.Batch) *model.Snapshot {
	ns := model.NamespaceKey(b.App, b.Group)
	p.commit(b, ns)
	snap := p.st.Snapshot(b.App, b.Group)
	p.recordVersion(ns, snap)
	return snap
}

func (p *Publisher) commit(b *model.Batch, ns string) {
	rev := p.st.Revision()
	txn := store.NewTxn(b.ID, ns, b.Entries, rev+1)
	_ = p.st.Commit(txn)
	if len(b.DeleteKeys) > 0 {
		_ = p.st.Delete(b.DeleteKeys)
	}
}

func (p *Publisher) recordVersion(ns string, snap *model.Snapshot) {
	p.versions.Record(version.RecordFromSnapshot(snap.Revision, ns, snap))
}

func (p *Publisher) recordAudit(kind model.EventKind, b *model.Batch, snap *model.Snapshot) {
	p.journal.Append(audit.RecordFromSnapshot(kind, b.ID, b.App, b.Group, snap))
}

func (p *Publisher) notify(ctx context.Context, b *model.Batch, snap *model.Snapshot) model.Result {
	ev := model.Event{
		Revision:  snap.Revision,
		Kind:      model.EventPublish,
		BatchID:   b.ID,
		App:       b.App,
		Group:     b.Group,
		Keys:      model.CanonicalKeys(b.Entries),
		Values:    model.CopyEntries(b.Entries),
		CreatedAt: time.Now().UTC(),
	}
	delivered, unacked, err := p.hub.BroadcastAndWait(ctx, ev, p.ackTimeout)
	p.metrics.RecordPublish(1, delivered, unacked)
	return model.Result{
		BatchID:   b.ID,
		Revision:  snap.Revision,
		Checksum:  snap.Checksum,
		Delivered: delivered,
		Unacked:   unacked,
		Error:     err,
	}
}
