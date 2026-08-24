package client

import (
	"confighub/internal/model"
	"confighub/internal/publish"
	"confighub/internal/store"
	"confighub/internal/version"
)

type Client struct {
	store    *store.Store
	versions *version.Table
	grader   *publish.Grader
	cache    *Cache
}

func New(st *store.Store, versions *version.Table, grader *publish.Grader, cache *Cache) *Client {
	return &Client{
		store:    st,
		versions: versions,
		grader:   grader,
		cache:    cache,
	}
}

func (c *Client) Pull(app model.AppID, group model.GroupID, etag string) (*model.Snapshot, bool) {
	snap := c.store.Snapshot(app, group)
	if plan, ok := c.grader.Get(app, group); ok && !c.grader.Evaluate(plan, c.cache.Revision(app, group)) {
		if rec, ok := c.versions.Lookup(plan.CapturedRevision); ok {
			snap = snapshotFromRecord(app, group, rec)
		}
	}
	if etag != "" && etag == snap.Checksum {
		if cached, ok := c.cache.Get(app, group); ok && cached.Checksum == etag {
			return cached, true
		}
		return snap, true
	}
	c.cache.Store(app, group, snap)
	return snap, false
}

func (c *Client) PullDelta(app model.AppID, group model.GroupID, from int64) []version.Record {
	ns := model.NamespaceKey(app, group)
	records := c.versions.Since(from)
	out := make([]version.Record, 0, len(records))
	for _, rec := range records {
		if rec.Namespace == ns {
			out = append(out, rec)
		}
	}
	return out
}

func (c *Client) ApplyEvent(ev model.Event) error {
	return c.cache.ApplyEvent(ev)
}

func snapshotFromRecord(app model.AppID, group model.GroupID, rec version.Record) *model.Snapshot {
	snap := &model.Snapshot{
		App:      app,
		Group:    group,
		Revision: rec.Revision,
		BatchID:  rec.BatchID,
		Entries:  model.CopyEntries(rec.Entries),
	}
	snap.Checksum = store.ChecksumFor(snap.Revision, snap.BatchID, snap.Entries)
	return snap
}
