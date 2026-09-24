package router

import "gobattle/btiface"

type PingRouter struct {
	BaseRouter
}

func NewPingRouter() *PingRouter {
	return &PingRouter{}
}

func (b *PingRouter) Handle(req btiface.IRequest) {
	println("req:%s", string(req.GetData()))

	req.GetConnection().SendMsg(1, req.GetData())
}
