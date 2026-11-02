package repository

import (
	"context"
	"gobattle/game/model"

	"github.com/jmoiron/sqlx"
)

type WeaponRepository struct {
	tx *sqlx.Tx
}

func (r *WeaponRepository) ListByUserId(ctx context.Context, userId int64) ([]*model.UserWeapon, error) {
	var weapons []*model.UserWeapon
	err := r.tx.SelectContext(ctx, &weapons, "select * from `user_item_weapon` where userId = ?", userId)
	return weapons, err
}

func (r *WeaponRepository) ListByIds(ctx context.Context, userId int64, ids []int64) ([]*model.UserWeapon, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In("select * from `user_item_weapon` where userId = ? and id in (?)", userId, ids)
	if err != nil {
		return nil, err
	}
	var weapons []*model.UserWeapon
	err = r.tx.SelectContext(ctx, &weapons, r.tx.Rebind(query), args...)
	return weapons, err
}

func (r *WeaponRepository) Insert(ctx context.Context, weapons []*model.UserWeapon) error {
	if len(weapons) == 0 {
		return nil
	}

	_, err := r.tx.NamedExecContext(ctx,
		"insert into `user_item_weapon` (id, userId, itemId, count, level, createTime, updateTime) values (:id, :userId, :itemId, :count, :level, :createTime, :updateTime)",
		weapons)
	return err
}

func (r *WeaponRepository) DeleteByIds(ctx context.Context, userId int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In("delete from `user_item_weapon` where userId = ? and id in (?)", userId, ids)
	if err != nil {
		return err
	}
	_, err = r.tx.ExecContext(ctx, r.tx.Rebind(query), args...)
	return err
}
