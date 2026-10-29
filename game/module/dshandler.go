package module

import (
	"gobattle/btlog"
	"gobattle/btnet"
)

var _ btnet.ActiveHandler = (*DsHandler)(nil)

type DsHandler struct {
	dsModule *DsModule
}

func NewDsHandler(dsModule *DsModule) *DsHandler {
	return &DsHandler{dsModule: dsModule}
}

func (h *DsHandler) OnActive(ctx *btnet.HandlerContext) {
	btlog.Ins().Info("dsConnActive", "connId", ctx.GetConnection().GetID())
	ctx.FireActive()
}

func (h *DsHandler) OnInactive(ctx *btnet.HandlerContext) {
	conn := ctx.GetConnection()
	btlog.Ins().Info("dsConnInactive", "connId", conn.GetID())
	if ds, ok := dsKey.Get(conn); ok {
		h.dsModule.unregister(ds)
	}
	ctx.FireInactive()
}
