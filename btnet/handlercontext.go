package btnet

import (
	"fmt"
	"gobattle/btlog"
	"runtime/debug"
)

const (
	maskRead = 1 << iota
	maskWrite
	maskActive
	maskException
)

type HandlerContext struct {
	prev             *HandlerContext
	next             *HandlerContext
	inHandler        InHandler
	outHandler       OutHandler
	activeHandler    ActiveHandler
	exceptionHandler ExceptionHandler
	pipeline         *Pipeline
	executor         *EventLoop
	mask             int
}

func newHandlerContext(p *Pipeline, h any, executor *EventLoop) *HandlerContext {
	if h == nil {
		panic("handler is nil")
	}
	ctx := &HandlerContext{pipeline: p}
	if in, ok := h.(InHandler); ok {
		ctx.inHandler = in
		ctx.mask |= maskRead
	}
	if out, ok := h.(OutHandler); ok {
		ctx.outHandler = out
		ctx.mask |= maskWrite
	}
	if active, ok := h.(ActiveHandler); ok {
		ctx.activeHandler = active
		ctx.mask |= maskActive
	}
	if ex, ok := h.(ExceptionHandler); ok {
		ctx.exceptionHandler = ex
		ctx.mask |= maskException
	}

	if ctx.mask == 0 {
		panic(fmt.Sprintf("handler type error: %T implements no handler interface", h))
	}
	ctx.executor = executor
	return ctx
}

func (hc *HandlerContext) GetConnection() *Connection {
	return hc.pipeline.GetConnection()
}

// 这里可以找到read active exception
func (hc *HandlerContext) findNext(mask int) *HandlerContext {
	next := hc.next
	for next.mask&mask == 0 {
		next = next.next
	}
	return next
}

// write
func (hc *HandlerContext) findPrev(mask int) *HandlerContext {
	prev := hc.prev
	for prev.mask&mask == 0 {
		prev = prev.prev
	}
	return prev
}

func (hc *HandlerContext) recoverInbound() {
	if r := recover(); r != nil {
		hc.handleException(&PanicError{Value: r, Stack: debug.Stack()})
	}
}

func (hc *HandlerContext) FireRead(msg any) {
	hc.findNext(maskRead).invokeRead(hc.executor, msg)
}

func (hc *HandlerContext) invokeRead(el *EventLoop, msg any) {
	if hc.executor == el {
		hc.callRead(msg)
		return
	}
	hc.executor.Execute(func() {
		if hc.GetConnection().IsClosed() {
			return
		}
		hc.callRead(msg)
	})
}

func (hc *HandlerContext) callRead(msg any) {
	defer hc.recoverInbound()
	hc.inHandler.OnRead(hc, msg)
}

func (hc *HandlerContext) FireWrite(msg any) {
	hc.findPrev(maskWrite).invokeWrite(hc.executor, msg)
}
func (hc *HandlerContext) invokeWrite(el *EventLoop, msg any) {
	if hc.executor == el {
		hc.callWrite(msg)
		return
	}
	hc.executor.Execute(func() {
		hc.callWrite(msg)
	})
}

// 出站的消息异常，先只打个log
func (hc *HandlerContext) callWrite(msg any) {
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Errorf("callWrite panic， connID:%v msg:%T err:%v s:%s", hc.GetConnection().GetID(), msg, r, debug.Stack())
		}
	}()
	hc.outHandler.OnWrite(hc, msg)
}

func (hc *HandlerContext) FireActive() {
	hc.findNext(maskActive).invokeActive(hc.executor)
}

func (hc *HandlerContext) invokeActive(executor *EventLoop) {
	if hc.executor == executor {
		hc.callActive()
		return
	}
	hc.executor.Execute(func() {
		hc.callActive()
	})
}

func (hc *HandlerContext) callActive() {
	defer hc.recoverInbound()
	hc.activeHandler.OnActive(hc)
}

func (hc *HandlerContext) FireInactive() {
	hc.findNext(maskActive).invokeInactive(hc.executor)
}

func (hc *HandlerContext) invokeInactive(executor *EventLoop) {
	if hc.executor == executor {
		hc.callInactive()
		return
	}
	hc.executor.Execute(func() {
		hc.callInactive()
	})
}

func (hc *HandlerContext) callInactive() {
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Errorf("OnInactive panic, connID:%v err:%v\n%s", hc.GetConnection().GetID(), r, debug.Stack())
			hc.FireInactive()
		}
	}()
	hc.activeHandler.OnInactive(hc)
}

func (hc *HandlerContext) FireException(err error) {
	hc.findNext(maskException).invokeException(hc.executor, err)
}

func (hc *HandlerContext) invokeException(executor *EventLoop, err error) {
	if hc.executor == executor {
		hc.handleException(err)
		return
	}
	hc.executor.Execute(func() {
		hc.handleException(err)
	})
}

// 本节点实现了onException就交给它，否则后传
func (hc *HandlerContext) handleException(err error) {
	if hc.exceptionHandler == nil {
		hc.FireException(err)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Errorf("handleException panic, connID:%v, painc:%v, casue:%v, stack:%s", hc.GetConnection().GetID(), r, err, debug.Stack())
		}
	}()
	hc.exceptionHandler.OnException(hc, err)
}
