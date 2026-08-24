package watch

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"confighub/internal/checkpoint"
	"confighub/internal/model"
	"confighub/internal/store"
	"confighub/internal/version"
)

var ErrConnClosed = errors.New("watch connection closed")

type Conn struct {
	id        string
	session   string
	app       model.AppID
	group     model.GroupID
	store     *store.Store
	versions  *version.Table
	cursors   *checkpoint.Cursor
	acks      chan model.Ack
	handler   func(model.Event) error
	maxTry    int
	queue     chan model.Delivery
	done      chan struct{}
	closeOnce sync.Once
	inflight  sync.WaitGroup
	mu        sync.Mutex
	cursor    int64
	applied   int64
	attempts  atomic.Int64
	lastAt    atomic.Int64
	created   time.Time
}

func NewConn(
	id, session string,
	app model.AppID,
	group model.GroupID,
	st *store.Store,
	versions *version.Table,
	cursors *checkpoint.Cursor,
	acks chan model.Ack,
	handler func(model.Event) error,
	maxRetries int,
) *Conn {
	c := &Conn{
		id:       id,
		session:  session,
		app:      app,
		group:    group,
		store:    st,
		versions: versions,
		cursors:  cursors,
		acks:     acks,
		handler:  handler,
		maxTry:   maxRetries,
		queue:    make(chan model.Delivery, 8),
		done:     make(chan struct{}),
		created:  time.Now().UTC(),
	}
	go c.applyLoop()
	return c
}

func (c *Conn) ID() string {
	return c.id
}

func (c *Conn) Session() string {
	return c.session
}

func (c *Conn) Deliver(ev model.Event) error {
	c.mu.Lock()
	if ev.Revision > c.cursor {
		c.cursor = ev.Revision
	}
	c.mu.Unlock()
	select {
	case c.queue <- model.Delivery{Event: ev, SentAt: time.Now().UTC()}:
		return nil
	case <-c.done:
		return ErrConnClosed
	}
}

func (c *Conn) ApplyEvent(ev model.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.handler(ev); err != nil {
		return err
	}
	c.cursor = ev.Revision
	c.applied++
	c.lastAt.Store(time.Now().UnixNano())
	c.cursors.Set(c.session, ev.Revision)
	select {
	case c.acks <- model.Ack{ConnID: c.id, Revision: ev.Revision}:
	default:
	}
	return nil
}

func (c *Conn) Cursor() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cursor
}

func (c *Conn) AppliedCount() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.applied
}

func (c *Conn) Attempts() int64 {
	return c.attempts.Load()
}

func (c *Conn) LastActive() time.Time {
	ns := c.lastAt.Load()
	if ns == 0 {
		return c.created
	}
	return time.Unix(0, ns)
}

func (c *Conn) Close() {
	c.inflight.Wait()
	c.closeOnce.Do(func() {
		close(c.done)
	})
}

func (c *Conn) applyLoop() {
	for {
		select {
		case d := <-c.queue:
			c.applyOne(d)
		case <-c.done:
			return
		}
	}
}

func (c *Conn) applyOne(d model.Delivery) error {
	c.inflight.Add(1)
	defer c.inflight.Done()
	var lastErr error
	for attempt := 0; attempt < c.maxTry; attempt++ {
		c.attempts.Add(1)
		lastErr = c.ApplyEvent(d.Event)
		if lastErr == nil {
			return nil
		}
		time.Sleep(backoff(attempt, d.Event.Revision))
	}
	select {
	case c.acks <- model.Ack{ConnID: c.id, Revision: d.Event.Revision, Err: lastErr}:
	default:
	}
	return lastErr
}

func backoff(attempt int, revision int64) time.Duration {
	base := time.Duration(attempt+1) * 10 * time.Millisecond
	jitter := time.Duration(revision%7) * time.Millisecond
	return base + jitter
}
