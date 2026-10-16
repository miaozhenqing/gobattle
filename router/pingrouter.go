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
	if err := req.GetConnection().SendMsg(util.MessageIDPing, req.GetData()); err != nil {
		btlog.Ins().Warnf("send ping failed, connID:%v, err:%v", req.GetConnection().GetID(), err)
	}
}
