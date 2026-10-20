package btnet

import (
	"gobattle/btiface"
	"gobattle/btlog"
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
func (t *tailHandler) OnActive(ctx btiface.IHandlerContext)   {}
func (t *tailHandler) OnInactive(ctx btiface.IHandlerContext) {}

var (
	sharedHead = &headHandler{}
	sharedTail = &tailHandler{}
)
