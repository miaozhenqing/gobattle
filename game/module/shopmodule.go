package module

import (
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/protocol"
	"gobattle/game/service"
)

type ShopModule struct {
	shopService *service.ShopService
}

func NewShopModule(shopService *service.ShopService) *ShopModule {
	return &ShopModule{shopService: shopService}
}

func (m *ShopModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.Handle(r, protocol.MsgShopBuy, m.Buy)
}

func (m *ShopModule) Buy(ctx *btrouter.RequestContext, req *protocol.ShopBuyReq) (*protocol.ShopBuyResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, protocol.CodeNotLogin
	}
	res, err := m.shopService.Buy(ctx, s, req.ShopId)
	if err != nil {
		return nil, err
	}
	return &protocol.ShopBuyResp{ShopId: res.Shop.ShopId, BuyCount: res.Shop.BuyCount, LastBuyTime: res.Shop.LastBuyTime}, nil
}
