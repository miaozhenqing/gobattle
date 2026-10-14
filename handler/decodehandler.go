package handler

import (
	"gobattle/btiface"
	"gobattle/btlog"
)

type DecodeHandler struct {
	s btiface.ISerializer
}

func NewDecodeHandler(s btiface.ISerializer) *DecodeHandler {
	return &DecodeHandler{s: s}
}

func (e *DecodeHandler) OnRead(ctx btiface.IHandlerContext, msg interface{}) {
	b, ok := msg.([]byte)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	m, err := e.s.Deserialize(b)
	if err != nil {
		btlog.Ins().Errorf("decode failed, connID:%v, err:%v", ctx.GetConnection().GetID(), err)
		return
	}
	ctx.FireRead(m)

}
