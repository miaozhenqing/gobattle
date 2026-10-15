package handler

import (
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/btnet"
)

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

func (d *DispatchHandler) OnRead(ctx btiface.IHandlerContext, msg interface{}) {
	m, ok := msg.(btiface.IMessage)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	router := d.routers[m.GetMsgId()]
	if router == nil {
		btlog.Ins().Errorf("no router, msgId:%v", m.GetMsgId())
		return
	}
	req := btnet.GetRequest(ctx.GetConnection(), m.GetData())
	defer btnet.PutRequest(req)
	//复用了req，需要回收，暂时不能异步（或复制数据）
	router.Handle(req)
}
