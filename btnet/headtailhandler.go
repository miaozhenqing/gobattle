package btnet

import (
	"gobattle/btiface"
	"gobattle/btlog"
)

// 包内未导出：只有 head 能绕过 pipeline 直接写连接
type rawWriter interface {
	writeRaw(data []byte)
}

// headHandler 出站链终点（tail → ... → head），把序列化后的字节交给连接的写协程
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

// tailHandler 入站链终点（head → ... → tail），走到这里说明没有 handler 消费
type tailHandler struct{}

func (t *tailHandler) OnRead(ctx btiface.IHandlerContext, msg interface{}) {
	btlog.Ins().Warnf("msg not handled, connID:%v type:%T", ctx.GetConnection().GetID(), msg)
}

// 无状态，所有 pipeline 共用
var (
	sharedHead = &headHandler{}
	sharedTail = &tailHandler{}
)
