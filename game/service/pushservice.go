package service

import (
	"context"
	"encoding/json"
	"errors"
	"gobattle/common/redis/sessionstore"
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"time"
)

var (
	errUserOffline   = errors.New("user offline")
	errInvalidUserId = errors.New("invalid userId")
)

// 统一推送出口
type PushService struct {
	dsRouterHandler *btrouter.RouterHandler
	onlineReader    *sessionstore.OnlineReader
}

func NewPushService(dsRouterHandler *btrouter.RouterHandler, onlineReader *sessionstore.OnlineReader) *PushService {
	return &PushService{dsRouterHandler: dsRouterHandler, onlineReader: onlineReader}
}

func (s *PushService) PushUser(userId int64, msgId uint32, v any) error {
	return s.PushUsers([]int64{userId}, msgId, v)
}

func (s *PushService) PushUsers(userIds []int64, msgId uint32, v any) error {
	if len(userIds) == 0 {
		return nil
	}
	for _, userId := range userIds {
		if userId <= 0 {
			btlog.Ins().Error("pushInvalidUserId", "userId", userId, "msgId", msgId)
			return errInvalidUserId
		}
	}
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	targets := make([]sessionstore.PushTarget, 0, len(userIds))
	seen := make(map[int64]struct{}, len(userIds))
	for _, userId := range userIds {
		if _, ok := seen[userId]; ok {
			continue
		}
		seen[userId] = struct{}{}
		targets = append(targets, sessionstore.PushTarget{UserId: userId, MsgId: msgId, Payload: payload})
	}
	count, err := s.onlineReader.PublishPushBatch(ctx, targets)
	if err == nil && count == 0 {
		return errUserOffline
	}
	return err
}

func (s *PushService) PushDs(ds *Ds, msgId uint32, v any) error {
	return s.dsRouterHandler.Push(ds.Conn, msgId, v)
}
