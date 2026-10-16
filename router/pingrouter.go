package router

import (
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/util"
)

var _ btiface.IRouter = (*PingRouter)(nil)

type PingRouter struct{}

func NewPingRouter() *PingRouter {
	return &PingRouter{}
}

func (b *PingRouter) Handle(req btiface.IRequest) {
	btlog.Ins().Info("req:", string(req.GetData()))
	req.GetConnection().SendMsg(util.MessageIDPing, req.GetData())
}
