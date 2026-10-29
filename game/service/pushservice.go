package service

import (
	"errors"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/model"
)

var errUserOffline = errors.New("user offline")

// 统一推送出口
type PushService struct {
	gameRouterHandler *btrouter.RouterHandler
	dsRouterHandler   *btrouter.RouterHandler
	sessionMgr        *btsession.Manager
}

func NewPushService(gameRouterHandler, dsRouterHandler *btrouter.RouterHandler, sessionMgr *btsession.Manager) *PushService {
	return &PushService{
		gameRouterHandler: gameRouterHandler,
		dsRouterHandler:   dsRouterHandler,
		sessionMgr:        sessionMgr,
	}
}

func (s *PushService) PushUser(userId int64, msgId uint32, v any) error {
	session, ok := s.sessionMgr.Get(userId)
	if !ok {
		return errUserOffline
	}
	return s.PushSession(session, msgId, v)
}

func (s *PushService) PushSession(session *btsession.Session, msgId uint32, v any) error {
	return s.gameRouterHandler.Push(session.Conn(), msgId, v)
}

func (s *PushService) PushDs(ds *model.DsModel, msgId uint32, v any) error {
	return s.dsRouterHandler.Push(ds.Conn, msgId, v)
}
