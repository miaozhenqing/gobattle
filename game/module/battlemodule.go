package module

import (
	"cmp"
	"context"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/model"
	"gobattle/game/proto"
	"maps"
	"net"
	"runtime/debug"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultMatchScore = 1000
	matchTimeout      = 30 * time.Second
	matchTickInterval = time.Second
)

var modePlayerCount = map[int]int{
	1: 2, // 1v1
	2: 4, // 2v2
}

type matchPoolKey struct {
	gameMode int
	mapID    int
}

type MatchPlayer struct {
	userId     int64
	matchScore int
	poolKey    matchPoolKey
	joinTime   time.Time
	conn       *btnet.Connection
}

type matchGroup struct {
	poolKey matchPoolKey
	players []*MatchPlayer
}

type BattleModule struct {
	deps     *Deps
	dsModule *DsModule

	matchMu             sync.Mutex
	matchPools          map[matchPoolKey]map[int64]*MatchPlayer
	matchPlayerByUserId map[int64]*MatchPlayer

	battleMu         sync.Mutex
	battleById       map[int64]*model.BattleModel
	battleIdByUserId map[int64]int64

	//todo battleId要全局唯一
	battleIdSeq atomic.Int64
	tickCancel  context.CancelFunc
	tickWg      sync.WaitGroup
}

func NewBattleModule(d *Deps, dsModule *DsModule) *BattleModule {
	m := &BattleModule{
		deps:                d,
		dsModule:            dsModule,
		matchPools:          make(map[matchPoolKey]map[int64]*MatchPlayer),
		matchPlayerByUserId: make(map[int64]*MatchPlayer),
		battleById:          make(map[int64]*model.BattleModel),
		battleIdByUserId:    make(map[int64]int64),
	}
	m.battleIdSeq.Store(time.Now().UnixMilli())
	d.SessionMgr.OnUnbind(m.onDisconnect)
	dsModule.OnUnregister(m.onDsUnregister)
	return m
}

func (m *BattleModule) onDisconnect(s *btsession.Session) {
	m.matchMu.Lock()
	defer m.matchMu.Unlock()
	p := m.matchPlayerByUserId[s.UserId()]
	if p == nil || p.conn != s.Conn() {
		return
	}
	m.removeFromPoolLocked(p)
	btlog.Ins().Info("matchRemoveOnDisconnect", "userId", p.userId, "connId", p.conn.GetID())
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

func (m *BattleModule) RunMatch() {
	ctx, cancel := context.WithCancel(context.Background())
	m.tickCancel = cancel
	m.tickWg.Add(1)
	go func() {
		defer m.tickWg.Done()
		ticker := time.NewTicker(matchTickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.safeTick()
			}
		}
	}()
}

func (m *BattleModule) Stop() {
	if m.tickCancel != nil {
		m.tickCancel()
	}
	m.tickWg.Wait()
}

func (m *BattleModule) nextBattleID() int64 {
	return m.battleIdSeq.Add(1)
}

func (m *BattleModule) StartMatch(ctx *btrouter.RequestContext, req *proto.MatchStartReq) (*proto.MatchStartResp, error) {
	if _, ok := modePlayerCount[req.GameMode]; !ok || req.MapID <= 0 {
		return nil, proto.CodeMatchInvalidParam
	}
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, proto.CodeNotLogin
	}
	if m.getUserBattle(s.UserId()) != nil {
		return nil, proto.CodeBattleAlreadyIn
	}
	p := &MatchPlayer{
		userId:     s.UserId(),
		matchScore: defaultMatchScore,
		poolKey:    matchPoolKey{gameMode: req.GameMode, mapID: req.MapID},
		joinTime:   time.Now(),
		conn:       ctx.Conn(),
	}

	m.matchMu.Lock()
	if old := m.matchPlayerByUserId[p.userId]; old != nil {
		// 已在匹配中
		if !old.conn.IsClosed() {
			m.matchMu.Unlock()
			return nil, proto.CodeMatchAlreadyIn
		}
		// 旧连接已关闭但断线回调还没执行
		m.removeFromPoolLocked(old)
	}
	m.addToPoolLocked(p)
	m.matchMu.Unlock()

	btlog.Ins().Info("matchStart", "userId", p.userId, "gameMode", req.GameMode, "mapId", req.MapID, "matchScore", p.matchScore)
	return &proto.MatchStartResp{GameMode: req.GameMode, MapID: req.MapID}, nil
}

