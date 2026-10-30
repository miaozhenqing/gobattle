package service

import (
	"cmp"
	"errors"
	"gobattle/btlog"
	"gobattle/btsession"
	"gobattle/game/model"
	"gobattle/game/protocol"
	"gobattle/game/tpl"
	"maps"
	"net"
	"slices"
	"strconv"
	"sync"
	"time"
)

const (
	defaultMatchScore = 1000
	matchTickInterval = time.Second
)

type matchPoolKey struct {
	gameMode int
	mapId    int
}

type matchPlayer struct {
	userId     int64
	matchScore int
	poolKey    matchPoolKey
	joinTime   time.Time
	session    *btsession.Session
}

func (p *matchPlayer) offline() bool {
	return p.session.Conn().IsClosed()
}

type matchGroup struct {
	poolKey matchPoolKey
	players []*matchPlayer
}

type MatchService struct {
	battleService *BattleService
	pushService   *PushService
	tplMgr        *tpl.TplManager

	mu                  sync.Mutex
	matchPools          map[matchPoolKey]map[int64]*matchPlayer
	matchPlayerByUserId map[int64]*matchPlayer

	tick *tickLoop
}

func NewMatchService(battleService *BattleService, pushService *PushService, sessionMgr *btsession.Manager, tplMgr *tpl.TplManager) *MatchService {
	s := &MatchService{
		battleService:       battleService,
		pushService:         pushService,
		tplMgr:              tplMgr,
		matchPools:          make(map[matchPoolKey]map[int64]*matchPlayer),
		matchPlayerByUserId: make(map[int64]*matchPlayer),
	}
	s.tick = newTickLoop("match", matchTickInterval, s.matchTick)
	sessionMgr.OnUnbind(s.onDisconnect)
	return s
}

func (s *MatchService) Run() {
	s.tick.run()
}

func (s *MatchService) Stop() {
	s.tick.stop()
}

func (s *MatchService) Join(session *btsession.Session, gameMode, mapId int) error {
	if modeTpl, ok := s.tplMgr.GetMatchModeTpl(gameMode); !ok || !modeTpl.HasMap(mapId) {
		return protocol.CodeMatchInvalidParam
	}
	if s.battleService.GetUserBattle(session.UserId()) != nil {
		return protocol.CodeBattleAlreadyIn
	}
	p := &matchPlayer{
		userId:     session.UserId(),
		matchScore: defaultMatchScore,
		poolKey:    matchPoolKey{gameMode: gameMode, mapId: mapId},
		joinTime:   time.Now(),
		session:    session,
	}

	s.mu.Lock()
	if old := s.matchPlayerByUserId[p.userId]; old != nil {
		// 已在匹配中
		if !old.offline() {
			s.mu.Unlock()
			return protocol.CodeMatchAlreadyIn
		}
		// 旧连接已关闭但断线回调还没执行
		s.removeFromPoolLocked(old)
	}
	s.addToPoolLocked(p)
	s.mu.Unlock()

	btlog.Ins().Info("matchStart", "userId", p.userId, "gameMode", gameMode, "mapId", mapId, "matchScore", p.matchScore)
	return nil
}

func (s *MatchService) Cancel(session *btsession.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.matchPlayerByUserId[session.UserId()]
	if p == nil || p.session != session {
		return protocol.CodeMatchNotIn
	}
	s.removeFromPoolLocked(p)
	btlog.Ins().Info("matchCancel", "userId", p.userId)
	return nil
}

func (s *MatchService) onDisconnect(session *btsession.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.matchPlayerByUserId[session.UserId()]
	if p == nil || p.session != session {
		return
	}
	s.removeFromPoolLocked(p)
	btlog.Ins().Info("matchRemoveOnDisconnect", "userId", p.userId, "connId", session.Conn().GetId())
}

