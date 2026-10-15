package router

import (
	"gobattle/btiface"
	"gobattle/btlog"
)

type PingRouter struct {
	BaseRouter
}

func NewPingRouter() *PingRouter {
	return &PingRouter{}
}

func (b *PingRouter) Handle(req btiface.IRequest) {
	btlog.Ins().Info("req:", string(req.GetData()))
	req.GetConnection().SendMsg(1001, req.GetData())
}
