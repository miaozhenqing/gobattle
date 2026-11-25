package service

import (
	"context"
	"encoding/json"
	"gobattle/common/redis/sessionstore"
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"gobattle/framework/btsession"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisTimeout = 2 * time.Second
const onlineCheckBatchSize = 300

// 订阅下线消息、对账
type OnlineService struct {
	client       *redis.Client
	sessionStore *sessionstore.Store
	mgr          *btsession.Manager
	kickService  *KickService
	gatewayId    int64
	ttl          time.Duration
	interval     time.Duration
	cleanup      chan sessionstore.OnlineInfo
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewOnlineService(client *redis.Client, store *sessionstore.Store, mgr *btsession.Manager,
	kick *KickService, gatewayId int64, ttl, interval time.Duration) *OnlineService {
	if interval <= 0 || ttl <= 2*interval {
		panic("session ttl must exceed twice reconcile interval")
	}
	return &OnlineService{client: client, sessionStore: store, mgr: mgr, kickService: kick,
		gatewayId: gatewayId, ttl: ttl, interval: interval, cleanup: make(chan sessionstore.OnlineInfo, 2048)}
}

func (p *OnlineService) sessionRecord(s *btsession.Session) sessionstore.OnlineInfo {
	return sessionstore.OnlineInfo{UserId: s.UserId(), GatewayId: p.gatewayId,
		SessionId: s.SessionId(), Version: s.Version()}
}

func (p *OnlineService) IsActiveSession(ctx *btrouter.RequestContext) bool {
	s, ok := btsession.GetSession(ctx.Conn())
	return ok && p.mgr.IsCurrentSession(s) && s.OnlineCheckedWithin(p.ttl-p.interval)
}

func (p *OnlineService) SetOnline(ctx context.Context, userId int64) (sessionstore.OnlineInfo, error) {
	id, err := sessionstore.NewSessionId()
	if err != nil {
		return sessionstore.OnlineInfo{}, err
	}
	setCtx, cancel := context.WithTimeout(ctx, redisTimeout)
	online, err := p.sessionStore.SetOnline(setCtx, userId, p.gatewayId, id)
	cancel()
	if err == nil {
		kickCtx, done := context.WithTimeout(context.Background(), redisTimeout)
		//有旧链接通知踢下去
		kickErr := p.sessionStore.PublishKickNotice(kickCtx, online.OldSession)
		done()
		if kickErr != nil {
			btlog.Ins().Warn("kickNotifyFail", "userId", userId, "oldGatewayId", online.OldSession.GatewayId, "err", kickErr)
		}
		return online.NewSession, nil
	}
	readCtx, done := context.WithTimeout(context.Background(), redisTimeout)
	current, exists, readErr := p.sessionStore.GetOnline(readCtx, userId)
	done()
	if readErr == nil && exists && current.GatewayId == p.gatewayId && current.SessionId == id {
		return current, nil
	}
	return sessionstore.OnlineInfo{}, err
}

func (p *OnlineService) GetOnline(ctx context.Context, userId int64) (sessionstore.OnlineInfo, bool, error) {
	readCtx, cancel := context.WithTimeout(ctx, redisTimeout)
	defer cancel()
	return p.sessionStore.GetOnline(readCtx, userId)
}

func (p *OnlineService) RemoveOnline(r sessionstore.OnlineInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()
	if err := p.sessionStore.DeleteOnlineIfMatch(ctx, r); err != nil {
		btlog.Ins().Warn("sessionReleaseFail", "userId", r.UserId, "sessionId", r.SessionId, "err", err)
	}
}

func (p *OnlineService) OnUnbind(s *btsession.Session) {
	select {
	case p.cleanup <- p.sessionRecord(s):
	default:
		btlog.Ins().Warn("sessionCleanupQueueFull", "userId", s.UserId())
	}
}

func (p *OnlineService) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	sub := p.client.Subscribe(ctx, sessionstore.KickChannel(p.gatewayId))
	readyCtx, done := context.WithTimeout(ctx, redisTimeout)
	_, err := sub.Receive(readyCtx)
	done()
	if err != nil {
		_ = sub.Close()
		cancel()
		return err
	}
	p.cancel = cancel
	p.wg.Add(3)
	go p.handleKickNotices(ctx, sub)
	go p.checkOnlineSessionsPeriodically(ctx)
	go p.deleteDisconnectedSessions(ctx)
	return nil
}

func (p *OnlineService) handleKickNotices(ctx context.Context, sub *redis.PubSub) {
	defer p.wg.Done()
	defer func() { _ = sub.Close() }()
	for ctx.Err() == nil {
		msg, err := sub.ReceiveMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			btlog.Ins().Warn("kickSubscribeFail", "gatewayId", p.gatewayId, "err", err)
			_ = sub.Close()
			for ctx.Err() == nil {
				timer := time.NewTimer(time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				next := p.client.Subscribe(ctx, sessionstore.KickChannel(p.gatewayId))
				readyCtx, done := context.WithTimeout(ctx, redisTimeout)
				_, readyErr := next.Receive(readyCtx)
				done()
				if readyErr == nil {
					sub = next
					break
				}
				_ = next.Close()
			}
			continue
		}
		var notice sessionstore.KickNotice
		if err := json.Unmarshal([]byte(msg.Payload), &notice); err != nil {
			btlog.Ins().Warn("kickNoticeInvalid", "err", err)
			continue
		}
		userId, err1 := strconv.ParseInt(notice.UserId, 10, 64)
		version, err2 := strconv.ParseInt(notice.Version, 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		s, ok := p.mgr.Get(userId)
		if ok && s.SessionId() == notice.SessionId && s.Version() == version && p.mgr.IsCurrentSession(s) {
			p.kickService.KickSession(s)
		}
	}
}

func (p *OnlineService) checkOnlineSessionsPeriodically(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.checkAndRenewSessions(ctx)
		}
	}
}

func (p *OnlineService) checkAndRenewSessions(ctx context.Context) {
	var sessions []*btsession.Session
	p.mgr.Range(func(s *btsession.Session) { sessions = append(sessions, s) })
	for start := 0; start < len(sessions); start += onlineCheckBatchSize {
		end := start + onlineCheckBatchSize
		if end > len(sessions) {
			end = len(sessions)
		}
		group := sessions[start:end]
		records := make([]sessionstore.OnlineInfo, len(group))
		for i, s := range group {
			records[i] = p.sessionRecord(s)
		}
		callCtx, cancel := context.WithTimeout(ctx, redisTimeout)
		matched, err := p.sessionStore.RenewOnlineBatch(callCtx, records)
		cancel()
		if err != nil {
			btlog.Ins().Warn("sessionReconcileFail", "gatewayId", p.gatewayId, "err", err)
			for _, s := range group {
				if p.mgr.IsCurrentSession(s) && !s.OnlineCheckedWithin(p.ttl-p.interval) {
					p.kickService.KickSession(s)
				}
			}
			continue
		}
		for i, s := range group {
			if !p.mgr.IsCurrentSession(s) {
				continue
			}
			if matched[i] {
				s.MarkOnlineChecked()
			} else {
				p.kickService.KickSession(s)
			}
		}
	}
}

func (p *OnlineService) deleteDisconnectedSessions(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-p.cleanup:
			p.RemoveOnline(r)
		}
	}
}

func (p *OnlineService) Stop(ctx context.Context) {
	if p.cancel != nil {
		p.cancel()
	}
	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		return
	}
	for {
		select {
		case r := <-p.cleanup:
			if ctx.Err() != nil {
				return
			}
			p.RemoveOnline(r)
		default:
			return
		}
	}
}
