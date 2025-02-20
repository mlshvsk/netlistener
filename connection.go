package netlistener

import (
	"context"
	"net"
)

type throttledConnection struct {
	net.Conn

	config *connectionBandwithConfig
}

func NewThrottledConnection(conn net.Conn, config *connectionBandwithConfig) *throttledConnection {
	return &throttledConnection{
		Conn:   conn,
		config: config,
	}
}

// In a real-world scenario we need to handle the case when the size of the buffer is bigger than the limit
// In that case we would split it by chunks
func (c *throttledConnection) Read(b []byte) (n int, err error) {
	if err := c.config.GlobalReadLimiter().WaitN(context.TODO(), len(b)); err != nil {
		return 0, err
	}

	if c.config.globalConfig.PerConnReadLimit() != c.config.PerConnReadLimiter().Limit() {
		c.config.SetPerConnReadLimit(c.config.globalConfig.perConnReadLimit)
	}

	if err := c.config.PerConnReadLimiter().WaitN(context.TODO(), len(b)); err != nil {
		return 0, err
	}

	return c.Conn.Read(b)
}

// In a real-world scenario we need to handle the case when the size of the buffer is bigger than the limit
// In that case we would split it by chunks
func (c *throttledConnection) Write(b []byte) (n int, err error) {
	if err := c.config.GlobalWriteLimiter().WaitN(context.TODO(), len(b)); err != nil {
		return 0, err
	}

	if c.config.globalConfig.PerConnWriteLimit() != c.config.PerConnWriteLimiter().Limit() {
		c.config.SetPerConnWriteLimit(c.config.globalConfig.perConnReadLimit)
	}

	if err := c.config.PerConnWriteLimiter().WaitN(context.TODO(), len(b)); err != nil {
		return 0, err
	}

	return c.Conn.Write(b)
}
