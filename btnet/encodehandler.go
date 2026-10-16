package btnet

import (
	"gobattle/btiface"
	"gobattle/btlog"
)

type EncodeHandler struct {
	s btiface.ISerializer
}

func NewEncodeHandler(s btiface.ISerializer) *EncodeHandler {
	return &EncodeHandler{s: s}
}

func (e *EncodeHandler) OnWrite(ctx btiface.IHandlerContext, msg interface{}) {
	m, ok := msg.(btiface.IMessage)
	if !ok {
		ctx.FireWrite(msg)
		return
	}
	data := e.s.Serialize(m)
	if data == nil {
		btlog.Ins().Errorf("encode failed, connID:%v msgId:%v", ctx.GetConnection().GetID(), m.GetMsgId())
		return
	}
	ctx.FireWrite(data)
}
