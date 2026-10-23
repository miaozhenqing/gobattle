package module

import (
	"gobattle/btrouter"
	"gobattle/example/proto"
	"time"
)

type PingModule struct{}

func (m *PingModule) RegisterRoutes(r *btrouter.Router) {
	btrouter.Handle(r, proto.MsgPing, m.Ping)
}

func (m *PingModule) Ping(ctx *btrouter.RequestContext, req *proto.PingReq) (*proto.PingResp, error) {
	if req.Text == "" {
		return nil, proto.CodeEmptyText
	}
	return &proto.PingResp{Text: req.Text, Time: time.Now().UnixMilli()}, nil
}
