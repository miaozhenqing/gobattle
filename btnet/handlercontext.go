package btnet

import "gobattle/btiface"

type HandlerContext struct {
	prev          *HandlerContext
	next          *HandlerContext
	inHandler     btiface.IInHandler
	outHandler    btiface.IOutHandler
	activeHandler btiface.IActiveHandler
	pipeline      *Pipeline
	executor      btiface.IEventLoop
}

func NewHandlerContext(p *Pipeline, h btiface.IHandler, executor btiface.IEventLoop) *HandlerContext {
	if h == nil {
		panic("handler is nil")
	}
	ctx := &HandlerContext{pipeline: p}
	if in, ok := h.(btiface.IInHandler); ok {
		ctx.inHandler = in
	}
	if out, ok := h.(btiface.IOutHandler); ok {
		ctx.outHandler = out
	}
	if active, ok := h.(btiface.IActiveHandler); ok {
		ctx.activeHandler = active
	}

	if ctx.inHandler == nil && ctx.outHandler == nil && ctx.activeHandler == nil {
		panic("handler type error")
	}
	ctx.executor = executor
	return ctx
}

func (hc *HandlerContext) GetConnection() btiface.IConnection {
	return hc.pipeline.GetConnection()
}

func (hc *HandlerContext) FireRead(msg interface{}) {
	next := hc.next
	for next.inHandler == nil {
		next = next.next
	}
	next.invokeRead(hc.executor, msg)
}

func (hc *HandlerContext) invokeRead(el btiface.IEventLoop, msg interface{}) {
	if hc.executor == el {
		hc.inHandler.OnRead(hc, msg)
		return
	}
	hc.executor.Execute(func() {
		if hc.GetConnection().IsClosed() {
			return
		}
		hc.inHandler.OnRead(hc, msg)
	})
}

func (hc *HandlerContext) FireWrite(msg interface{}) {
	prev := hc.prev
	for prev.outHandler == nil {
		prev = prev.prev
	}
	prev.invokeWrite(hc.executor, msg)
}
func (hc *HandlerContext) invokeWrite(el btiface.IEventLoop, msg interface{}) {
	if hc.executor == el {
		hc.outHandler.OnWrite(hc, msg)
		return
	}
	hc.executor.Execute(func() {
		hc.outHandler.OnWrite(hc, msg)
	})
}

func (hc *HandlerContext) FireActive() {
	next := hc.next
	for next.activeHandler == nil {
		next = next.next
	}
	next.invokeActive(hc.executor)
}

func (hc *HandlerContext) invokeActive(executor btiface.IEventLoop) {
	if hc.executor == executor {
		hc.activeHandler.OnActive(hc)
		return
	}
	hc.executor.Execute(func() {
		hc.activeHandler.OnActive(hc)
	})
}

func (hc *HandlerContext) FireInactive() {
	next := hc.next
	for next.activeHandler == nil {
		next = next.next
	}
	next.invokeInactive(hc.executor)
}

func (hc *HandlerContext) invokeInactive(executor btiface.IEventLoop) {
	if hc.executor == executor {
		hc.activeHandler.OnInactive(hc)
		return
	}
	hc.executor.Execute(func() {
		hc.activeHandler.OnInactive(hc)
	})
}
