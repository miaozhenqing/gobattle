package module

import (
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/game/service"
)

var _ btnet.ActiveHandler = (*DsConnHandler)(nil)

type DsConnHandler struct {
	dsService *service.DsService
}

func NewDsConnHandler(dsService *service.DsService) *DsConnHandler {
	return &DsConnHandler{dsService: dsService}
}

func (h *DsConnHandler) OnActive(ctx *btnet.HandlerContext) {
	btlog.Ins().Info("dsConnActive", "connId", ctx.GetConnection().GetID())
	ctx.FireActive()
}

func (h *DsConnHandler) OnInactive(ctx *btnet.HandlerContext) {
	conn := ctx.GetConnection()
	btlog.Ins().Info("dsConnInactive", "connId", conn.GetID())
	h.dsService.Unregister(conn)
	ctx.FireInactive()
}
