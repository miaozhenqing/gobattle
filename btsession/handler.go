package btsession

import (
	"gobattle/btlog"
	"gobattle/btnet"
)

var _ btnet.ActiveHandler = (*SessionHandler)(nil)

type SessionHandler struct {
	mgr *Manager
}

func NewSessionHandler(mgr *Manager) *SessionHandler {
	return &SessionHandler{mgr: mgr}
}

func (h *SessionHandler) OnActive(ctx *btnet.HandlerContext) {
	btlog.Ins().Info("connActive", "connId", ctx.GetConnection().GetId())
	ctx.FireActive()
}

func (h *SessionHandler) OnInactive(ctx *btnet.HandlerContext) {
	btlog.Ins().Info("connInactive", "connId", ctx.GetConnection().GetId())
	h.mgr.unbind(ctx.GetConnection())
	ctx.FireInactive()
}
