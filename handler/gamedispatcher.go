package handler

import (
	"gobattle/btiface"
	"gobattle/btlog"
)

type GameDispatcher struct {
	routers map[uint32]btiface.IRouter
}

func NewGameDispatcher() *GameDispatcher {
	return &GameDispatcher{routers: make(map[uint32]btiface.IRouter)}
}

func (d *GameDispatcher) AddRouter(msgId uint32, r btiface.IRouter) {
	if r == nil {
		panic("add router failed, router is nil")
	}
	if _, exist := d.routers[msgId]; exist {
		panic("add router failed, router exists")
	}

	d.routers[msgId] = r
}

func (d *GameDispatcher) Dispatch(req btiface.IRequest) {
	router := d.routers[req.GetMsgId()]
	if router == nil {
		btlog.Ins().Errorf("no router, connID:%v msgId:%v", req.GetConnection().GetID(), req.GetMsgId())
		return
	}
	router.Handle(req)
}
