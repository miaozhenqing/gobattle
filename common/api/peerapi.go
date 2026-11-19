package api

import (
	"context"
	"gobattle/framework/btrpc"
)

type RegisterPeerReq struct {
	PeerType string `json:"peerType"`
	PeerId   int64  `json:"peerId"`
}

type RegisterPeerResp struct{}

type PeerApi interface {
	Register(ctx context.Context, req *RegisterPeerReq) (*RegisterPeerResp, error)
}

var _ PeerApi = (*PeerApiClient)(nil)

type PeerApiClient struct {
	inv btrpc.Invoker
}

func NewPeerApiClient(inv btrpc.Invoker) *PeerApiClient {
	return &PeerApiClient{inv: inv}
}

func (c *PeerApiClient) Register(ctx context.Context, req *RegisterPeerReq) (*RegisterPeerResp, error) {
	return btrpc.Invoke[RegisterPeerResp](ctx, c.inv, "PeerApi.Register", req)
}
