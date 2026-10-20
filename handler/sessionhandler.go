package handler

import (
	"gobattle/btiface"
	"gobattle/btlog"
)

var _ btiface.IActiveHandler = (*SessionHandler)(nil)

type SessionHandler struct {
}

func NewSessionHandler() *SessionHandler {
	return &SessionHandler{}
}

func (a *SessionHandler) OnActive(ctx btiface.IHandlerContext) {
	btlog.Ins().Infof("conn active, connID:%v", ctx.GetConnection().GetID())
	ctx.FireActive()
}

func (a *SessionHandler) OnInactive(ctx btiface.IHandlerContext) {
	btlog.Ins().Infof("conn inactive, connID:%v", ctx.GetConnection().GetID())
	ctx.FireInactive()
}