func (m *BattleModule) CancelMatch(ctx *btrouter.RequestContext, _ *proto.MatchCancelReq) (*proto.MatchCancelResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, proto.CodeNotLogin
	}

	m.matchMu.Lock()
	defer m.matchMu.Unlock()
	p := m.matchPlayerByUserId[s.UserId()]
	if p == nil || p.conn != ctx.Conn() {
		return nil, proto.CodeMatchNotIn
	}
	m.removeFromPoolLocked(p)
	btlog.Ins().Info("matchCancel", "userId", p.userId)
	return &proto.MatchCancelResp{}, nil
}

func (m *BattleModule) safeTick() {
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Error("matchTickPanic", "panic", r, "stack", string(debug.Stack()))
		}
	}()
	m.matchTick()
	m.checkBattleTimeout()
}

// 按入池先后为每人挑分数最近的人，凑够开局；等待超时的有多少人就带多少人开局
func (m *BattleModule) matchTick() {
	now := time.Now()

	m.matchMu.Lock()
	var matchedGroups []*matchGroup
	for poolKey, pool := range m.matchPools {
		playerCount := modePlayerCount[poolKey.gameMode]
		// 先入池的先匹配，以他为基准挑分数最近的人
		sortedPlayers := slices.SortedFunc(maps.Values(pool), func(a, b *MatchPlayer) int { return a.joinTime.Compare(b.joinTime) })
		for _, p := range sortedPlayers {
			// 可能已被前面的人挑走
			if m.matchPlayerByUserId[p.userId] != p {
				continue
			}
			timeout := now.Sub(p.joinTime) >= matchTimeout
			opponents := m.pickOpponentsLocked(p, playerCount-1, timeout)
			if opponents == nil {
				continue
			}
			m.removeFromPoolLocked(p)
			matchedGroups = append(matchedGroups, &matchGroup{poolKey: poolKey, players: append([]*MatchPlayer{p}, opponents...)})
		}
	}
	m.matchMu.Unlock()

	for _, group := range matchedGroups {
		m.startBattle(group)
	}
}

func (m *BattleModule) pickOpponentsLocked(p *MatchPlayer, need int, force bool) []*MatchPlayer {
	candidates := make([]*MatchPlayer, 0, len(m.matchPools[p.poolKey]))
	for _, c := range m.matchPools[p.poolKey] {
		if c.userId != p.userId && !c.conn.IsClosed() {
			candidates = append(candidates, c)
		}
	}
	//人数不足（不超时的情况）直接返回nil
	if len(candidates) < need && !force {
		return nil
	}
	//做个简单的按分差升序排序
	slices.SortFunc(candidates, func(a *MatchPlayer, b *MatchPlayer) int {
		return cmp.Compare(abs(a.matchScore-p.matchScore), abs(b.matchScore-p.matchScore))
	})
	picked := candidates[:min(need, len(candidates))]
	for _, c := range picked {
		m.removeFromPoolLocked(c)
	}
	return picked
}

func (m *BattleModule) addToPoolLocked(p *MatchPlayer) {
	pool := m.matchPools[p.poolKey]
	if pool == nil {
		pool = make(map[int64]*MatchPlayer)
		m.matchPools[p.poolKey] = pool
	}
	pool[p.userId] = p
	m.matchPlayerByUserId[p.userId] = p
}

func (m *BattleModule) removeFromPoolLocked(p *MatchPlayer) {
	if pool := m.matchPools[p.poolKey]; pool[p.userId] == p {
		delete(pool, p.userId)
		if len(pool) == 0 {
			delete(m.matchPools, p.poolKey)
		}
	}
	if m.matchPlayerByUserId[p.userId] == p {
		delete(m.matchPlayerByUserId, p.userId)
	}
}

