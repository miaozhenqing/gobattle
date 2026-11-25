package service

import (
	"gobattle/common/protocol"
	"gobattle/framework/btlog"
	"gobattle/framework/btsession"
)

type KickService struct {
	pushService *PushService
}

func NewKickService(push *PushService) *KickService {
	return &KickService{pushService: push}
}
func (k *KickService) KickSession(s *btsession.Session) {
	if s == nil || s.Conn().IsClosed() {
		return
	}
	btlog.Ins().Info("kickService", "userId", s.UserId(), "sessionId", s.SessionId(), "connId", s.Conn().GetId())
	_ = k.pushService.PushSession(s, protocol.PushKick, &protocol.KickPush{})
	s.Conn().FlushAndStop()
}
