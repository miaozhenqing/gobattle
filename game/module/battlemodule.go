package module

import (
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/protocol"
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
	btrouter.Handle(r, protocol.MsgMatchStart, m.StartMatch)
	btrouter.Handle(r, protocol.MsgMatchCancel, m.CancelMatch)
	btrouter.Handle(r, protocol.MsgBattleQuery, m.QueryBattle)
}

// ds协议
func (m *BattleModule) RegisterDsRoutes(r *btrouter.RouterHandler) {
	btrouter.Handle(r, protocol.MsgDsBattleStart, m.DsBattleStart)
	btrouter.Handle(r, protocol.MsgDsBattleEnd, m.DsBattleEnd)
}

func (m *BattleModule) StartMatch(ctx *btrouter.RequestContext, req *protocol.MatchStartReq) (*protocol.MatchStartResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, protocol.CodeNotLogin
	}
	if err := m.matchService.Join(s, req.GameMode, req.MapId); err != nil {
		return nil, err
	}
	return &protocol.MatchStartResp{GameMode: req.GameMode, MapId: req.MapId}, nil
}

func (m *BattleModule) CancelMatch(ctx *btrouter.RequestContext, _ *protocol.MatchCancelReq) (*protocol.MatchCancelResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, protocol.CodeNotLogin
	}
	if err := m.matchService.Cancel(s); err != nil {
		return nil, err
	}
	return &protocol.MatchCancelResp{}, nil
}

func (m *BattleModule) QueryBattle(ctx *btrouter.RequestContext, _ *protocol.BattleQueryReq) (*protocol.BattleQueryResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, protocol.CodeNotLogin
	}
	battle := m.battleService.GetUserBattle(s.UserId())
	if battle == nil {
		return &protocol.BattleQueryResp{}, nil
	}
	return &protocol.BattleQueryResp{
		BattleId: battle.Id,
		GameMode: battle.GameMode,
		MapId:    battle.MapId,
		DsIP:     battle.Ds.IP,
		DsPort:   battle.Ds.Port,
		State:    battle.State,
	}, nil
}

func (m *BattleModule) DsBattleStart(ctx *btrouter.RequestContext, req *protocol.DsBattleStartReq) (*protocol.DsBattleStartResp, error) {
	ds, ok := service.GetDs(ctx.Conn())
	if !ok {
		return nil, protocol.CodeDsNotRegistered
	}
	battle, err := m.battleService.Start(ds, req.BattleId)
	if err != nil {
		return nil, err
	}
	return &protocol.DsBattleStartResp{BattleId: battle.Id}, nil
}

func (m *BattleModule) DsBattleEnd(ctx *btrouter.RequestContext, req *protocol.DsBattleEndReq) (*protocol.DsBattleEndResp, error) {
	ds, ok := service.GetDs(ctx.Conn())
	if !ok {
		return nil, protocol.CodeDsNotRegistered
	}
	battle, err := m.battleService.End(ds, req.BattleId, req.Statistics)
	if err != nil {
		return nil, err
	}
	return &protocol.DsBattleEndResp{BattleId: battle.Id}, nil
}
