package btnet

import (
	"gobattle/btiface"
)

type Pipeline struct {
	conn btiface.IConnection
	head *HandlerContext
	tail *HandlerContext
}

func NewPipeline(conn btiface.IConnection) *Pipeline {
	p := &Pipeline{conn: conn}

	// head 只处理出站（写连接），tail 只处理入站（兜底），传播循环靠它们终止
	p.head = NewHandlerContext(p, sharedHead)
	p.tail = NewHandlerContext(p, sharedTail)

	p.head.next = p.tail
	p.tail.prev = p.head
	return p
}

func (p *Pipeline) AddLast(h interface{}) btiface.IPipeline {
	ctx := NewHandlerContext(p, h)
	prev := p.tail.prev
	prev.next = ctx
	ctx.prev = prev
	ctx.next = p.tail
	p.tail.prev = ctx
	return p
}

func (p *Pipeline) FireRead(msg interface{}) {
	p.head.FireRead(msg)
}

func (p *Pipeline) FireWrite(msg interface{}) {
	p.tail.FireWrite(msg)
}

func (p *Pipeline) GetConnection() btiface.IConnection {
	return p.conn
}
