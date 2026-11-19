package module

import (
	"gobattle/common/api"
	"gobattle/common/protocol"
	"gobattle/framework/btrouter"
)

type ItemModule struct {
	itemApi api.ItemApi
}

func NewItemModule(itemApi api.ItemApi) *ItemModule {
	return &ItemModule{itemApi: itemApi}
}

func (m *ItemModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.Handle(r, protocol.MsgItemList, m.List)
}

func (m *ItemModule) List(ctx *btrouter.RequestContext, req *protocol.ItemListReq) (*protocol.ItemListResp, error) {
	return m.itemApi.List(rpcCtx(ctx), req)
}
