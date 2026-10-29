package service

import (
	"errors"
	"gobattle/btlog"
	"gobattle/game/model"
	"gobattle/game/proto"
	"sync"
	"sync/atomic"
	"time"
)

const (
	battleStartTimeout = 60 * time.Second
	battleMaxDuration  = 120 * time.Minute
	battleTickInterval = time.Second
)

type BattleService struct {
	dsService   *DsService
	pushService *PushService

	mu               sync.Mutex
	battleById       map[int64]*model.BattleModel
	battleIdByUserId map[int64]int64

	//todo battleId要全局唯一
	battleIdSeq atomic.Int64
	tick        *tickLoop
}

func NewBattleService(dsService *DsService, pushService *PushService) *BattleService {
	s := &BattleService{
		dsService:        dsService,
		pushService:      pushService,
		battleById:       make(map[int64]*model.BattleModel),
		battleIdByUserId: make(map[int64]int64),
	}
	s.battleIdSeq.Store(time.Now().UnixMilli())
	s.tick = newTickLoop("battle", battleTickInterval, s.checkTimeout)
	dsService.OnUnregister(s.onDsUnregister)
	return s
}

func (s *BattleService) Run() {
	s.tick.run()
}

func (s *BattleService) Stop() {
	s.tick.stop()
}

func (s *BattleService) nextBattleID() int64 {
	return s.battleIdSeq.Add(1)
}

// 选ds并通知ds开局；玩家还没收到匹配成功，失败时不推战斗结束
func (s *BattleService) Create(gameMode, mapId int, players []*model.BattlePlayer) (*model.BattleModel, error) {
	ds := s.dsService.PickDs()
	if ds == nil {
		btlog.Ins().Warn("matchNoAvailableDs", "gameMode", gameMode, "mapId", mapId, "playerIds", playerIdsOf(players))
		return nil, proto.CodeMatchNoDs
	}

	battle := &model.BattleModel{
		ID:         s.nextBattleID(),
		GameMode:   gameMode,
		MapID:      mapId,
		Ds:         ds,
		Players:    players,
		State:      model.BattleStateWaiting,
		CreateTime: time.Now().UnixMilli(),
	}
	s.mu.Lock()
	s.battleById[battle.ID] = battle
	for _, p := range battle.Players {
		s.battleIdByUserId[p.UserId] = battle.ID
	}
	s.mu.Unlock()

	// 先通知ds，ds收不到这局就开不了
	assign := &proto.DsBattleAssignPush{BattleId: battle.ID, GameMode: gameMode, MapID: mapId, Players: toPlayerInfos(players)}
	if err := s.pushService.PushDs(ds, proto.PushDsBattleAssign, assign); err != nil {
		btlog.Ins().Warn("pushDsBattleAssignError", "battleId", battle.ID, "dsId", ds.ID, "err", err)
		s.mu.Lock()
		s.removeLocked(battle)
		s.mu.Unlock()
		s.dsService.ReleaseDs(ds)
		return nil, proto.CodeMatchNoDs
	}
	return battle, nil
}

func (s *BattleService) GetUserBattle(userId int64) *model.BattleModel {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.battleById[s.battleIdByUserId[userId]]
}

func (s *BattleService) Start(ds *model.DsModel, battleId int64) (*model.BattleModel, error) {
	s.mu.Lock()
	battle, err := s.getDsOwnedLocked(ds, battleId)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if battle.State != model.BattleStateWaiting {
		s.mu.Unlock()
		return nil, proto.CodeBattleStateError
	}
	battle.State = model.BattleStateRunning
	battle.StartTime = time.Now().UnixMilli()
	s.mu.Unlock()

	btlog.Ins().Info("battleStart", "battleId", battle.ID, "dsId", ds.ID, "waitMs", battle.StartTime-battle.CreateTime)
	return battle, nil
}

