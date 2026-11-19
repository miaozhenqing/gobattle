package api

import (
	"context"
	"gobattle/common/protocol"
	"gobattle/framework/btrpc"
)

type GmApi interface {
	Exec(ctx context.Context, req *protocol.GMReq) (*protocol.GMResp, error)
}

var _ GmApi = (*GmApiClient)(nil)

type GmApiClient struct {
	inv btrpc.Invoker
}

func NewGmApiClient(inv btrpc.Invoker) *GmApiClient {
	return &GmApiClient{inv: inv}
}

func (c *GmApiClient) Exec(ctx context.Context, req *protocol.GMReq) (*protocol.GMResp, error) {
	return btrpc.Invoke[protocol.GMResp](ctx, c.inv, "GmApi.Exec", req)
}
