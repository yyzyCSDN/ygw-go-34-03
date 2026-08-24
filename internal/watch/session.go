package watch

func CloseSession(c *Conn) {
	c.finishSession()
}

func (c *Conn) finishSession() {
	c.persistCursor()
	c.Close()
}

func (c *Conn) persistCursor() {
	c.mu.Lock()
	cursor := c.cursor
	c.mu.Unlock()
	c.cursors.Set(c.session, cursor)
}
