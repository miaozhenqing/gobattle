package handler

import "gobattle/btiface"

type DispatchHandler struct {
	routers map[uint32]btiface.IRouter
}

func NewDispatchHandler() *DispatchHandler {
	return &DispatchHandler{routers: make(map[uint32]btiface.IRouter)}
}

func (d *DispatchHandler) AddRouter(msgId uint32, r btiface.IRouter) {

	if r == nil {
		panic("add router failed, router is nil")
	}
	if _, exist := d.routers[msgId]; exist {
		panic("add router failed, router exists")
	}

	d.routers[msgId] = r
}
