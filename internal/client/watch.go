package client

import (
	"context"

	"confighub/internal/checkpoint"
	"confighub/internal/model"
	"confighub/internal/watch"
)

func (c *Client) Watch(
	ctx context.Context,
	id, session string,
	app model.AppID,
	group model.GroupID,
	cursors *checkpoint.Cursor,
	hub *watch.Hub,
	registry *watch.Registry,
	handler func(model.Event) error,
) (*watch.Conn, error) {
	conn := watch.NewConn(id, session, app, group, c.store, c.versions, cursors, hub.Acks(), handler, 3)
	if err := registry.Register(conn); err != nil {
		return nil, err
	}
	if err := conn.Resume(ctx); err != nil {
		registry.Unregister(conn)
		cursors.Release(session)
		return nil, err
	}
	return conn, nil
}
