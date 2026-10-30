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
	_, err := r.tx.ExecContext(ctx,
		"INSERT INTO `user` (id, username, roleName, createTime, lastLoginTime, updateTime) VALUES (?, ?, ?, ?, ?, ?)",
		user.UserId, user.Username, user.RoleName, user.CreateTime, user.LastLoginTime, user.UpdateTime)
	return err
}

func (r *UserRepository) UpdateLastLoginTime(ctx context.Context, userId int64, loginTime int64) error {
	_, err := r.tx.ExecContext(ctx, "UPDATE `user` SET lastLoginTime = ?, updateTime = ? WHERE id = ?", loginTime, loginTime, userId)
	return err
}

// todo 先锁玩家行
func (r *UserRepository) LockForUpdate(ctx context.Context, userId int64) (exists bool, err error) {
	var id int64
	err = r.tx.GetContext(ctx, &id, "SELECT id FROM `user` WHERE id = ? FOR UPDATE", userId)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
