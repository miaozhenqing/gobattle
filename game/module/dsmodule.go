package module

import (
	"gobattle/btrouter"
	"gobattle/game/proto"
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
	btrouter.HandlePublic(r, proto.MsgDsRegister, m.Register)
	btrouter.Handle(r, proto.MsgDsHeartBeat, m.HeartBeat)
}

func IsDsRegistered(ctx *btrouter.RequestContext) bool {
	_, ok := service.GetDs(ctx.Conn())
	return ok
}

func (m *DsModule) Register(ctx *btrouter.RequestContext, req *proto.DsRegisterReq) (*proto.DsRegisterResp, error) {
	if net.ParseIP(req.IP) == nil || req.Port <= 0 || req.DsID <= 0 {
		return nil, proto.CodeDsInvalidParam
	}
	ds, err := m.dsService.Register(ctx.Conn(), req.DsID, req.IP, req.Port)
	if err != nil {
		return nil, err
	}
	return &proto.DsRegisterResp{DsID: ds.ID}, nil
}

func (m *DsModule) HeartBeat(ctx *btrouter.RequestContext, _ *proto.DsHeartBeatReq) (*proto.DsHeartBeatResp, error) {
	ds, ok := service.GetDs(ctx.Conn())
	if !ok {
		return nil, proto.CodeDsNotRegistered
	}
	return &proto.DsHeartBeatResp{DsID: ds.ID}, nil
}
