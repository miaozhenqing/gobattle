package module

import (
	"cmp"
	"context"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/proto"
	"maps"
	"slices"
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

type poolKey struct {
	gameMode int
	mapID    int
}

type MatchPlayer struct {
	userId     int64
	matchScore int
	key        poolKey
	joinTime   time.Time
	conn       *btnet.Connection
}

type matchGroup struct {
	key     poolKey
	players []*MatchPlayer
}

type MatchModule struct {
	*Deps

	mu      sync.Mutex
	pools   map[poolKey]map[int64]*MatchPlayer
	inMatch map[int64]*MatchPlayer

	//todo battleId要全局唯一
	seq    atomic.Int64
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewMatchModule(d *Deps) *MatchModule {
	m := &MatchModule{
		Deps:    d,
		pools:   make(map[poolKey]map[int64]*MatchPlayer),
		inMatch: make(map[int64]*MatchPlayer),
	}
	m.seq.Store(time.Now().UnixMilli())
	d.SessionMgr.OnUnbind(m.onDisconnect)
	return m
}

func (m *MatchModule) onDisconnect(s *btsession.Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.inMatch[s.UID()]
	if p == nil || p.conn != s.Conn() {
		return
	}
	m.removeLocked(p)
	btlog.Ins().Infof("match remove on disconnect, uid:%v connID:%v", p.userId, p.conn.GetID())
}

func (m *MatchModule) RegisterRoutes(r *btrouter.Router) {
	btrouter.Handle(r, proto.MsgMatchStart, m.StartMatch)
	btrouter.Handle(r, proto.MsgMatchCancel, m.CancelMatch)
}

func (m *MatchModule) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
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

func (m *MatchModule) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
}

func (m *MatchModule) StartMatch(ctx *btrouter.RequestContext, req *proto.MatchStartReq) (*proto.MatchStartResp, error) {
	if _, ok := modePlayerCount[req.GameMode]; !ok || req.MapID <= 0 {
		return nil, proto.CodeMatchInvalidParam
	}
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, proto.CodeNotLogin
	}
	p := &MatchPlayer{
		userId:     s.UID(),
		matchScore: defaultMatchScore,
		key:        poolKey{gameMode: req.GameMode, mapID: req.MapID},
		joinTime:   time.Now(),
		conn:       ctx.Conn(),
	}

	m.mu.Lock()
	if old := m.inMatch[p.userId]; old != nil {
		// 已在匹配中
		if !old.conn.IsClosed() {
			m.mu.Unlock()
			return nil, proto.CodeMatchAlreadyIn
		}
		// 旧连接已关闭但断线回调还没执行
		m.removeLocked(old)
	}
	m.addLocked(p)
	m.mu.Unlock()

	btlog.Ins().Infof("match start, uid:%v gameMode:%v mapID:%v matchScore:%v", p.userId, req.GameMode, req.MapID, p.matchScore)
	return &proto.MatchStartResp{BattleId: m.seq.Load(), GameMode: req.GameMode, MapID: req.MapID}, nil
}

func (m *MatchModule) CancelMatch(ctx *btrouter.RequestContext, _ *proto.MatchCancelReq) (*proto.MatchCancelResp, error) {
	s, ok := btsession.GetSession(ctx.Conn())
	if !ok {
		return nil, proto.CodeNotLogin
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.inMatch[s.UID()]
	if p == nil || p.conn != ctx.Conn() {
		return nil, proto.CodeMatchNotIn
	}
	m.removeLocked(p)
	btlog.Ins().Infof("match cancel, uid:%v", p.userId)
	return &proto.MatchCancelResp{}, nil
}

func (m *MatchModule) safeTick() {
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Errorf("match tick panic:%v", r)
		}
	}()
	m.tick()
}

// 按入池先后为每人挑分数最近的人，凑够开局；等待超时的有多少人就带多少人开局
func (m *MatchModule) tick() {
	now := time.Now()

	m.mu.Lock()
	var groups []*matchGroup
	for key, pool := range m.pools {
		n := modePlayerCount[key.gameMode]
		// 先入池的先匹配，以他为基准挑分数最近的人
		waiting := slices.SortedFunc(maps.Values(pool), func(a, b *MatchPlayer) int { return a.joinTime.Compare(b.joinTime) })
		for _, p := range waiting {
			// 可能已被前面的人挑走
			if m.inMatch[p.userId] != p {
				continue
			}
			timeout := now.Sub(p.joinTime) >= matchTimeout
			others := m.pickLocked(p, n-1, timeout)
			if others == nil {
				continue
			}
			m.removeLocked(p)
			groups = append(groups, &matchGroup{key: key, players: append([]*MatchPlayer{p}, others...)})
		}
	}
	m.mu.Unlock()

	for _, g := range groups {
		m.pushSuccess(g)
	}
}

func (m *MatchModule) pickLocked(p *MatchPlayer, need int, force bool) []*MatchPlayer {
	candidates := make([]*MatchPlayer, 0, len(m.pools[p.key]))
	for _, c := range m.pools[p.key] {
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
		m.removeLocked(c)
	}
	return picked
}

func (m *MatchModule) addLocked(p *MatchPlayer) {
	pool := m.pools[p.key]
	if pool == nil {
		pool = make(map[int64]*MatchPlayer)
		m.pools[p.key] = pool
	}
	pool[p.userId] = p
	m.inMatch[p.userId] = p
}

func (m *MatchModule) removeLocked(p *MatchPlayer) {
	if pool := m.pools[p.key]; pool[p.userId] == p {
		delete(pool, p.userId)
		if len(pool) == 0 {
			delete(m.pools, p.key)
		}
	}
	if m.inMatch[p.userId] == p {
		delete(m.inMatch, p.userId)
	}
}

func (m *MatchModule) pushSuccess(g *matchGroup) {
	ds := m.DsModule.PickDs()
	if ds == nil {
		btlog.Ins().Warnf("match no available ds, gameMode:%v mapID:%v players:%v", g.key.gameMode, g.key.mapID, len(g.players))
		// 通知客户端重新发起匹配
		push := &proto.MatchFailPush{Code: uint32(proto.CodeMatchNoDs)}
		for _, p := range g.players {
			if err := m.Router.Push(p.conn, proto.PushMatchFail, push); err != nil {
				btlog.Ins().Warnf("push match fail error, uid:%v err:%v", p.userId, err)
			}
		}
		return
	}

	push := &proto.MatchSuccessPush{
		BattleId: m.seq.Add(1),
		GameMode: g.key.gameMode,
		MapID:    g.key.mapID,
		DsIP:     ds.IP,
		DsPort:   ds.Port,
	}
	for i, p := range g.players {
		push.Players = append(push.Players, &proto.MatchPlayerInfo{UID: p.userId, MatchScore: p.matchScore, Team: i % 2})
	}

	btlog.Ins().Infof("match success, BattleId:%v gameMode:%v mapID:%v ds:%v:%v players:%v",
		push.BattleId, g.key.gameMode, g.key.mapID, ds.IP, ds.Port, len(g.players))
	for _, p := range g.players {
		if err := m.Router.Push(p.conn, proto.PushMatchSuccess, push); err != nil {
			btlog.Ins().Warnf("push match success error, BattleId:%v uid:%v err:%v", push.BattleId, p.userId, err)
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
