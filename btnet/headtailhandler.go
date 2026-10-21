package btnet

import (
	"errors"
	"gobattle/btiface"
	"gobattle/btlog"
)

var (
	sharedHead = &headHandler{}
	sharedTail = &tailHandler{}
)

type rawWriter interface {
	writeRaw(data []byte)
}

type headHandler struct{}

func (h *headHandler) OnWrite(ctx btiface.IHandlerContext, msg interface{}) {
	data, ok := msg.([]byte)
	if !ok {
		btlog.Ins().Errorf("msg must be []byte, connID:%v type:%T", ctx.GetConnection().GetID(), msg)
		return
	}
	w, ok := ctx.GetConnection().(rawWriter)
	if !ok {
		btlog.Ins().Errorf("connection not writable, connID:%v type:%T", ctx.GetConnection().GetID(), ctx.GetConnection())
		return
	}
	w.writeRaw(data)
}

type tailHandler struct{}

func (t *tailHandler) OnRead(ctx btiface.IHandlerContext, msg interface{}) {
	btlog.Ins().Warnf("msg not handled, connID:%v type:%T", ctx.GetConnection().GetID(), msg)
}

type inactiveNotifier interface {
	inactiveDone()
}

func (t *tailHandler) OnActive(ctx btiface.IHandlerContext) {}
func (t *tailHandler) OnInactive(ctx btiface.IHandlerContext) {
	if n, ok := ctx.GetConnection().(inactiveNotifier); ok {
		n.inactiveDone()
	}
}

// 没有 handler 处理的异常,打日志并关闭连接
func (t *tailHandler) OnException(ctx btiface.IHandlerContext, err error) {
	var pe *PanicError
	if errors.As(err, &pe) {
		btlog.Ins().Errorf("unhandled exception, connID:%v err:%v\n%s", ctx.GetConnection().GetID(), err, pe.Stack)
	} else {
		btlog.Ins().Errorf("unhandled exception, connID:%v err:%v", ctx.GetConnection().GetID(), err)
	}
	ctx.GetConnection().Stop()
}
