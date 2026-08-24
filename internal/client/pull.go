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
	snap = c.applyGray(app, group, snap)
	if isNotModified(etag, snap) {
		if cached, ok := c.cache.Get(app, group); ok && cached.Checksum == etag {
			return cached, true
		}
		return snap, true
	}
	c.cache.Store(app, group, snap)
	return snap, false
}

func (c *Client) applyGray(app model.AppID, group model.GroupID, snap *model.Snapshot) *model.Snapshot {
	plan, ok := c.grader.Get(app, group)
	if !ok || c.grader.Evaluate(plan, 0) {
		return snap
	}
	rec, ok := c.versions.Lookup(plan.CapturedRevision)
	if !ok {
		return snap
	}
	return snapshotFromRecord(app, group, rec)
}

func isNotModified(etag string, snap *model.Snapshot) bool {
	return etag != "" && etag == snap.Checksum
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
