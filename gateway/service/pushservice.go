package service

import (
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"gobattle/framework/btrpc"
	"gobattle/framework/btsession"
)

type PushService struct {
	router     *btrouter.RouterHandler
	sessionMgr *btsession.Manager
}

func NewPushService(router *btrouter.RouterHandler, sessionMgr *btsession.Manager) *PushService {
	return &PushService{router: router, sessionMgr: sessionMgr}
}

func (s *PushService) OnBackendPush(p *btrpc.Packet) {
	msgId := p.MsgId
	if p.UserId == 0 {
		s.sessionMgr.Range(func(sess *btsession.Session) {
			_ = s.router.PushRaw(sess.Conn(), msgId, p.Payload)
		})
		return
	}
	sess, ok := s.sessionMgr.Get(p.UserId)
	if !ok {
		btlog.Ins().Debug("pushUserOffline", "userId", p.UserId, "msgId", msgId)
		return
	}
	_ = s.router.PushRaw(sess.Conn(), msgId, p.Payload)
}

func (s *PushService) PushSession(sess *btsession.Session, msgId uint32, v any) error {
	return s.router.Push(sess.Conn(), msgId, v)
}
