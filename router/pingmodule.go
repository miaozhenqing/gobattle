package router

import (
	"gobattle/btrouter"
	"time"
)

const (
	MsgPing uint32 = 1001
)

type PingReq struct {
	Text string `json:"text"`
}

type PingResp struct {
	Text string `json:"text"`
	Time int64  `json:"time"`
}

var ErrEmptyText = btrouter.NewError(1001, "text is empty")

type PingModule struct{}

func (m *PingModule) Routes(r *btrouter.Router) {
	btrouter.Register(r, MsgPing, m.Ping)
}

func (m *PingModule) Ping(ctx *btrouter.Context, req *PingReq) (*PingResp, error) {
	if req.Text == "" {
		return nil, ErrEmptyText
	}
	return &PingResp{Text: req.Text, Time: time.Now().UnixMilli()}, nil
}
