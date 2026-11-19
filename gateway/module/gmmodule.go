package module

import (
	"gobattle/framework/btrouter"
	"gobattle/shared/api"
	"gobattle/shared/protocol"
)

type GmModule struct {
	gmApi api.GmApi
}

func NewGmModule(gmApi api.GmApi) *GmModule {
	return &GmModule{gmApi: gmApi}
}

func (m *GmModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.Handle(r, protocol.MsgGM, m.Exec)
}

func (m *GmModule) Exec(ctx *btrouter.RequestContext, req *protocol.GMReq) (*protocol.GMResp, error) {
	return m.gmApi.Exec(rpcCtx(ctx), req)
}