func (s *BattleService) End(ds *model.DsModel, battleId int64, statistics map[int]int) (*model.BattleModel, error) {
	s.mu.Lock()
	battle, err := s.getDsOwnedLocked(ds, battleId)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	if battle.State != model.BattleStateRunning {
		s.mu.Unlock()
		return nil, proto.CodeBattleStateError
	}
	s.removeLocked(battle)
	s.mu.Unlock()

	s.dsService.ReleaseDs(ds)
	// todo 结算落库
	btlog.Ins().Info("battleEnd", "battleId", battle.ID, "dsId", ds.ID, "gameMode", battle.GameMode,
		"durationMs", time.Now().UnixMilli()-battle.StartTime, "statistics", statistics)
	s.pushBattleEnd(battle, proto.BattleEndNormal)
	return battle, nil
}

func (s *BattleService) getDsOwnedLocked(ds *model.DsModel, battleId int64) (*model.BattleModel, error) {
	battle := s.battleById[battleId]
	if battle == nil || battle.Ds != ds {
		return nil, proto.CodeBattleNotFound
	}
	return battle, nil
}

func (s *BattleService) removeLocked(battle *model.BattleModel) bool {
	if s.battleById[battle.ID] != battle {
		return false
	}
	delete(s.battleById, battle.ID)
	for _, p := range battle.Players {
		if s.battleIdByUserId[p.UserId] == battle.ID {
			delete(s.battleIdByUserId, p.UserId)
		}
	}
	return true
}

func (s *BattleService) abort(battle *model.BattleModel, reason string) {
	s.mu.Lock()
	removed := s.removeLocked(battle)
	state := battle.State
	s.mu.Unlock()
	if !removed {
		return
	}
	s.dsService.ReleaseDs(battle.Ds)
	btlog.Ins().Warn("battleAbort", "battleId", battle.ID, "dsId", battle.Ds.ID, "reason", reason, "state", state)
	s.pushBattleEnd(battle, proto.BattleEndAbort)
}

type timeoutBattle struct {
	battle *model.BattleModel
	reason string
}

func (s *BattleService) checkTimeout() {
	now := time.Now().UnixMilli()

	var timeoutBattles []timeoutBattle
	s.mu.Lock()
	for _, battle := range s.battleById {
		switch {
		case battle.State == model.BattleStateWaiting && now-battle.CreateTime >= battleStartTimeout.Milliseconds():
			timeoutBattles = append(timeoutBattles, timeoutBattle{battle, "startTimeout"})
		case battle.State == model.BattleStateRunning && now-battle.StartTime >= battleMaxDuration.Milliseconds():
			timeoutBattles = append(timeoutBattles, timeoutBattle{battle, "durationExceeded"})
		}
	}
	s.mu.Unlock()

	for _, t := range timeoutBattles {
		s.abort(t.battle, t.reason)
	}
}

func (s *BattleService) onDsUnregister(ds *model.DsModel) {
	var dsBattles []*model.BattleModel
	s.mu.Lock()
	for _, battle := range s.battleById {
		if battle.Ds == ds {
			dsBattles = append(dsBattles, battle)
		}
	}
	s.mu.Unlock()

	for _, battle := range dsBattles {
		s.abort(battle, "dsDisconnect")
	}
}

func (s *BattleService) pushBattleEnd(battle *model.BattleModel, reason int) {
	push := &proto.BattleEndPush{BattleId: battle.ID, Reason: reason}
	for _, p := range battle.Players {
		if err := s.pushService.PushUser(p.UserId, proto.PushBattleEnd, push); err != nil && !errors.Is(err, errUserOffline) {
			btlog.Ins().Warn("pushBattleEndError", "battleId", battle.ID, "userId", p.UserId, "err", err)
		}
	}
}

func toPlayerInfos(players []*model.BattlePlayer) []*proto.MatchPlayerInfo {
	infos := make([]*proto.MatchPlayerInfo, 0, len(players))
	for _, p := range players {
		infos = append(infos, &proto.MatchPlayerInfo{UserId: p.UserId, MatchScore: p.MatchScore, Team: p.Team})
	}
	return infos
}

func playerIdsOf(players []*model.BattlePlayer) []int64 {
	ids := make([]int64, 0, len(players))
	for _, p := range players {
		ids = append(ids, p.UserId)
	}
	return ids
}