func (m *BattleModule) startBattle(group *matchGroup) {
	playerIds := make([]int64, 0, len(group.players))
	for _, p := range group.players {
		playerIds = append(playerIds, p.userId)
	}

	ds := m.dsModule.PickDs()
	if ds == nil {
		btlog.Ins().Warn("matchNoAvailableDs", "gameMode", group.poolKey.gameMode, "mapId", group.poolKey.mapID, "playerIds", playerIds)
		m.pushMatchFail(group, proto.CodeMatchNoDs)
		return
	}

	battle := m.createBattle(group, ds)
	players := toPlayerInfos(battle.Players)
	log := btlog.Ins().With("battleId", battle.ID, "dsId", ds.ID)

	// 先通知ds，ds收不到这局就开不了
	assign := &proto.DsBattleAssignPush{BattleId: battle.ID, GameMode: battle.GameMode, MapID: battle.MapID, Players: players}
	if err := m.deps.DsRouterHandler.Push(ds.Conn, proto.PushDsBattleAssign, assign); err != nil {
		log.Warn("pushDsBattleAssignError", "err", err)
		// 玩家还没收到匹配成功，只推匹配失败，不推战斗结束
		m.battleMu.Lock()
		m.removeBattleLocked(battle)
		m.battleMu.Unlock()
		m.dsModule.ReleaseDs(ds)
		m.pushMatchFail(group, proto.CodeMatchNoDs)
		return
	}

	push := &proto.MatchSuccessPush{
		BattleId: battle.ID,
		GameMode: battle.GameMode,
		MapID:    battle.MapID,
		DsIP:     ds.IP,
		DsPort:   ds.Port,
		Players:  players,
	}
	// 第一个玩家是最早入池的基准玩家
	log.Info("matchSuccess", "gameMode", group.poolKey.gameMode, "mapId", group.poolKey.mapID,
		"dsAddr", net.JoinHostPort(ds.IP, strconv.Itoa(ds.Port)), "playerIds", playerIds,
		"waitMs", time.Since(group.players[0].joinTime).Milliseconds())
	for _, p := range group.players {
		if err := m.deps.GameRouterHandler.Push(p.conn, proto.PushMatchSuccess, push); err != nil {
			log.Warn("pushMatchSuccessError", "userId", p.userId, "err", err)
		}
	}
}

