package btrouter

import (
	"context"
	"gobattle/btnet"
	"time"
)

type RequestContext struct {
	context.Context
	conn    *btnet.Connection
	msgId   uint32
	route   *route
	startAt time.Time
}

func (c *RequestContext) Conn() *btnet.Connection {
	return c.conn
}

func (c *RequestContext) MsgId() uint32 {
	return c.msgId
}

func (c *RequestContext) Method() string {
	return c.route.method
}

func (c *RequestContext) IsPublic() bool {
	return c.route.public
}
