package watch

func CloseSession(c *Conn) {
	c.cursors.Set(c.session, c.store.Revision())
	c.Close()
}
