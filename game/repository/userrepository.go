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

func (r *UserRepository) GetByUsernameForUpdate(ctx context.Context, username string) (*model.UserModel, error) {
	user := &model.UserModel{}
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

func (r *UserRepository) Insert(ctx context.Context, user *model.UserModel) error {
	res, err := r.tx.ExecContext(ctx,
		"INSERT INTO `user` (username, roleName, createTime, lastLoginTime) VALUES (?, ?, ?, ?)",
		user.Username, user.RoleName, user.CreateTime, user.LastLoginTime)
	if err != nil {
		return err
	}
	uid, err := res.LastInsertId()
	if err != nil {
		return err
	}
	user.UID = uid
	return nil
}

func (r *UserRepository) UpdateLastLoginTime(ctx context.Context, uid int64, loginTime int64) error {
	_, err := r.tx.ExecContext(ctx, "UPDATE `user` SET lastLoginTime = ? WHERE uid = ?", loginTime, uid)
	return err
}
