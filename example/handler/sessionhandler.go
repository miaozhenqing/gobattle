package handler

import (
	"gobattle/btlog"
	"gobattle/btnet"
)

var _ btnet.ActiveHandler = (*SessionHandler)(nil)

type SessionHandler struct {
}

func NewSessionHandler() *SessionHandler {
	return &SessionHandler{}
}

func (a *SessionHandler) OnActive(ctx *btnet.HandlerContext) {
	btlog.Ins().Infof("conn active, connID:%v", ctx.GetConnection().GetID())
	ctx.FireActive()
}

func (a *SessionHandler) OnInactive(ctx *btnet.HandlerContext) {
	btlog.Ins().Infof("conn inactive, connID:%v", ctx.GetConnection().GetID())
	ctx.FireInactive()
}
