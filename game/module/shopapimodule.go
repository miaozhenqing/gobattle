package module

import (
	"context"
	"gobattle/common/api"
	"gobattle/common/protocol"
	"gobattle/framework/btrpc"
	"gobattle/game/service"
)

var _ api.ShopApi = (*ShopApiModule)(nil)

type ShopApiModule struct {
	shopService *service.ShopService
}

func NewShopApiModule(shopService *service.ShopService) *ShopApiModule {
	return &ShopApiModule{shopService: shopService}
}

func (m *ShopApiModule) Buy(ctx context.Context, req *protocol.ShopBuyReq) (*protocol.ShopBuyResp, error) {
	res, err := m.shopService.Buy(ctx, btrpc.UserId(ctx), req.ShopId)
	if err != nil {
		return nil, err
	}
	return &protocol.ShopBuyResp{ShopId: res.Shop.ShopId, BuyCount: res.Shop.BuyCount, LastBuyTime: res.Shop.LastBuyTime}, nil
}
