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
	*Deps
}

func NewUserService(d *Deps) *UserService {
	return &UserService{Deps: d}
}

type LoginResult struct {
	User    *model.UserModel
	Created bool
}

func (s *UserService) Login(ctx context.Context, username string) (*LoginResult, error) {
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
	return repository.Tx(ctx, s.DB, func(tx *repository.Store) (*LoginResult, error) {
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
		err = tx.User.UpdateLastLoginTime(ctx, user.UID, now)
		return &LoginResult{User: user}, err
	})
}
