package api

import (
	"context"
	"gobattle/framework/btrpc"
)

type LoginReq struct {
	Username string `json:"username"`
}

type LoginResp struct {
	UserId   int64  `json:"userId"`
	Username string `json:"username"`
	RoleName string `json:"roleName"`
	IsNew    bool   `json:"isNew"`
}

type UserApi interface {
	Login(ctx context.Context, req *LoginReq) (*LoginResp, error)
}

var _ UserApi = (*UserApiClient)(nil)

type UserApiClient struct {
	inv btrpc.Invoker
}

func NewUserApiClient(inv btrpc.Invoker) *UserApiClient {
	return &UserApiClient{inv: inv}
}

func (c *UserApiClient) Login(ctx context.Context, req *LoginReq) (*LoginResp, error) {
	return btrpc.Invoke[LoginResp](ctx, c.inv, "UserApi.Login", req)
}