// 按入池先后为每人挑分数最近的人，凑够开局；等待超时的有多少人就带多少人开局
func (s *MatchService) matchTick() {
	now := time.Now()

	s.mu.Lock()
	var matchedGroups []*matchGroup
	for poolKey, pool := range s.matchPools {
		modeTpl, _ := s.tplMgr.GetMatchModeTpl(poolKey.gameMode)
		// 先入池的先匹配，以他为基准挑分数最近的人
		sortedPlayers := slices.SortedFunc(maps.Values(pool), func(a, b *matchPlayer) int { return a.joinTime.Compare(b.joinTime) })
		for _, p := range sortedPlayers {
			// 可能已被前面的人挑走
			if s.matchPlayerByUserId[p.userId] != p {
				continue
			}
			timeout := now.Sub(p.joinTime) >= modeTpl.MatchTimeout()
			opponents := s.pickOpponentsLocked(p, modeTpl.PlayerCount-1, timeout)
			if opponents == nil {
				continue
			}
			s.removeFromPoolLocked(p)
			matchedGroups = append(matchedGroups, &matchGroup{poolKey: poolKey, players: append([]*matchPlayer{p}, opponents...)})
		}
	}
	s.mu.Unlock()

	for _, group := range matchedGroups {
		s.startBattle(group)
	}
}

func (s *MatchService) pickOpponentsLocked(p *matchPlayer, need int, force bool) []*matchPlayer {
	candidates := make([]*matchPlayer, 0, len(s.matchPools[p.poolKey]))
	for _, c := range s.matchPools[p.poolKey] {
		if c.userId != p.userId && !c.offline() {
			candidates = append(candidates, c)
		}
	}
	//人数不足（不超时的情况）直接返回nil
	if len(candidates) < need && !force {
		return nil
	}
	//做个简单的按分差升序排序
	slices.SortFunc(candidates, func(a *matchPlayer, b *matchPlayer) int {
		return cmp.Compare(abs(a.matchScore-p.matchScore), abs(b.matchScore-p.matchScore))
	})
	picked := candidates[:min(need, len(candidates))]
	for _, c := range picked {
		s.removeFromPoolLocked(c)
	}
	return picked
}

func (s *MatchService) addToPoolLocked(p *matchPlayer) {
	pool := s.matchPools[p.poolKey]
	if pool == nil {
		pool = make(map[int64]*matchPlayer)
		s.matchPools[p.poolKey] = pool
	}
	pool[p.userId] = p
	s.matchPlayerByUserId[p.userId] = p
}

func (s *MatchService) removeFromPoolLocked(p *matchPlayer) {
	if pool := s.matchPools[p.poolKey]; pool[p.userId] == p {
		delete(pool, p.userId)
		if len(pool) == 0 {
			delete(s.matchPools, p.poolKey)
		}
	}
	if s.matchPlayerByUserId[p.userId] == p {
		delete(s.matchPlayerByUserId, p.userId)
	}
}

func (s *MatchService) startBattle(group *matchGroup) {
	players := make([]*model.BattlePlayer, 0, len(group.players))
	for i, p := range group.players {
		players = append(players, &model.BattlePlayer{UserId: p.userId, MatchScore: p.matchScore, Team: i % 2})
	}

	battle, err := s.battleService.Create(group.poolKey.gameMode, group.poolKey.mapId, players)
	if err != nil {
		s.pushMatchFail(group, err)
		return
	}

	ds := battle.Ds
	push := &protocol.MatchSuccessPush{
		BattleId: battle.Id,
		GameMode: battle.GameMode,
		MapId:    battle.MapId,
		DsIP:     ds.IP,
		DsPort:   ds.Port,
		Players:  toPlayerInfos(battle.Players),
	}
	log := btlog.Ins().With("battleId", battle.Id, "dsId", ds.Id)
	// 第一个玩家是最早入池的基准玩家
	log.Info("matchSuccess", "gameMode", battle.GameMode, "mapId", battle.MapId,
		"dsAddr", net.JoinHostPort(ds.IP, strconv.Itoa(ds.Port)), "playerIds", playerIdsOf(battle.Players),
		"waitMs", time.Since(group.players[0].joinTime).Milliseconds())
	for _, p := range group.players {
		if err := s.pushService.PushUser(p.userId, protocol.PushMatchSuccess, push); err != nil {
			log.Warn("pushMatchSuccessError", "userId", p.userId, "err", err)
		}
	}
}

// 通知客户端重新发起匹配
func (s *MatchService) pushMatchFail(group *matchGroup, err error) {
	code := protocol.CodeMatchNoDs
	errors.As(err, &code)
	push := &protocol.MatchFailPush{Code: uint32(code)}
	for _, p := range group.players {
		if err := s.pushService.PushUser(p.userId, protocol.PushMatchFail, push); err != nil {
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
