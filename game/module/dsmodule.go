package module

import (
	"gobattle/btrouter"
	"gobattle/game/protocol"
	"gobattle/game/service"
	"net"
)

type DsModule struct {
	dsService *service.DsService
}

func NewDsModule(dsService *service.DsService) *DsModule {
	return &DsModule{dsService: dsService}
}

func (m *DsModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.HandlePublic(r, protocol.MsgDsRegister, m.Register)
	btrouter.Handle(r, protocol.MsgDsHeartBeat, m.HeartBeat)
}

func IsDsRegistered(ctx *btrouter.RequestContext) bool {
	_, ok := service.GetDs(ctx.Conn())
	return ok
}

func (m *DsModule) Register(ctx *btrouter.RequestContext, req *protocol.DsRegisterReq) (*protocol.DsRegisterResp, error) {
	if net.ParseIP(req.IP) == nil || req.Port <= 0 || req.DsId <= 0 {
		return nil, protocol.CodeDsInvalidParam
	}
	ds, err := m.dsService.Register(ctx.Conn(), req.DsId, req.IP, req.Port)
	if err != nil {
		return nil, err
	}
	return &protocol.DsRegisterResp{DsId: ds.Id}, nil
}

func (m *DsModule) HeartBeat(ctx *btrouter.RequestContext, _ *protocol.DsHeartBeatReq) (*protocol.DsHeartBeatResp, error) {
	ds, ok := service.GetDs(ctx.Conn())
	if !ok {
		return nil, protocol.CodeDsNotRegistered
	}
	return &protocol.DsHeartBeatResp{DsId: ds.Id}, nil
}
