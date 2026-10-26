package service

import (
	"context"
	"database/sql"
	"errors"
	"gobattle/btlog"
	"gobattle/game/dao"
	"gobattle/game/db"
	"gobattle/game/model"
	"time"

	"github.com/jmoiron/sqlx"
)

type AccountService struct {
	db *sqlx.DB
}

func NewAccountService(d *sqlx.DB) *AccountService {
	return &AccountService{db: d}
}

func (s *AccountService) Login(ctx context.Context, username string) (acc *model.Account, created bool, err error) {
	acc, created, err = s.login(ctx, username)
	if db.IsDupEntry(err) || db.IsDeadlock(err) {
		btlog.Ins().Warnf("login conflict, retry, username:%v err:%v", username, err)
		acc, created, err = s.login(ctx, username)
	}
	if err != nil {
		return nil, false, dbFailed(err, "login username:%s", username)
	}
	return acc, created, nil
}

func (s *AccountService) login(ctx context.Context, username string) (acc *model.Account, created bool, err error) {
	now := time.Now().UnixMilli()
	err = db.WithTx(ctx, s.db, func(tx *sqlx.Tx) error {
		var qErr error
		acc, qErr = dao.GetAccountByUsernameForUpdate(ctx, tx, username)
		if errors.Is(qErr, sql.ErrNoRows) {
			acc = &model.Account{Username: username, RoleName: username, CreateTime: now, LastLoginTime: now}
			uid, iErr := dao.InsertAccount(ctx, tx, acc)
			if iErr != nil {
				return iErr
			}
			acc.UID = uid
			created = true
			return nil
		}
		if qErr != nil {
			return qErr
		}
		if uErr := dao.UpdateAccountLastLoginTime(ctx, tx, acc.UID, now); uErr != nil {
			return uErr
		}
		acc.LastLoginTime = now
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return acc, created, nil
}
