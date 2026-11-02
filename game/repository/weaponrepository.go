package repository

import (
	"context"
	"gobattle/game/model"
	"strings"

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
	var sb strings.Builder
	//todo 写法优化
	sb.WriteString("insert into `user_item_weapon` (id, userId, itemId, count, level, createTime, updateTime) values ")
	args := make([]any, 0, len(weapons)*7)
	for i, w := range weapons {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?, ?, ?, ?, ?)")
		args = append(args, w.Id, w.UserId, w.ItemId, w.Count, w.Level, w.CreateTime, w.UpdateTime)
	}
	_, err := r.tx.ExecContext(ctx, sb.String(), args...)
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
