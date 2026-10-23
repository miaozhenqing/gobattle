package btrouter

import (
	"context"
	"gobattle/btnet"
	"time"
)

type Context struct {
	context.Context
	conn    *btnet.Connection
	msgId   uint32
	method  string
	startAt time.Time // dispatch 收到请求的时间
}

func (c *Context) Conn() *btnet.Connection {
	return c.conn
}

func (c *Context) MsgId() uint32 {
	return c.msgId
}

func (c *Context) Method() string {
	return c.method
}
