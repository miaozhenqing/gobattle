package api

import (
	"context"
	"gobattle/framework/btrpc"
	"gobattle/shared/protocol"
)

type UserApi interface {
	Login(ctx context.Context, req *protocol.LoginReq) (*protocol.LoginResp, error)
}

var _ UserApi = (*UserApiClient)(nil)

type UserApiClient struct {
	inv btrpc.Invoker
}

func NewUserApiClient(inv btrpc.Invoker) *UserApiClient {
	return &UserApiClient{inv: inv}
}

func (c *UserApiClient) Login(ctx context.Context, req *protocol.LoginReq) (*protocol.LoginResp, error) {
	return btrpc.Invoke[protocol.LoginResp](ctx, c.inv, "UserApi.Login", req)
}
