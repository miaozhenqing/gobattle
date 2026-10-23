package btnet

type PipelineInitializer func(p *Pipeline)

// handler 至少实现以下接口之一

type InHandler interface {
	OnRead(ctx *HandlerContext, msg any)
}

type OutHandler interface {
	OnWrite(ctx *HandlerContext, msg any)
}

type ActiveHandler interface {
	OnActive(ctx *HandlerContext)
	OnInactive(ctx *HandlerContext)
}

type ExceptionHandler interface {
	OnException(ctx *HandlerContext, err error)
}

type Pipeline struct {
	conn   *Connection
	head   *HandlerContext
	tail   *HandlerContext
	ioLoop *EventLoop
}

func newPipeline(conn *Connection, ioLoop *EventLoop) *Pipeline {
	p := &Pipeline{conn: conn, ioLoop: ioLoop}

	p.head = newHandlerContext(p, sharedHead, ioLoop)
	p.tail = newHandlerContext(p, sharedTail, ioLoop)

	p.head.next = p.tail
	p.tail.prev = p.head

	return p
}

func (p *Pipeline) AddLastGroup(g *EventLoopGroup, h any) *Pipeline {
	return p.addLast(g.Next(p.conn.GetID()), h)
}

func (p *Pipeline) AddLast(h any) *Pipeline {
	return p.addLast(p.ioLoop, h)
}

func (p *Pipeline) FireRead(msg any) {
	p.ioLoop.Execute(func() {
		p.head.FireRead(msg)
	})
}

func (p *Pipeline) FireWrite(msg any) {
	p.ioLoop.Execute(func() {
		p.tail.FireWrite(msg)
	})
}

func (p *Pipeline) fireReadInLoop(msg any) {
	p.head.FireRead(msg)
}

func (p *Pipeline) GetConnection() *Connection {
	return p.conn
}

func (p *Pipeline) addLast(executor *EventLoop, h any) *Pipeline {
	ctx := newHandlerContext(p, h, executor)
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
func (p *Pipeline) fireExceptionInLoop(err error) {
	p.head.FireException(err)
}
