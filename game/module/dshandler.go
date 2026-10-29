package module

import (
	"gobattle/btlog"
	"gobattle/btnet"
)

var _ btnet.ActiveHandler = (*DsSessionHandler)(nil)

type DsSessionHandler struct {
	dsModule *DsModule
}

func NewDsSessionHandler(dsModule *DsModule) *DsSessionHandler {
	return &DsSessionHandler{dsModule: dsModule}
}

func (h *DsSessionHandler) OnActive(ctx *btnet.HandlerContext) {
	btlog.Ins().Info("dsConnActive", "connId", ctx.GetConnection().GetID())
	ctx.FireActive()
}

func (h *DsSessionHandler) OnInactive(ctx *btnet.HandlerContext) {
	conn := ctx.GetConnection()
	btlog.Ins().Info("dsConnInactive", "connId", conn.GetID())
	if ds, ok := dsKey.Get(conn); ok {
		h.dsModule.unregister(ds)
	}
	ctx.FireInactive()
}
