package btnet

import (
	"errors"
	"gobattle/btlog"
)

var (
	sharedHead = &headHandler{}
	sharedTail = &tailHandler{}
)

type headHandler struct{}

// 只接收已编码好的完整帧，只读，可能被多个连接共享，不能原地修改
func (h *headHandler) OnWrite(ctx *HandlerContext, msg any) {
	data, ok := msg.([]byte)
	if !ok {
		btlog.Ins().Errorf("msg must be encoded frame []byte, connID:%v type:%T", ctx.GetConnection().GetID(), msg)
		return
	}
	ctx.GetConnection().writeRaw(data)
}

type tailHandler struct{}

func (t *tailHandler) OnRead(ctx *HandlerContext, msg any) {
	btlog.Ins().Warnf("msg not handled, connID:%v type:%T", ctx.GetConnection().GetID(), msg)
}

func (t *tailHandler) OnActive(ctx *HandlerContext) {}
func (t *tailHandler) OnInactive(ctx *HandlerContext) {
	ctx.GetConnection().inactiveDone()
}

// 没有 handler 处理的异常,打日志并关闭连接
func (t *tailHandler) OnException(ctx *HandlerContext, err error) {
	var pe *PanicError
	if errors.As(err, &pe) {
		btlog.Ins().Errorf("unhandled exception, connID:%v err:%v\n%s", ctx.GetConnection().GetID(), err, pe.Stack)
	} else {
		btlog.Ins().Errorf("unhandled exception, connID:%v err:%v", ctx.GetConnection().GetID(), err)
	}
	ctx.GetConnection().Stop()
}
