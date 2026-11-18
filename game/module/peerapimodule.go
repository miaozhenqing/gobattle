package module

import (
	"context"
	"gobattle/framework/btrpc"
	"gobattle/shared/api"
)

var _ api.PeerApi = (*PeerApiModule)(nil)

type PeerApiModule struct {
	server *btrpc.Server
}

func NewPeerApiModule(server *btrpc.Server) *PeerApiModule {
	return &PeerApiModule{server: server}
}

func (m *PeerApiModule) Register(ctx context.Context, req *api.RegisterPeerReq) (*api.RegisterPeerResp, error) {
	if err := m.server.BindPeer(ctx, btrpc.Peer{Type: req.PeerType, Id: req.PeerId}); err != nil {
		return nil, err
	}
	return &api.RegisterPeerResp{}, nil
}
