package module

import (
	"gobattle/common/api"
	"gobattle/common/protocol"
	"gobattle/framework/btrouter"
)

type ShopModule struct {
	shopApi api.ShopApi
}

func NewShopModule(shopApi api.ShopApi) *ShopModule {
	return &ShopModule{shopApi: shopApi}
}

func (m *ShopModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.Handle(r, protocol.MsgShopBuy, m.Buy)
}

func (m *ShopModule) Buy(ctx *btrouter.RequestContext, req *protocol.ShopBuyReq) (*protocol.ShopBuyResp, error) {
	return m.shopApi.Buy(rpcCtx(ctx), req)
}
