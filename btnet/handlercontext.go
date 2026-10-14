package btnet

import "gobattle/btiface"

type HandlerContext struct {
	prev       *HandlerContext
	next       *HandlerContext
	inHandler  btiface.IInHandler
	outHandler btiface.IOutHandler
	pipeline   *Pipeline
}

func NewHandlerContext(p *Pipeline, h btiface.IHandler) *HandlerContext {
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
	if ctx.inHandler == nil && ctx.outHandler == nil {
		panic("handler type error")
	}
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
	next.inHandler.OnRead(next, msg)
}

func (hc *HandlerContext) FireWrite(msg interface{}) {
	prev := hc.prev
	for prev.outHandler == nil {
		prev = prev.prev
	}
	prev.outHandler.OnWrite(prev, msg)
}
