package service

import (
	"context"
	"gobattle/btlog"
	"gobattle/game/db"
	"gobattle/game/model"
	"gobattle/game/repository"
	"time"
)

type UserService struct {
	db *repository.DB
	// 单次业务请求内 DB 操作的总超时
	queryTimeout time.Duration
}

func NewUserService(db *repository.DB, queryTimeout time.Duration) *UserService {
	return &UserService{db: db, queryTimeout: queryTimeout}
}

type LoginResult struct {
	User    *model.UserModel
	Created bool
}

func (s *UserService) Login(ctx context.Context, username string) (*LoginResult, error) {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()
	res, err := s.login(ctx, username)
	if db.IsDupEntry(err) || db.IsDeadlock(err) {
		btlog.Ins().Warn("loginConflictRetry", "username", username, "err", err)
		res, err = s.login(ctx, username)
	}
	if err != nil {
		return nil, dbFailed(err, "login username:%s", username)
	}
	return res, nil
}

func (s *UserService) login(ctx context.Context, username string) (*LoginResult, error) {
	return repository.Tx(ctx, s.db, func(tx *repository.Store) (*LoginResult, error) {
		now := time.Now().UnixMilli()
		user, err := tx.User.GetByUsernameForUpdate(ctx, username)
		if err != nil {
			return nil, err
		}
		if user == nil {
			user = &model.UserModel{Username: username, RoleName: username, CreateTime: now, LastLoginTime: now}
			return &LoginResult{User: user, Created: true}, tx.User.Insert(ctx, user)
		}
		user.LastLoginTime = now
		err = tx.User.UpdateLastLoginTime(ctx, user.UserId, now)
		return &LoginResult{User: user}, err
	})
}
