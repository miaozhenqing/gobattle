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
		"select * from `user` where username = ? for update", username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Insert(ctx context.Context, user *model.User) error {
	_, err := r.tx.NamedExecContext(ctx,
		"insert into `user` (id, username, roleName, createTime, lastLoginTime, updateTime) values (:id, :username, :roleName, :createTime, :lastLoginTime, :updateTime)",
		user)
	return err
}

func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	_, err := r.tx.NamedExecContext(ctx,
		"update `user` set roleName = :roleName, lastLoginTime = :lastLoginTime, updateTime = :updateTime where id = :id",
		user)
	return err
}

// todo 先锁玩家行
func (r *UserRepository) LockForUpdate(ctx context.Context, userId int64) (exists bool, err error) {
	var id int64
	err = r.tx.GetContext(ctx, &id, "select id from `user` where id = ? for update", userId)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
