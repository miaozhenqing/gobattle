package btrouter

import (
	"context"
	"gobattle/btiface"
)

type Context struct {
	context.Context
	conn   btiface.IConnection
	msgId  uint32
	method string
}

func (c *Context) Conn() btiface.IConnection {
	return c.conn
}

func (c *Context) MsgId() uint32 {
	return c.msgId
}

func (c *Context) Method() string {
	return c.method
}
