package btnet

import "gobattle/btiface"

type DispatchHandler struct {
	d btiface.IDispatcher
}

func NewDispatchHandler(d btiface.IDispatcher) *DispatchHandler {
	if d == nil {
		panic("dispatcher is nil")
	}
	return &DispatchHandler{d: d}
}

func (h *DispatchHandler) OnRead(ctx btiface.IHandlerContext, msg interface{}) {
	m, ok := msg.(btiface.IMessage)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	h.d.Dispatch(NewRequest(ctx.GetConnection(), m))
}
