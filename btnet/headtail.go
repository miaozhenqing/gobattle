package btnet

import (
	"errors"
	"fmt"
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
		btlog.Ins().Error("writeMsgNotFrame", "connId", ctx.GetConnection().GetId(), "type", fmt.Sprintf("%T", msg))
		return
	}
	ctx.GetConnection().writeRaw(data)
}

type tailHandler struct{}

func (t *tailHandler) OnRead(ctx *HandlerContext, msg any) {
	btlog.Ins().Warn("msgNotHandled", "connId", ctx.GetConnection().GetId(), "type", fmt.Sprintf("%T", msg))
}

func (t *tailHandler) OnActive(ctx *HandlerContext) {}
func (t *tailHandler) OnInactive(ctx *HandlerContext) {
	ctx.GetConnection().inactiveDone()
}

// 没有 handler 处理的异常,打日志并关闭连接
func (t *tailHandler) OnException(ctx *HandlerContext, err error) {
	var pe *PanicError
	if errors.As(err, &pe) {
		btlog.Ins().Error("unhandledException", "connId", ctx.GetConnection().GetId(), "err", err, "stack", string(pe.Stack))
	} else {
		btlog.Ins().Error("unhandledException", "connId", ctx.GetConnection().GetId(), "err", err)
	}
	ctx.GetConnection().Stop()
}
