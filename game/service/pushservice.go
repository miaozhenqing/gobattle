package service

import (
	"errors"
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"gobattle/framework/btrpc"
)

var (
	errUserOffline   = errors.New("user offline")
	errInvalidUserId = errors.New("invalid userId")
)

// 统一推送出口
type PushService struct {
	rpcServer       *btrpc.Server
	dsRouterHandler *btrouter.RouterHandler
}

func NewPushService(rpcServer *btrpc.Server, dsRouterHandler *btrouter.RouterHandler) *PushService {
	return &PushService{
		rpcServer:       rpcServer,
		dsRouterHandler: dsRouterHandler,
	}
}

// todo 当前方案：广播给所有 gateway，由 gateway 按 userId 过滤（UserId=0 在 gateway 侧表示全服广播，这里必须拦截）
func (s *PushService) PushUser(userId int64, msgId uint32, v any) error {
	if userId <= 0 {
		btlog.Ins().Error("pushInvalidUserId", "userId", userId, "msgId", msgId)
		return errInvalidUserId
	}
	return s.rpcServer.Broadcast("gateway", userId, msgId, v)
}

func (s *PushService) PushDs(ds *Ds, msgId uint32, v any) error {
	return s.dsRouterHandler.Push(ds.Conn, msgId, v)
}