// 通知客户端重新发起匹配
func (m *BattleModule) pushMatchFail(group *matchGroup, code btrouter.Code) {
	push := &proto.MatchFailPush{Code: uint32(code)}
	for _, p := range group.players {
		if err := m.deps.GameRouterHandler.Push(p.conn, proto.PushMatchFail, push); err != nil {
			btlog.Ins().Warn("pushMatchFailError", "userId", p.userId, "err", err)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ==================== 战斗状态 ====================

const (
	battleStartTimeout = 60 * time.Second
	battleMaxDuration  = 120 * time.Minute
)

func (m *BattleModule) createBattle(group *matchGroup, ds *model.DsModel) *model.BattleModel {
	battle := &model.BattleModel{
		ID:         m.nextBattleID(),
		GameMode:   group.poolKey.gameMode,
		MapID:      group.poolKey.mapID,
		Ds:         ds,
		State:      model.BattleStateWaiting,
		CreateTime: time.Now().UnixMilli(),
	}
	for i, p := range group.players {
		battle.Players = append(battle.Players, &model.BattlePlayer{UserId: p.userId, MatchScore: p.matchScore, Team: i % 2})
	}

	m.battleMu.Lock()
	m.battleById[battle.ID] = battle
	for _, p := range battle.Players {
		m.battleIdByUserId[p.UserId] = battle.ID
	}
	m.battleMu.Unlock()
	return battle
}

func (m *BattleModule) removeBattleLocked(battle *model.BattleModel) bool {
	if m.battleById[battle.ID] != battle {
		return false
	}
	delete(m.battleById, battle.ID)
	for _, p := range battle.Players {
		if m.battleIdByUserId[p.UserId] == battle.ID {
			delete(m.battleIdByUserId, p.UserId)
		}
	}
	return true
}

func (m *BattleModule) getUserBattle(userId int64) *model.BattleModel {
	m.battleMu.Lock()
	defer m.battleMu.Unlock()
	return m.battleById[m.battleIdByUserId[userId]]
}

func (m *BattleModule) getDsOwnedBattleLocked(ctx *btrouter.RequestContext, battleId int64) (*model.BattleModel, error) {
	ds, ok := GetDs(ctx.Conn())
	if !ok {
		return nil, proto.CodeDsNotRegistered
	}
	battle := m.battleById[battleId]
	if battle == nil || battle.Ds != ds {
		return nil, proto.CodeBattleNotFound
	}
	return battle, nil
}

func (m *BattleModule) DsBattleStart(ctx *btrouter.RequestContext, req *proto.DsBattleStartReq) (*proto.DsBattleStartResp, error) {
	m.battleMu.Lock()
	battle, err := m.getDsOwnedBattleLocked(ctx, req.BattleId)
	if err != nil {
		m.battleMu.Unlock()
		return nil, err
	}
	if battle.State != model.BattleStateWaiting {
		m.battleMu.Unlock()
		return nil, proto.CodeBattleStateError
	}
	battle.State = model.BattleStateRunning
	battle.StartTime = time.Now().UnixMilli()
	m.battleMu.Unlock()

	btlog.Ins().Info("battleStart", "battleId", battle.ID, "dsId", battle.Ds.ID,
		"waitMs", battle.StartTime-battle.CreateTime)
	return &proto.DsBattleStartResp{BattleId: battle.ID}, nil
}

func (m *BattleModule) DsBattleEnd(ctx *btrouter.RequestContext, req *proto.DsBattleEndReq) (*proto.DsBattleEndResp, error) {
	m.battleMu.Lock()
	battle, err := m.getDsOwnedBattleLocked(ctx, req.BattleId)
	if err != nil {
		m.battleMu.Unlock()
		return nil, err
	}
	if battle.State != model.BattleStateRunning {
		m.battleMu.Unlock()
		return nil, proto.CodeBattleStateError
	}
	m.removeBattleLocked(battle)
	m.battleMu.Unlock()

	m.dsModule.ReleaseDs(battle.Ds)
	// todo 结算落库
	btlog.Ins().Info("battleEnd", "battleId", battle.ID, "dsId", battle.Ds.ID, "gameMode", battle.GameMode,
		"durationMs", time.Now().UnixMilli()-battle.StartTime, "statistics", req.Statistics)
	m.pushBattleEnd(battle, proto.BattleEndNormal)
	return &proto.DsBattleEndResp{BattleId: battle.ID}, nil
}

func (m *BattleModule) QueryBattle(ctx *btrouter.RequestContext, _ *proto.BattleQueryReq) (*proto.BattleQueryResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, proto.CodeNotLogin
	}
	m.battleMu.Lock()
	defer m.battleMu.Unlock()
	battle := m.battleById[m.battleIdByUserId[s.UserId()]]
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

func (m *BattleModule) abortBattle(battle *model.BattleModel, reason string) {
	m.battleMu.Lock()
	removed := m.removeBattleLocked(battle)
	state := battle.State
	m.battleMu.Unlock()
	if !removed {
		return
	}
	m.dsModule.ReleaseDs(battle.Ds)
	btlog.Ins().Warn("battleAbort", "battleId", battle.ID, "dsId", battle.Ds.ID, "reason", reason, "state", state)
	m.pushBattleEnd(battle, proto.BattleEndAbort)
}

// todo 先定义在这里
type timeoutBattle struct {
	battle *model.BattleModel
	reason string
}

func (m *BattleModule) checkBattleTimeout() {
	now := time.Now().UnixMilli()

	var timeoutBattles []timeoutBattle

	m.battleMu.Lock()
	for _, battle := range m.battleById {
		switch {
		case battle.State == model.BattleStateWaiting && now-battle.CreateTime >= battleStartTimeout.Milliseconds():
			timeoutBattles = append(timeoutBattles, timeoutBattle{battle, "startTimeout"})
		case battle.State == model.BattleStateRunning && now-battle.StartTime >= battleMaxDuration.Milliseconds():
			timeoutBattles = append(timeoutBattles, timeoutBattle{battle, "durationExceeded"})
		}
	}
	m.battleMu.Unlock()

	for _, t := range timeoutBattles {
		m.abortBattle(t.battle, t.reason)
	}
}

func (m *BattleModule) onDsUnregister(ds *model.DsModel) {
	var dsBattles []*model.BattleModel
	m.battleMu.Lock()
	for _, battle := range m.battleById {
		if battle.Ds == ds {
			dsBattles = append(dsBattles, battle)
		}
	}
	m.battleMu.Unlock()

	for _, battle := range dsBattles {
		m.abortBattle(battle, "dsDisconnect")
	}
}

func (m *BattleModule) pushBattleEnd(battle *model.BattleModel, reason int) {
	push := &proto.BattleEndPush{BattleId: battle.ID, Reason: reason}
	for _, p := range battle.Players {
		s, ok := m.deps.SessionMgr.Get(p.UserId)
		if !ok {
			continue
		}
		if err := m.deps.GameRouterHandler.Push(s.Conn(), proto.PushBattleEnd, push); err != nil {
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
