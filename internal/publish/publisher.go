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

	ns := model.NamespaceKey(b.App, b.Group)
	rev := p.st.Revision()
	txn := store.NewTxn(b.ID, ns, b.Entries, rev+1)
	_ = p.st.Commit(txn)
	if len(b.DeleteKeys) > 0 {
		_ = p.st.Delete(b.DeleteKeys)
	}
	snap := p.st.Snapshot(b.App, b.Group)
	p.versions.Record(version.RecordFromSnapshot(snap.Revision, ns, snap))
	p.journal.Append(audit.RecordFromSnapshot(model.EventPublish, b.ID, b.App, b.Group, snap))

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
	delivered := p.hub.Broadcast(ev)
	p.metrics.RecordPublish(1, delivered, 0)
	return model.Result{
		BatchID:   b.ID,
		Revision:  snap.Revision,
		Checksum:  snap.Checksum,
		Delivered: delivered,
		Unacked:   0,
		Error:     nil,
	}
}
