package api

import (
	"context"
	"gobattle/common/protocol"
	"gobattle/framework/btrpc"
)

type ItemApi interface {
	List(ctx context.Context, req *protocol.ItemListReq) (*protocol.ItemListResp, error)
}

var _ ItemApi = (*ItemApiClient)(nil)

type ItemApiClient struct {
	inv btrpc.Invoker
}

func NewItemApiClient(inv btrpc.Invoker) *ItemApiClient {
	return &ItemApiClient{inv: inv}
}

func (c *ItemApiClient) List(ctx context.Context, req *protocol.ItemListReq) (*protocol.ItemListResp, error) {
	return btrpc.Invoke[protocol.ItemListResp](ctx, c.inv, "ItemApi.List", req)
}
