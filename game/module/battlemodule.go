package module

import (
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/proto"
	"gobattle/game/service"
)

type BattleModule struct {
	matchService  *service.MatchService
	battleService *service.BattleService
}

func NewBattleModule(matchService *service.MatchService, battleService *service.BattleService) *BattleModule {
	return &BattleModule{matchService: matchService, battleService: battleService}
}

// 客户端协议
func (m *BattleModule) RegisterRoutes(r *btrouter.RouterHandler) {
	btrouter.Handle(r, proto.MsgMatchStart, m.StartMatch)
	btrouter.Handle(r, proto.MsgMatchCancel, m.CancelMatch)
	btrouter.Handle(r, proto.MsgBattleQuery, m.QueryBattle)
}

// ds协议
func (m *BattleModule) RegisterDsRoutes(r *btrouter.RouterHandler) {
	btrouter.Handle(r, proto.MsgDsBattleStart, m.DsBattleStart)
	btrouter.Handle(r, proto.MsgDsBattleEnd, m.DsBattleEnd)
}

func (m *BattleModule) StartMatch(ctx *btrouter.RequestContext, req *proto.MatchStartReq) (*proto.MatchStartResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, proto.CodeNotLogin
	}
	if err := m.matchService.Join(s, req.GameMode, req.MapID); err != nil {
		return nil, err
	}
	return &proto.MatchStartResp{GameMode: req.GameMode, MapID: req.MapID}, nil
}

func (m *BattleModule) CancelMatch(ctx *btrouter.RequestContext, _ *proto.MatchCancelReq) (*proto.MatchCancelResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, proto.CodeNotLogin
	}
	if err := m.matchService.Cancel(s); err != nil {
		return nil, err
	}
	return &proto.MatchCancelResp{}, nil
}

func (m *BattleModule) QueryBattle(ctx *btrouter.RequestContext, _ *proto.BattleQueryReq) (*proto.BattleQueryResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, proto.CodeNotLogin
	}
	battle := m.battleService.GetUserBattle(s.UserId())
	if battle == nil {
		return &proto.BattleQueryResp{}, nil
	}
	return &proto.BattleQueryResp{
		BattleId: battle.ID,
		GameMode: battle.GameMode,
		MapID:    battle.MapID,
		DsIP:     battle.Ds.IP,
		DsPort:   battle.Ds.Port,
		State:    battle.State,
	}, nil
}

func (m *BattleModule) DsBattleStart(ctx *btrouter.RequestContext, req *proto.DsBattleStartReq) (*proto.DsBattleStartResp, error) {
	ds, ok := service.GetDs(ctx.Conn())
	if !ok {
		return nil, proto.CodeDsNotRegistered
	}
	battle, err := m.battleService.Start(ds, req.BattleId)
	if err != nil {
		return nil, err
	}
	return &proto.DsBattleStartResp{BattleId: battle.ID}, nil
}

func (m *BattleModule) DsBattleEnd(ctx *btrouter.RequestContext, req *proto.DsBattleEndReq) (*proto.DsBattleEndResp, error) {
	ds, ok := service.GetDs(ctx.Conn())
	if !ok {
		return nil, proto.CodeDsNotRegistered
	}
	battle, err := m.battleService.End(ds, req.BattleId, req.Statistics)
	if err != nil {
		return nil, err
	}
	return &proto.DsBattleEndResp{BattleId: battle.ID}, nil
}
