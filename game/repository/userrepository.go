package repository

import (
	"context"
	"database/sql"
	"errors"
	"gobattle/game/model"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	tx *sqlx.Tx
}

func (r *UserRepository) GetByUsernameForUpdate(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	err := r.tx.GetContext(ctx, user,
		"SELECT * FROM `user` WHERE username = ? FOR UPDATE", username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Insert(ctx context.Context, user *model.User) error {
	res, err := r.tx.ExecContext(ctx,
		"INSERT INTO `user` (username, roleName, createTime, lastLoginTime) VALUES (?, ?, ?, ?)",
		user.Username, user.RoleName, user.CreateTime, user.LastLoginTime)
	if err != nil {
		return err
	}
	userId, err := res.LastInsertId()
	if err != nil {
		return err
	}
	user.UserId = userId
	return nil
}

func (r *UserRepository) UpdateLastLoginTime(ctx context.Context, userId int64, loginTime int64) error {
	_, err := r.tx.ExecContext(ctx, "UPDATE `user` SET lastLoginTime = ? WHERE id = ?", loginTime, userId)
	return err
}

// 锁玩家行并读出道具序号
func (r *UserRepository) GetItemSeqForUpdate(ctx context.Context, userId int64) (itemSeq int64, exists bool, err error) {
	err = r.tx.GetContext(ctx, &itemSeq, "SELECT itemSeq FROM `user` WHERE id = ? FOR UPDATE", userId)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return itemSeq, true, nil
}
