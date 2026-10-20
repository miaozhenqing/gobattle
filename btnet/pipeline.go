package btnet

import (
	"gobattle/btiface"
)

type Pipeline struct {
	conn   btiface.IConnection
	head   *HandlerContext
	tail   *HandlerContext
	ioLoop btiface.IEventLoop
}

func NewPipeline(conn btiface.IConnection, ioLoop btiface.IEventLoop) *Pipeline {
	p := &Pipeline{conn: conn, ioLoop: ioLoop}

	p.head = NewHandlerContext(p, sharedHead, ioLoop)
	p.tail = NewHandlerContext(p, sharedTail, ioLoop)

	p.head.next = p.tail
	p.tail.prev = p.head

	return p
}

func (p *Pipeline) AddLastGroup(g btiface.IEventLoopGroup, h interface{}) btiface.IPipeline {
	return p.addLast(g.Next(p.GetConnection().GetID()), h)
}

func (p *Pipeline) AddLast(h interface{}) btiface.IPipeline {
	return p.addLast(p.ioLoop, h)
}

func (p *Pipeline) FireRead(msg interface{}) {
	p.ioLoop.Execute(func() {
		p.head.FireRead(msg)
	})
}

func (p *Pipeline) FireWrite(msg interface{}) {
	p.ioLoop.Execute(func() {
		p.tail.FireWrite(msg)
	})
}

func (p *Pipeline) fireReadInLoop(msg interface{}) {
	p.head.FireRead(msg)
}

func (p *Pipeline) GetConnection() btiface.IConnection {
	return p.conn
}

func (p *Pipeline) addLast(executor btiface.IEventLoop, h interface{}) btiface.IPipeline {
	ctx := NewHandlerContext(p, h, executor)
	prev := p.tail.prev
	prev.next = ctx
	ctx.prev = prev
	ctx.next = p.tail
	p.tail.prev = ctx
	return p
}

func (p *Pipeline) fireActiveInLoop() {
	p.head.FireActive()
}
func (p *Pipeline) fireInactiveInLoop() {
	p.head.FireInactive()
}
