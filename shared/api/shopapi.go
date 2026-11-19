package api

import (
	"context"
	"gobattle/framework/btrpc"
	"gobattle/shared/protocol"
)

type ShopApi interface {
	Buy(ctx context.Context, req *protocol.ShopBuyReq) (*protocol.ShopBuyResp, error)
}

var _ ShopApi = (*ShopApiClient)(nil)

type ShopApiClient struct {
	inv btrpc.Invoker
}

func NewShopApiClient(inv btrpc.Invoker) *ShopApiClient {
	return &ShopApiClient{inv: inv}
}

func (c *ShopApiClient) Buy(ctx context.Context, req *protocol.ShopBuyReq) (*protocol.ShopBuyResp, error) {
	return btrpc.Invoke[protocol.ShopBuyResp](ctx, c.inv, "ShopApi.Buy", req)
}
